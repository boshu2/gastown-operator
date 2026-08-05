#!/usr/bin/env bash
# Build and deploy BTE promo-api + polecat-agent (promo-finish).
set -euo pipefail
BTE="$(cd "$(dirname "$0")/.." && pwd)"
ROOT="$(cd "${BTE}/.." && pwd)"
ENV_FILE="${BTE}/.env"
cd "$ROOT"

kubectl config use-context docker-desktop >/dev/null

if [[ ! -f "${ENV_FILE}" ]]; then
  echo "ERROR: missing ${BTE}/.env (copy from bte-promo-script/.env.example)"
  exit 1
fi
# shellcheck disable=SC1090
source "${ENV_FILE}"
: "${GENAXIS_API_KEY:?set GENAXIS_API_KEY in bte-promo-script/.env}"

echo "==> Creating genaxis-auth secret"
kubectl -n gastown-system create secret generic genaxis-auth \
  --from-literal=api-key="${GENAXIS_API_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

TAG="${PROMO_API_TAG:-bte-$(date +%Y%m%d%H%M%S)}"
PAGENT_TAG="${POLECAT_AGENT_TAG:-${TAG}}"

CACHE_FLAG=""
if [[ "${DOCKER_NO_CACHE:-}" == "1" ]]; then
  CACHE_FLAG="--no-cache"
fi

echo "==> Building polecat-agent (includes promo-finish) tag=${PAGENT_TAG}"
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
