# Local setup (Docker Desktop Kubernetes)

Run the full MadEye stack on your laptop after cloning this repo:

```
promo-api → Operator → Polecat (Claude Code) → LiteLLM → MadEye
```

## Prerequisites

- [Docker Desktop](https://www.docker.com/products/docker-desktop/) with **Kubernetes enabled**
  - Settings → Kubernetes → Enable Kubernetes → Apply & Restart
- `kubectl` working: `kubectl config use-context docker-desktop && kubectl get nodes`
- MadEye reachable (company VPN / internal DNS)
- MadEye Bearer token + your PocketFM email
- SSH private key with push access to the target git repo

## 1. Clone

```bash
git clone https://github.com/priyanshur01/gastown-operator.git
cd gastown-operator
```

## 2. Configure secrets

```bash
cp deploy/docker-desktop/secret.env.example deploy/docker-desktop/secret.env
```

Edit `deploy/docker-desktop/secret.env`:

| Variable | Example | Purpose |
|----------|---------|---------|
| `MADEYE_API_KEY` | `sk-…` | MadEye Bearer token |
| `MADEYE_USER_EMAIL` | `you@pocketfm.com` | Required MadEye metadata |
| `LITELLM_MASTER_KEY` | `sk-local-litellm` | Key Polecats use toward LiteLLM |
| `GIT_SSH_PRIVATE_KEY_PATH` | `$HOME/.ssh/id_ed25519` | SSH key for clone/push |
| `GIT_REPO` | `git@github.com:org/gastown-static.git` | Repo Polecats work in |
| `GIT_BRANCH` | `main` | Base branch |
| `RIG_NAME` | `local-smoke` | Rig name |
| `POLECAT_NAME` | `madeye-smoke` | Optional smoke Polecat name |

`secret.env` is gitignored — never commit it.

## 3. Deploy the stack

```bash
kubectl config use-context docker-desktop

# Builds local images + installs operator, LiteLLM → MadEye, Rig
./deploy/docker-desktop/apply-full-stack.sh

# Builds and deploys promo-api (HTTP front door)
./deploy/docker-desktop/deploy-promo-api.sh
```

First run can take several minutes (image builds).

## 4. Port-forward promo-api

NodePort is unreliable on Docker Desktop. Keep this running in a **separate terminal**:

```bash
kubectl -n gastown-system port-forward svc/promo-api 30080:8080
```

## 5. Curl examples

### Health

```bash
curl -s http://127.0.0.1:30080/health
```

Expected: `"ready":true` with LiteLLM / Rig / secrets ok.

### Start a promo job

```bash
curl -s -X POST http://127.0.0.1:30080/v1/promo/generate \
  -H 'Content-Type: application/json' \
  -d '{
    "prompt": "write a 5 minutes script for the rise of beatmaster",
    "name": "demo-1"
  }'
```

Response includes `job_name` and `status_url`.

### Job / Polecat status

```bash
curl -s http://127.0.0.1:30080/v1/promo/jobs/demo-1
```

### Watch Claude logs

```bash
kubectl -n gastown-system logs -f -l gastown.io/polecat=demo-1 -c claude
```

### Watch LiteLLM (MadEye calls)

```bash
kubectl -n gastown-system logs -f deploy/litellm | grep -E 'POST|prefill|BadRequest'
```

## Request body (`POST /v1/promo/generate`)

```json
{
  "prompt": "required user request for script generation",
  "name": "optional-job-name",
  "branch": "optional base branch (default main)"
}
```

Async: returns `202` with `job_name`. The operator runs a Polecat against Rig `local-smoke` via LiteLLM → MadEye (`claude-opus-4-8` only).

## Verify pods

```bash
kubectl -n gastown-operator-system get pods
kubectl -n gastown-system get pods
kubectl get rig
kubectl -n gastown-system get polecat
```

## Cleanup

```bash
kubectl -n gastown-system delete polecat --all --ignore-not-found
kubectl delete rig --all --ignore-not-found
kubectl -n gastown-system delete deploy,svc -l app=promo-api --ignore-not-found
kubectl delete -f deploy/litellm/k8s-docker-desktop.yaml --ignore-not-found
make undeploy
make uninstall
```

## More detail

- LiteLLM / MadEye adapter: [docs/LITE_LLM.md](../../docs/LITE_LLM.md)
- Promo API notes: [PROMO_API.md](PROMO_API.md)
