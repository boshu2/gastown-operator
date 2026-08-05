#!/usr/bin/env bash
# Deploy Gas Town stack + BTE Rig on Docker Desktop Kubernetes:
#   operator + LiteLLM → MadEye + Rig (Polecats created by promo-api)
set -euo pipefail

BTE="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "${BTE}/.." && pwd)"
LITELLM_DIR="${ROOT}/deploy/litellm"
ENV_FILE="${BTE}/.env"
DEFAULTS_GO="${BTE}/api/defaults.go"
IMG="${IMG:-gastown-operator:dev}"
CONTEXT="${KUBE_CONTEXT:-docker-desktop}"
NS_WORKLOAD="gastown-system"

go_default() {
  local name="$1"
  sed -n "s/.*${name}[[:space:]]*=[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "${DEFAULTS_GO}" | head -1
}
GIT_REPO="$(go_default DefaultGitRepo)"
GIT_BRANCH="$(go_default DefaultGitBranch)"
RIG_NAME="$(go_default DefaultRigName)"
: "${GIT_REPO:?could not read DefaultGitRepo from ${DEFAULTS_GO}}"
: "${GIT_BRANCH:?could not read DefaultGitBranch from ${DEFAULTS_GO}}"
: "${RIG_NAME:?could not read DefaultRigName from ${DEFAULTS_GO}}"

cd "${ROOT}"

echo "==> Checking Docker Desktop Kubernetes (${CONTEXT})"
kubectl config use-context "${CONTEXT}" >/dev/null
kubectl get nodes >/dev/null

if [[ ! -f "${ENV_FILE}" ]]; then
  cp "${BTE}/.env.example" "${BTE}/.env"
  echo "Created ${BTE}/.env — edit secrets, then re-run."
  exit 1
fi

# shellcheck disable=SC1090
source "${ENV_FILE}"

: "${MADEYE_API_KEY:?set MADEYE_API_KEY in bte-promo-script/.env}"
: "${MADEYE_USER_EMAIL:?set MADEYE_USER_EMAIL in bte-promo-script/.env}"
: "${GIT_SSH_PRIVATE_KEY_PATH:?set GIT_SSH_PRIVATE_KEY_PATH in bte-promo-script/.env}"
LITELLM_MASTER_KEY="${LITELLM_MASTER_KEY:-sk-local-litellm}"

if [[ ! -f "${GIT_SSH_PRIVATE_KEY_PATH}" ]]; then
  echo "ERROR: SSH key not found: ${GIT_SSH_PRIVATE_KEY_PATH}"
  exit 1
fi

echo "==> Building operator image ${IMG}"
make docker-build-e2e IMG="${IMG}"

echo "==> Installing CRDs"
make install

echo "==> Deploying operator"
make deploy IMG="${IMG}"

OP_NS="gastown-operator-system"
kubectl -n "${OP_NS}" set image deploy/gastown-operator-controller-manager manager="${IMG}"
kubectl -n "${OP_NS}" patch deploy gastown-operator-controller-manager --type=json \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"Never"}]' \
  >/dev/null || true
kubectl -n "${OP_NS}" rollout status deploy/gastown-operator-controller-manager --timeout=180s

echo "==> Ensuring workload namespace ${NS_WORKLOAD}"
kubectl get ns "${NS_WORKLOAD}" >/dev/null 2>&1 || kubectl create ns "${NS_WORKLOAD}"

echo "==> Deploying LiteLLM (MadEye adapter)"
kubectl -n "${NS_WORKLOAD}" create secret generic litellm-secrets \
  --from-literal=MADEYE_API_KEY="${MADEYE_API_KEY}" \
  --from-literal=MADEYE_USER_EMAIL="${MADEYE_USER_EMAIL}" \
  --from-literal=LITELLM_MASTER_KEY="${LITELLM_MASTER_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${NS_WORKLOAD}" create configmap litellm-callbacks \
  --from-file=strip_prefill.py="${LITELLM_DIR}/strip_prefill.py" \
  --from-file=prefill_proxy.py="${LITELLM_DIR}/prefill_proxy.py" \
  --from-file=k8s-entrypoint.sh="${LITELLM_DIR}/k8s-entrypoint.sh" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl apply -f "${LITELLM_DIR}/k8s-docker-desktop.yaml"
kubectl -n "${NS_WORKLOAD}" rollout status deploy/litellm --timeout=180s

echo "==> Creating git + LiteLLM auth secrets"
kubectl -n "${NS_WORKLOAD}" create secret generic git-creds \
  --from-file=ssh-privatekey="${GIT_SSH_PRIVATE_KEY_PATH}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${NS_WORKLOAD}" create secret generic litellm-auth \
  --from-literal=master-key="${LITELLM_MASTER_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "==> Applying Rig ${RIG_NAME}"
TMP_RIG="$(mktemp)"
sed \
  -e "s|GIT_REPO_PLACEHOLDER|${GIT_REPO}|g" \
  -e "s|GIT_BRANCH_PLACEHOLDER|${GIT_BRANCH}|g" \
  -e "s|name: promo-script-tool|name: ${RIG_NAME}|g" \
  "${BTE}/deploy/rig.yaml.tpl" > "${TMP_RIG}"
kubectl apply -f "${TMP_RIG}"
rm -f "${TMP_RIG}"

echo
echo "Full stack ready (Rig: ${RIG_NAME}). Next: ./bte-promo-script/deploy/deploy.sh"
