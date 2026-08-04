# Full Gas Town on Docker Desktop Kubernetes

Deploys:
1. Operator (built from this repo — includes LiteLLM/`agentConfig` wiring)
2. LiteLLM → MadEye adapter
3. Sample Rig + smoke Polecat

## Prerequisites

- Docker Desktop running with **Kubernetes enabled**
- `kubectl config use-context docker-desktop` works
- MadEye reachable (VPN) + Bearer token
- SSH private key with access to a git repo

## Steps

```bash
kubectl config use-context docker-desktop
kubectl get nodes

cd deploy/docker-desktop
cp secret.env.example secret.env
```

Edit `secret.env`:

| Variable | Purpose |
|----------|---------|
| `MADEYE_API_KEY` | MadEye Bearer token |
| `MADEYE_USER_EMAIL` | Your PocketFM email |
| `GIT_SSH_PRIVATE_KEY_PATH` | Path to SSH key |
| `GIT_REPO` | e.g. `git@github.com:org/repo.git` |

Then:

```bash
./apply-full-stack.sh
```

First run builds the operator image (needs network; several minutes).

## Verify

```bash
kubectl -n gastown-operator-system get pods
kubectl -n gastown-system get pods
kubectl get rig
kubectl -n gastown-system get polecat

# LiteLLM
curl -s http://localhost:30040/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-local-litellm" \
  -H "anthropic-version: 2023-06-01" \
  -d '{"model":"claude-opus-4-8","max_tokens":32,"messages":[{"role":"user","content":"ping"}]}'

# Polecat agent logs
kubectl -n gastown-system logs -f -l gastown.io/polecat=madeye-smoke -c claude
```

## Cleanup

```bash
kubectl -n gastown-system delete polecat --all --ignore-not-found
kubectl delete rig --all --ignore-not-found
kubectl -n gastown-system delete deploy,svc,cm,secret -l app=litellm --ignore-not-found
kubectl delete -f ../litellm/k8s-docker-desktop.yaml --ignore-not-found
make undeploy
make uninstall
```
