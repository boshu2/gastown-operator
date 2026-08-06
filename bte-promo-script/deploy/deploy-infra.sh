#!/usr/bin/env bash
# Deploy base infra on Docker Desktop Kubernetes:
#   gastown-operator + madeye-proxy + Rig (polecat pods come from deploy-promo.sh)
set -euo pipefail

BTE="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "${BTE}/.." && pwd)"
DEFAULTS_GO="${BTE}/api/defaults.go"
IMG="${IMG:-gastown-operator:bte-$(date +%Y%m%d%H%M%S)}"
CONTEXT="${KUBE_CONTEXT:-docker-desktop}"
NS_WORKLOAD="gastown-system"

# shellcheck source=lib/require-k8s-secrets.sh
source "${BTE}/deploy/lib/require-k8s-secrets.sh"

go_default() {
  local name="$1"
  sed -n "s/.*${name}[[:space:]]*=[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "${DEFAULTS_GO}" | head -1
}
GIT_REPO="$(go_default DefaultRigGitURL)"
RIG_NAME="$(go_default DefaultRigName)"
PROXY_MASTER_KEY="$(go_default DefaultProxyMasterKey)"
MADEYE_USER_EMAIL="$(go_default DefaultMadEyeUserEmail)"
: "${GIT_REPO:?could not read DefaultRigGitURL from ${DEFAULTS_GO}}"
: "${RIG_NAME:?could not read DefaultRigName from ${DEFAULTS_GO}}"
: "${PROXY_MASTER_KEY:?could not read DefaultProxyMasterKey from ${DEFAULTS_GO}}"
: "${MADEYE_USER_EMAIL:?could not read DefaultMadEyeUserEmail from ${DEFAULTS_GO}}"

cd "${ROOT}"

echo "==> Checking Docker Desktop Kubernetes (${CONTEXT})"
kubectl config use-context "${CONTEXT}" >/dev/null
kubectl get nodes >/dev/null

echo "==> Checking K8s secrets (MADEYE_API_KEY, GENAXIS_API_KEY injected via secretKeyRef)"
require_promo_secrets "${NS_WORKLOAD}"

echo "==> Building operator image ${IMG}"
if [[ "${DOCKER_NO_CACHE:-}" == "1" ]]; then
  docker build --no-cache -t "${IMG}" -f "${ROOT}/Dockerfile" "${ROOT}"
else
  make docker-build-e2e IMG="${IMG}"
fi

echo "==> Installing CRDs"
make install

echo "==> Deploying operator"
make deploy IMG="${IMG}"

OP_NS="gastown-operator-system"
kubectl -n "${OP_NS}" set image deploy/gastown-operator-controller-manager manager="${IMG}"
kubectl -n "${OP_NS}" patch deploy gastown-operator-controller-manager --type=json \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"IfNotPresent"}]' \
  >/dev/null || true
kubectl -n "${OP_NS}" rollout status deploy/gastown-operator-controller-manager --timeout=180s

echo "==> Ensuring workload namespace ${NS_WORKLOAD}"
kubectl get ns "${NS_WORKLOAD}" >/dev/null 2>&1 || kubectl create ns "${NS_WORKLOAD}"

PROXY_TAG="${MADEYE_PROXY_TAG:-bte-$(date +%Y%m%d%H%M%S)}"
echo "==> Building madeye-proxy tag=${PROXY_TAG}"
docker build -f "${BTE}/madeye-proxy/Dockerfile" -t "madeye-proxy:${PROXY_TAG}" -t madeye-proxy:local .

echo "==> Deploying madeye-proxy (Anthropic → OpenAI → MadEye)"
kubectl apply -f "${BTE}/deploy/madeye-proxy.yaml"
kubectl -n "${NS_WORKLOAD}" set env deploy/madeye-proxy \
  PROXY_MASTER_KEY="${PROXY_MASTER_KEY}" \
  MADEYE_USER_EMAIL="${MADEYE_USER_EMAIL}"
kubectl -n "${NS_WORKLOAD}" set image deploy/madeye-proxy madeye-proxy="madeye-proxy:${PROXY_TAG}"
kubectl -n "${NS_WORKLOAD}" rollout status deploy/madeye-proxy --timeout=180s

echo "==> Applying Rig ${RIG_NAME}"
TMP_RIG="$(mktemp)"
sed \
  -e "s|GIT_REPO_PLACEHOLDER|${GIT_REPO}|g" \
  -e "s|name: promo-script-tool|name: ${RIG_NAME}|g" \
  "${BTE}/deploy/rig.yaml.tpl" > "${TMP_RIG}"
kubectl apply -f "${TMP_RIG}"
rm -f "${TMP_RIG}"

echo
echo "Infra ready (Rig: ${RIG_NAME}). Next: ./bte-promo-script/deploy/deploy-promo.sh"
