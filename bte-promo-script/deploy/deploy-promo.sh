#!/usr/bin/env bash
# Build and deploy promo-api + polecat-agent (includes tools/finish → promo-finish).
set -euo pipefail
BTE="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "${BTE}/.." && pwd)"
cd "$ROOT"

# shellcheck source=lib/require-k8s-secrets.sh
source "${BTE}/deploy/lib/require-k8s-secrets.sh"

kubectl config use-context docker-desktop >/dev/null

echo "==> Checking K8s secrets (GENAXIS_API_KEY injected via secretKeyRef on promo-api)"
require_promo_secrets gastown-system

TAG="${PROMO_API_TAG:-bte-$(date +%Y%m%d%H%M%S)}"
PAGENT_TAG="${POLECAT_AGENT_TAG:-${TAG}}"

CACHE_FLAG=""
if [[ "${DOCKER_NO_CACHE:-}" == "1" ]]; then
  CACHE_FLAG="--no-cache"
fi

echo "==> Building polecat-agent (includes tools/finish) tag=${PAGENT_TAG}"
docker build ${CACHE_FLAG} -f images/polecat-agent/Dockerfile -t "polecat-agent:${PAGENT_TAG}" -t polecat-agent:local .

echo "==> Building promo-api tag=${TAG}"
docker buildx build ${CACHE_FLAG} --load -t "promo-api:${TAG}" -t promo-api:local -f "${BTE}/api/Dockerfile" .

echo "==> Applying manifest"
kubectl apply -f "${BTE}/deploy/bte-promo-script.yaml"

kubectl -n gastown-system set image deploy/promo-api promo-api="promo-api:${TAG}"
kubectl -n gastown-system set env deploy/promo-api AGENT_IMAGE="polecat-agent:${PAGENT_TAG}"
kubectl -n gastown-system set env deploy/promo-api GIT_REPO- GIT_BRANCH- WEBHOOK_URL- RIG_NAME- >/dev/null

echo "==> Ensuring Rig"
DEFAULTS_GO="${BTE}/api/defaults.go"
go_default() { sed -n "s/.*${1}[[:space:]]*=[[:space:]]*\"\\([^\"]*\\)\".*/\\1/p" "${DEFAULTS_GO}" | head -1; }
RIG_NAME="$(go_default DefaultRigName)"
GIT_REPO="$(go_default DefaultRigGitURL)"
TMP_RIG="$(mktemp)"
sed \
  -e "s|GIT_REPO_PLACEHOLDER|${GIT_REPO}|g" \
  -e "s|name: promo-script-tool|name: ${RIG_NAME}|g" \
  "${BTE}/deploy/rig.yaml.tpl" > "${TMP_RIG}"
kubectl apply -f "${TMP_RIG}"
rm -f "${TMP_RIG}"

kubectl -n gastown-system scale deploy/promo-api --replicas=1
kubectl -n gastown-system delete pod -l app=promo-api --ignore-not-found --wait=false
kubectl -n gastown-system rollout status deploy/promo-api --timeout=180s

echo
echo "promo-api ready"
if grep -q 'DefaultDevOutputHostPath = "/Users' "${BTE}/api/defaults.go" 2>/dev/null; then
  DEV_OUT="$(sed -n 's/.*DefaultDevOutputHostPath = "\([^"]*\)".*/\1/p' "${BTE}/api/defaults.go" | head -1)"
  if [[ -n "${DEV_OUT}" ]]; then
    mkdir -p "${DEV_OUT}"
    chmod 777 "${DEV_OUT}" 2>/dev/null || true
    echo "NOTE (LOCAL TEST ONLY): W4 script saves via HTTP dev-save (hostPath does not sync on Docker Desktop Mac)."
    echo "  1. Run: python3 ${BTE}/deploy/dev-webhook-catcher.py"
    echo "  2. Set: kubectl -n gastown-system set env deploy/promo-api GENAXIS_WEBHOOK_URL='http://host.docker.internal:9999/v1/bte/internal/webhook'"
    echo "  3. Scripts land in ${DEV_OUT}/<filename>.md"
  fi
fi
