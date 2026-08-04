# Promo API

Small HTTP front door for Gastown promo-script jobs on Docker Desktop.

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `GET` | `/health` or `/healthz` | Stack health (Rig + secrets + LiteLLM) |
| `POST` | `/v1/promo/generate` | Start promo script generation (creates a Polecat) |
| `GET` | `/v1/promo/jobs/{name}` | Job / Polecat status |

## Deploy

```bash
cd deploy/docker-desktop
./deploy-promo-api.sh
```

Service is exposed on **NodePort 30080**.

## Examples

```bash
# Health
curl -s http://localhost:30080/health | jq

# Invoke promo script generation against gastown-static Rig
curl -s -X POST http://localhost:30080/v1/promo/generate \
  -H 'Content-Type: application/json' \
  -d '{
    "prompt": "Create a 30-second promo script for a thriller podcast launch",
    "name": "thriller-promo-1"
  }' | jq

# Poll status
curl -s http://localhost:30080/v1/promo/jobs/thriller-promo-1 | jq

# Watch agent logs
kubectl -n gastown-system logs -f -l gastown.io/polecat=thriller-promo-1 -c claude
```

## Request body (`POST /v1/promo/generate`)

```json
{
  "prompt": "required user request for script generation",
  "name": "optional-job-name",
  "branch": "optional base branch (default main)"
}
```

Async: returns `202` with `job_name` + `status_url`. The operator runs the Polecat against Rig `local-smoke` (`gastown-static`) via LiteLLM → MadEye.
