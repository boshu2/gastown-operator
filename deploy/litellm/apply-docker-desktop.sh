#!/usr/bin/env bash
# Deploy LiteLLM to Docker Desktop Kubernetes.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")" && pwd)"
CONTEXT="${KUBE_CONTEXT:-docker-desktop}"
SECRET_ENV="${ROOT}/secret.env"

echo "==> Checking kubectl context: ${CONTEXT}"
if ! kubectl config get-contexts "${CONTEXT}" >/dev/null 2>&1; then
  cat <<EOF
ERROR: kubectl context '${CONTEXT}' not found.

Enable Kubernetes in Docker Desktop:
  Docker Desktop → Settings → Kubernetes → Enable Kubernetes → Apply & Restart

Then:
  kubectl config use-context docker-desktop
  kubectl get nodes
EOF
  exit 1
fi

kubectl config use-context "${CONTEXT}" >/dev/null
if ! kubectl get nodes >/dev/null 2>&1; then
  echo "ERROR: cannot reach cluster '${CONTEXT}'. Is Docker Desktop Kubernetes running?"
  exit 1
fi

if [[ ! -f "${SECRET_ENV}" ]]; then
  cp "${ROOT}/secret.env.example" "${SECRET_ENV}"
  echo "Created ${SECRET_ENV} — edit MADEYE_API_KEY, then re-run this script."
  exit 1
fi

# shellcheck disable=SC1090
source "${SECRET_ENV}"

: "${MADEYE_API_KEY:?set MADEYE_API_KEY in secret.env}"
: "${MADEYE_USER_EMAIL:?set MADEYE_USER_EMAIL in secret.env}"
LITELLM_MASTER_KEY="${LITELLM_MASTER_KEY:-sk-local-litellm}"

echo "==> Ensuring namespace gastown-system"
kubectl get ns gastown-system >/dev/null 2>&1 || kubectl create ns gastown-system

echo "==> Creating/updating Secret litellm-secrets"
kubectl -n gastown-system create secret generic litellm-secrets \
  --from-literal=MADEYE_API_KEY="${MADEYE_API_KEY}" \
  --from-literal=MADEYE_USER_EMAIL="${MADEYE_USER_EMAIL}" \
  --from-literal=LITELLM_MASTER_KEY="${LITELLM_MASTER_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "==> Creating/updating ConfigMap litellm-callbacks (strip MadEye-incompatible prefills)"
kubectl -n gastown-system create configmap litellm-callbacks \
  --from-file=strip_prefill.py="${ROOT}/strip_prefill.py" \
  --from-file=prefill_proxy.py="${ROOT}/prefill_proxy.py" \
  --from-file=k8s-entrypoint.sh="${ROOT}/k8s-entrypoint.sh" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "==> Applying LiteLLM manifests"
kubectl apply -f "${ROOT}/k8s-docker-desktop.yaml"

echo "==> Waiting for LiteLLM rollout"
kubectl -n gastown-system rollout status deploy/litellm --timeout=180s

NODE_PORT="$(kubectl -n gastown-system get svc litellm -o jsonpath='{.spec.ports[0].nodePort}')"
echo
echo "LiteLLM is up on Docker Desktop Kubernetes."
echo "  In-cluster:  http://litellm.gastown-system.svc:4000"
echo "  From host:   http://localhost:${NODE_PORT}"
echo
echo "Smoke test:"
echo "  curl -s http://localhost:${NODE_PORT}/v1/messages \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Authorization: Bearer ${LITELLM_MASTER_KEY}' \\"
echo "    -H 'anthropic-version: 2023-06-01' \\"
echo "    -d '{\"model\":\"claude-opus-4-8\",\"max_tokens\":64,\"messages\":[{\"role\":\"user\",\"content\":\"Hello! which model are you using?\"}]}'"
