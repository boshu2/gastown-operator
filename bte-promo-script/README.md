# BTE Promo Script

PocketFM BTE promo pipeline — API, finish tool, deploy manifests, and config.

## Layout

```
bte-promo-script/
  .env.example        local defaults (email, LiteLLM key) — committed
  .env                infra secrets only (gitignored)
  api/                promo-api HTTP service
  finish/             promo-finish CLI (runs on Polecat)
  repos/
    bte-promo-script-repo/   promo pipeline tool (workflows, canon, PROMO_FINISH)
  deploy/
    bte-promo-script.yaml   K8s Deployment + Service + RBAC
    rig.yaml.tpl            Rig CR template
    apply-full-stack.sh     operator + LiteLLM + Rig (local Docker Desktop)
    deploy.sh               promo-api + polecat-agent images
  README.md
```

## Flow

1. GenAxis → `POST /v1/promo/generate` → Polecat + `PROMO_SCRIPT_STARTED`
2. Polecat uses a **static pipeline** at `/workspace/promo-tool` (baked into `polecat-agent`, no git clone); Claude runs W2→W3→W4
3. `promo-finish finish map=… briefs=… script=… receipt=…` → `PROMO_SCRIPT_COMPLETED`

Finish instructions: `repos/bte-promo-script-repo/scripts/PROMO_FINISH.md`

## Config

| Type | Location |
|------|----------|
| Infra secrets | `bte-promo-script/.env` (gitignored) |
| Local defaults | `bte-promo-script/.env.example` (email, LiteLLM key) |
| Non-secret | `api/defaults.go` (rig, workspace path, GenAxis webhook URL) |

**`.env` (infra injects):**

| Variable | Purpose |
|----------|---------|
| `GENAXIS_API_KEY` | GenAxis webhook `X-API-Key` |
| `MADEYE_API_KEY` | MadEye Bearer token (LiteLLM) |

**`.env.example` (committed defaults):**

| Variable | Purpose |
|----------|---------|
| `MADEYE_USER_EMAIL` | MadEye metadata |
| `LITELLM_MASTER_KEY` | Polecats → LiteLLM auth |

**`defaults.go` constants:**

| Constant | Default |
|----------|---------|
| GenAxis webhook | `https://gen-axis.pocketfm.com/v1/bte/internal/webhook` |
| Promo workspace | `/workspace/promo-tool` (copied from image at pod start) |
| Rig | `promo-script-tool` |

Upload/S3: stub — logs paths, returns hardcoded URLs until real S3 (`api/s3.go`).

## Deploy

```bash
cp bte-promo-script/.env.example bte-promo-script/.env
# edit .env with real secrets

./bte-promo-script/deploy/apply-full-stack.sh   # once: operator + LiteLLM + Rig
./bte-promo-script/deploy/deploy.sh             # promo-api + polecat-agent
kubectl -n gastown-system port-forward svc/promo-api 30080:8080
```

**Infra production:**

1. Build/push `promo-api` and `polecat-agent` images (`promo-finish` + pipeline repo must be in agent)
2. Inject secrets from `.env` → K8s secrets (`genaxis-auth`, `litellm-auth`)
3. Apply `deploy/bte-promo-script.yaml` (set image tags)
4. Apply Rig from `deploy/rig.yaml.tpl`
5. Smoke: `POST /v1/promo/generate` → STARTED → Claude → `promo-finish` → COMPLETED

## Endpoints

| Method | Path | Purpose |
|--------|------|---------|
| `POST` | `/v1/promo/generate` | Start job |
| `GET` | `/v1/promo/jobs/{name}` | Job status |
| `POST` | `/v1/promo/jobs/{name}/upload` | Upload files → `artifacts[]` |
| `POST` | `/v1/promo/jobs/{name}/webhook` | GenAxis completed webhook |
| `POST` | `/v1/promo/jobs/{name}/complete` | Upload + webhook |
