#!/usr/bin/env bash
# Create/update K8s secrets for promo stack. Pods inject these via secretKeyRef
# (see deploy/madeye-proxy.yaml and deploy/bte-promo-script.yaml).
#
# Usage:
#   cp secrets.env.example secrets.env   # fill in keys once
#   ./bte-promo-script/deploy/apply-secrets.sh
#
# Or: SECRETS_FILE=/path/to/env ./apply-secrets.sh
set -euo pipefail

BTE="$(cd "$(dirname "$0")/.." && pwd)"
NS="${NAMESPACE:-gastown-system}"
CONTEXT="${KUBE_CONTEXT:-docker-desktop}"

SECRETS_FILE="${SECRETS_FILE:-${BTE}/secrets.env}"
if [[ ! -f "${SECRETS_FILE}" && -f "${BTE}/.env" ]]; then
  SECRETS_FILE="${BTE}/.env"
fi

if [[ ! -f "${SECRETS_FILE}" ]]; then
  echo "ERROR: missing secrets file."
  echo "  cp ${BTE}/secrets.env.example ${BTE}/secrets.env"
  echo "  edit GENAXIS_API_KEY and MADEYE_API_KEY, then re-run."
  exit 1
fi

# shellcheck disable=SC1090
source "${SECRETS_FILE}"

: "${GENAXIS_API_KEY:?set GENAXIS_API_KEY in ${SECRETS_FILE}}"
: "${MADEYE_API_KEY:?set MADEYE_API_KEY in ${SECRETS_FILE}}"

kubectl config use-context "${CONTEXT}" >/dev/null
kubectl get ns "${NS}" >/dev/null 2>&1 || kubectl create ns "${NS}"

echo "==> Applying K8s secrets in namespace ${NS} (from ${SECRETS_FILE})"
echo "    PROXY_MASTER_KEY and MADEYE_USER_EMAIL come from api/defaults.go — not K8s."

kubectl -n "${NS}" create secret generic madeye-proxy-secrets \
  --from-literal=MADEYE_API_KEY="${MADEYE_API_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${NS}" create secret generic genaxis-auth \
  --from-literal=api-key="${GENAXIS_API_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "Secrets ready:"
echo "  madeye-proxy-secrets  → MADEYE_API_KEY (madeye-proxy pod)"
echo "  genaxis-auth          → api-key (promo-api → GenAxis webhooks)"
