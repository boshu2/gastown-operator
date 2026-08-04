#!/usr/bin/env bash
# Deploy full Gas Town stack on Docker Desktop Kubernetes:
#   operator (local build with MadEye agentConfig wiring)
#   + LiteLLM → MadEye
#   + Rig + smoke Polecat
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LITELLM_DIR="${ROOT}/deploy/litellm"
SECRET_ENV="$(cd "$(dirname "$0")" && pwd)/secret.env"
IMG="${IMG:-gastown-operator:dev}"
CONTEXT="${KUBE_CONTEXT:-docker-desktop}"
NS_WORKLOAD="gastown-system"

cd "${ROOT}"

echo "==> Checking Docker Desktop Kubernetes (${CONTEXT})"
kubectl config use-context "${CONTEXT}" >/dev/null
kubectl get nodes >/dev/null

if [[ ! -f "${SECRET_ENV}" ]]; then
  cp "$(dirname "$0")/secret.env.example" "${SECRET_ENV}"
  echo "Created ${SECRET_ENV}"
  echo "Edit MADEYE_API_KEY, GIT_SSH_PRIVATE_KEY_PATH, and GIT_REPO, then re-run."
  exit 1
fi

# shellcheck disable=SC1090
source "${SECRET_ENV}"

: "${MADEYE_API_KEY:?set MADEYE_API_KEY in secret.env}"
: "${MADEYE_USER_EMAIL:?set MADEYE_USER_EMAIL in secret.env}"
: "${GIT_SSH_PRIVATE_KEY_PATH:?set GIT_SSH_PRIVATE_KEY_PATH in secret.env}"
: "${GIT_REPO:?set GIT_REPO in secret.env}"
LITELLM_MASTER_KEY="${LITELLM_MASTER_KEY:-sk-local-litellm}"
GIT_BRANCH="${GIT_BRANCH:-main}"
RIG_NAME="${RIG_NAME:-local-smoke}"
POLECAT_NAME="${POLECAT_NAME:-madeye-smoke}"

if [[ ! -f "${GIT_SSH_PRIVATE_KEY_PATH}" ]]; then
  echo "ERROR: SSH key not found: ${GIT_SSH_PRIVATE_KEY_PATH}"
  exit 1
fi

echo "==> Building operator image ${IMG} (includes agentConfig → LiteLLM wiring)"
# docker-build-e2e tags the exact IMG reference (no VERSION suffix)
make docker-build-e2e IMG="${IMG}"

echo "==> Installing CRDs"
make install

echo "==> Deploying operator"
make deploy IMG="${IMG}"

OP_NS="gastown-operator-system"
echo "==> Forcing imagePullPolicy=Never for local Docker Desktop image"
kubectl -n "${OP_NS}" set image deploy/gastown-operator-controller-manager manager="${IMG}"
kubectl -n "${OP_NS}" patch deploy gastown-operator-controller-manager --type=json \
  -p='[{"op":"replace","path":"/spec/template/spec/containers/0/imagePullPolicy","value":"Never"}]' \
  >/dev/null || true
kubectl -n "${OP_NS}" rollout status deploy/gastown-operator-controller-manager --timeout=180s

echo "==> Ensuring workload namespace ${NS_WORKLOAD}"
kubectl get ns "${NS_WORKLOAD}" >/dev/null 2>&1 || kubectl create ns "${NS_WORKLOAD}"

echo "==> Deploying LiteLLM (MadEye adapter)"
# Reuse litellm docker-desktop manifests + secrets from secret.env
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

# Render user email into litellm config via apply script assets
kubectl apply -f "${LITELLM_DIR}/k8s-docker-desktop.yaml"
# Patch ConfigMap email if needed — k8s-docker-desktop uses ${MADEYE_USER_EMAIL} placeholder in template;
# deployment entrypoint sed-substitutes from env. Good.

kubectl -n "${NS_WORKLOAD}" rollout status deploy/litellm --timeout=180s

echo "==> Creating git + LiteLLM auth secrets for Polecats"
kubectl -n "${NS_WORKLOAD}" create secret generic git-creds \
  --from-file=ssh-privatekey="${GIT_SSH_PRIVATE_KEY_PATH}" \
  --dry-run=client -o yaml | kubectl apply -f -

kubectl -n "${NS_WORKLOAD}" create secret generic litellm-auth \
  --from-literal=master-key="${LITELLM_MASTER_KEY}" \
  --dry-run=client -o yaml | kubectl apply -f -

echo "==> Applying Rig + smoke Polecat"
TMP_SAMPLES="$(mktemp)"
sed \
  -e "s|GIT_REPO_PLACEHOLDER|${GIT_REPO}|g" \
  -e "s|GIT_BRANCH_PLACEHOLDER|${GIT_BRANCH}|g" \
  -e "s|name: local-smoke|name: ${RIG_NAME}|g" \
  -e "s|rig: local-smoke|rig: ${RIG_NAME}|g" \
  -e "s|name: madeye-smoke|name: ${POLECAT_NAME}|g" \
  "$(dirname "$0")/samples.yaml.tpl" > "${TMP_SAMPLES}"
kubectl apply -f "${TMP_SAMPLES}"
rm -f "${TMP_SAMPLES}"

echo
echo "============================================"
echo "  Gas Town full stack is up (Docker Desktop)"
echo "============================================"
echo
echo "Operator:  kubectl -n ${OP_NS} get pods"
echo "LiteLLM:   kubectl -n ${NS_WORKLOAD} get pods"
echo "           curl http://localhost:30040/v1/messages ..."
echo "Rig:       kubectl get rig ${RIG_NAME}"
echo "Polecat:   kubectl -n ${NS_WORKLOAD} get polecat ${POLECAT_NAME} -w"
echo "Logs:      kubectl -n ${NS_WORKLOAD} logs -f -l gastown.io/polecat=${POLECAT_NAME} -c claude"
echo
echo "LiteLLM smoke:"
echo "  curl -s http://localhost:30040/v1/messages \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -H 'Authorization: Bearer ${LITELLM_MASTER_KEY}' \\"
echo "    -H 'anthropic-version: 2023-06-01' \\"
echo "    -d '{\"model\":\"claude-opus-4-8\",\"max_tokens\":64,\"messages\":[{\"role\":\"user\",\"content\":\"ping\"}]}'"
