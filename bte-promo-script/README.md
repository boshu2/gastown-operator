# BTE Promo Script

PocketFM BTE promo pipeline — API, finish tool, deploy manifests, and config.

**Infrastructure & flow:** [INFRA-README.md](INFRA-README.md)

## Layout

```
bte-promo-script/
  secrets.env.example   template for K8s secrets (apply once)
  secrets.env           gitignored — GENAXIS + MADEYE keys
  api/                promo-api HTTP service
  madeye-proxy/       Anthropic → OpenAI proxy to MadEye
  tools/
    finish/           promo-finish CLI (runs on Polecat)
  repos/
    bte-promo-script-repo/   promo pipeline tool (workflows, canon, PROMO_FINISH)
  deploy/
    bte-promo-script.yaml   K8s Deployment + Service + RBAC
    madeye-proxy.yaml       madeye-proxy Deployment + Service
    rig.yaml.tpl            Rig CR template
    apply-secrets.sh          K8s secrets from secrets.env (run once)
    deploy-infra.sh         operator + madeye-proxy + Rig (run once)
    deploy-promo.sh         promo-api + polecat-agent images
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
| Infra secrets | K8s secrets (`genaxis-auth`, `madeye-proxy-secrets`) via `apply-secrets.sh` |
| Non-secret constants | `api/defaults.go` (rig, proxy URL, MadEye email, proxy key) |

**K8s secrets (injected into pods via `secretKeyRef`):**

| Secret | Key | Pod env | Purpose |
|--------|-----|---------|---------|
| `genaxis-auth` | `api-key` | `GENAXIS_API_KEY` | promo-api → GenAxis |
| `madeye-proxy-secrets` | `MADEYE_API_KEY` | `MADEYE_API_KEY` | madeye-proxy → MadEye |

**`defaults.go` constants (committed, not in `.env`):**

| Constant | Purpose |
|----------|---------|
| `DefaultMadEyeUserEmail` | MadEye metadata |
| `DefaultProxyMasterKey` | Polecats → madeye-proxy auth |
| `DefaultMadEyeProxyURL` | In-cluster proxy URL |

| Constant | Default |
|----------|---------|
| GenAxis webhook | `https://gen-axis.pocketfm.com/v1/bte/internal/webhook` |
| Promo workspace | `/workspace/promo-tool` (copied from image at pod start) |
| Rig | `promo-script-tool` |

Upload/S3: stub — logs paths, returns hardcoded URLs until real S3 (`api/s3.go`).

## Deploy

```bash
cp bte-promo-script/secrets.env.example bte-promo-script/secrets.env
# edit GENAXIS_API_KEY and MADEYE_API_KEY

./bte-promo-script/deploy/apply-secrets.sh    # once → K8s secrets
./bte-promo-script/deploy/deploy-infra.sh     # operator + madeye-proxy + Rig
./bte-promo-script/deploy/deploy-promo.sh     # promo-api + polecat-agent
kubectl -n gastown-system port-forward svc/promo-api 30080:8080
```

**Infra production:**

1. Build/push `promo-api` and `polecat-agent` images (`promo-finish` + pipeline repo must be in agent)
2. Create K8s secrets (`genaxis-auth`, `madeye-proxy-secrets`) via Vault/CI or `apply-secrets.sh` — only `MADEYE_API_KEY` and `GENAXIS_API_KEY`; proxy master key and email come from `api/defaults.go`
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
