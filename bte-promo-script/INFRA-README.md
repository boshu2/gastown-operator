# BTE Promo Script — Infrastructure & Flow

How a GenAxis request becomes a promo script: who calls whom and where data moves.

---

## Flow

```
GenAxis
  │  POST /v1/promo/generate
  ▼
promo-api (:8080, NodePort 30080)
  │  creates Polecat CR
  │  webhook → GenAxis  PROMO_SCRIPT_STARTED
  ▼
gastown-operator → polecat pod
  │
  ├─ workspace-init  copies promo tool → /workspace/promo-tool
  │
  └─ claude (Claude Code)
        ├─ LLM → madeye-proxy (:4000) → MadEye
        └─ invokes promo-finish finish …
              ▼
        promo-finish (orchestrator — Claude only runs this)
              │  POST promo-api …/upload  (stub S3)
              │  POST promo-api …/webhook
              ▼
        promo-api → GenAxis  PROMO_SCRIPT_COMPLETED
```

---

## Services

| Service | Cluster | Docker Desktop | Caller |
|---------|---------|----------------|--------|
| promo-api | `:8080` | `localhost:30080` | GenAxis, promo-finish |
| madeye-proxy | `:4000` | `localhost:30040` | Claude Code |
| MadEye | external | `madeye-dev.internal.pocketfm.org` (+ `/v1` in code) | madeye-proxy only |

---

## MadEye proxy (our code — no LiteLLM)

Claude Code speaks **Anthropic API**. MadEye speaks **OpenAI API**.  
**`bte-promo-script/madeye-proxy/`** is a small Go service we own that:

1. Accepts `POST /v1/messages` (Anthropic format)
2. Strips trailing assistant prefills (MadEye rejects them)
3. Converts request → OpenAI `chat/completions`
4. Calls MadEye with `MADEYE_API_KEY`
5. Converts response back → Anthropic format (including streaming)

```
Claude Code  ──Anthropic──►  madeye-proxy  ──OpenAI──►  MadEye
              PROXY_MASTER_KEY              MADEYE_API_KEY
```

Polecat env (set by promo-api): `ANTHROPIC_BASE_URL=http://madeye-proxy.gastown-system.svc:4000`, `ANTHROPIC_AUTH_TOKEN` from `api/defaults.go` (`DefaultProxyMasterKey`).

**Two different keys on the LLM path:**

| Key | Who sends it | Who checks it | Real purpose |
|-----|--------------|---------------|--------------|
| `PROXY_MASTER_KEY` | Claude Code as `ANTHROPIC_AUTH_TOKEN` (Bearer) | madeye-proxy | **Gate** — only our polecats may call the proxy |
| `MADEYE_API_KEY` | madeye-proxy as `Authorization: Bearer` | MadEye | **Upstream** — actual LLM billing/access |

`PROXY_MASTER_KEY` is **not** a real Anthropic key and **not** the MadEye key. Claude Code always sends an API key header; we reuse that slot for our proxy password. `PROXY_MASTER_KEY` and `MADEYE_USER_EMAIL` live in `api/defaults.go` and are applied to the madeye-proxy deployment by `deploy-infra.sh` — not stored in K8s secrets.

---

## Who calls what

| Step | Caller | Callee |
|------|--------|--------|
| Start job | GenAxis | promo-api |
| LLM | Claude Code | madeye-proxy → MadEye |
| Finish | Claude | `promo-finish finish …` |
| Upload + webhook | **promo-finish** | promo-api |
| Done | promo-api | GenAxis |

Claude never calls GenAxis, S3, or MadEye directly.

---

## Secrets (K8s-injected)

Only **real upstream keys** live in Kubernetes secrets. Pods receive them via `secretKeyRef` — not from `.env` at runtime.

| Secret | Key | Env in pod | Used by |
|--------|-----|------------|---------|
| `madeye-proxy-secrets` | `MADEYE_API_KEY` | `MADEYE_API_KEY` | madeye-proxy → MadEye |
| `genaxis-auth` | `api-key` | `GENAXIS_API_KEY` | promo-api → GenAxis webhooks |

**Non-secret config** (`PROXY_MASTER_KEY`, `MADEYE_USER_EMAIL`, in-cluster proxy URL, rig name, etc.) lives in `api/defaults.go`. `deploy-infra.sh` sets `PROXY_MASTER_KEY` and `MADEYE_USER_EMAIL` on the madeye-proxy deployment; promo-api sets polecat `ANTHROPIC_AUTH_TOKEN` from the same defaults.

**K8s env (not secrets, not `defaults.go`):** `MADEYE_BASE_URL` on the madeye-proxy pod — upstream MadEye **host only** (e.g. `https://madeye-dev.internal.pocketfm.org`). Same name as GenAxis. The proxy appends `/v1/chat/completions` in code. Set in `deploy/madeye-proxy.yaml` (local) or infra manifest (prod).

**One-time setup (local or CI → Vault):**

```bash
cp bte-promo-script/secrets.env.example bte-promo-script/secrets.env
# edit GENAXIS_API_KEY and MADEYE_API_KEY only
./bte-promo-script/deploy/apply-secrets.sh
```

---

## Deploy

```bash
./bte-promo-script/deploy/apply-secrets.sh    # once — K8s secrets
./bte-promo-script/deploy/deploy-infra.sh     # operator + madeye-proxy + Rig
./bte-promo-script/deploy/deploy-promo.sh     # promo-api + polecat-agent
```

| Script | Builds |
|--------|--------|
| `deploy-infra.sh` | gastown-operator, **madeye-proxy**, Rig |
| `deploy-promo.sh` | polecat-agent, promo-api |

---

## Code locations

| Piece | Path |
|-------|------|
| MadEye proxy | `bte-promo-script/madeye-proxy/` |
| promo-api | `bte-promo-script/api/` |
| promo-finish | `bte-promo-script/tools/finish/` |
| Promo tool | `bte-promo-script/repos/bte-promo-script-repo/` |

S3 upload is still a stub (`api/s3.go`). Job outputs stay in the pod unless copied out.

---

## Infra changes required (cloud K8s)

Local scripts target **Docker Desktop** (local `docker build`, `*:local` images, NodePort). Architecture is cloud-ready; **values and CI** need infra updates before production.

### Who owns what

| Pipeline | Script | Infra | App team |
|----------|--------|-------|----------|
| Base platform | `deploy-infra.sh` | Yes | — |
| Promo app | `deploy-promo.sh` | Secrets + network | Builds & pushes images |

### Secrets (create in K8s — not from `.env` in prod)

| Secret | Keys | Used by |
|--------|------|---------|
| `madeye-proxy-secrets` | `MADEYE_API_KEY` | madeye-proxy pod → MadEye |
| `genaxis-auth` | `api-key` | promo-api → GenAxis webhooks |

`PROXY_MASTER_KEY`, `MADEYE_USER_EMAIL`, and polecat `ANTHROPIC_AUTH_TOKEN` come from `api/defaults.go` (applied at deploy time, not K8s secrets).

### Files infra adapts

**`deploy/deploy-infra.sh`**

| Today (local) | Cloud change |
|---------------|--------------|
| `KUBE_CONTEXT=docker-desktop` | Set to prod cluster context |
| `make docker-build-e2e` (local image) | CI builds + pushes operator to registry |
| `imagePullPolicy: Never` patch on operator | Remove; use registry URL + `Always` |
| `docker build madeye-proxy` → `madeye-proxy:local` | CI push → `registry/madeye-proxy:tag` |
| Secrets from `bte-promo-script/secrets.env` via `apply-secrets.sh` | Vault → K8s secrets |

**`deploy/deploy-promo.sh`** (app CI runs build; infra provides cluster + secrets)

| Today (local) | Cloud change |
|---------------|--------------|
| `kubectl config use-context docker-desktop` | Prod context |
| `docker build` → `promo-api:local`, `polecat-agent:local` | CI push full registry URLs |
| `AGENT_IMAGE=polecat-agent:tag` | Must be registry URL (worker nodes pull on each job) |
| `genaxis-auth` from `apply-secrets.sh` | Vault → K8s secret |

**`deploy/madeye-proxy.yaml`**

| Field | Local | Cloud |
|-------|-------|-------|
| `image` | `madeye-proxy:local` | Registry URL + tag |
| `imagePullPolicy` | `IfNotPresent` | `Always` (typical) |
| `MADEYE_BASE_URL` | `https://madeye-dev.internal.pocketfm.org` (host only; `/v1` in app) | Confirm prod host + network egress |
| Service | `NodePort` 30040 | `ClusterIP` (internal only) |

**`deploy/bte-promo-script.yaml`**

| Field | Local | Cloud |
|-------|-------|-------|
| `image` | `promo-api:local` | Registry URL + tag |
| `AGENT_IMAGE` | `polecat-agent:local` | Registry URL + tag |
| Service | `NodePort` 30080 | Ingress or internal LB for GenAxis |
| Namespace | `gastown-system` | Same or infra standard |

**Operator** (repo root, not under `bte-promo-script/`)

| Step | Cloud change |
|------|--------------|
| `make install` | Install CRDs once per cluster |
| `make deploy IMG=...` | `IMG` = registry operator image |
| Namespace | `gastown-operator-system` |

**`deploy/rig.yaml.tpl`** — infra may tune `maxPolecats`; rig name/URL usually from `api/defaults.go` (app).

### Network (no code change)

| URL | Caller | Notes |
|-----|--------|-------|
| `http://promo-api.gastown-system.svc:8080` | promo-finish | In-cluster only |
| `http://madeye-proxy.gastown-system.svc:4000` | Claude Code | In-cluster only |
| GenAxis → promo-api | External | Infra exposes via Ingress/LB |
| madeye-proxy → MadEye | Egress | Cluster must reach MadEye host |

### Infra checklist

```
□ Namespaces: gastown-operator-system, gastown-system
□ Image registry + imagePullSecrets (if private)
□ CI: push gastown-operator, madeye-proxy, promo-api, polecat-agent
□ K8s secrets: madeye-proxy-secrets, genaxis-auth
□ CRDs + operator deploy
□ madeye-proxy.yaml (registry image, ClusterIP)
□ rig.yaml.tpl
□ bte-promo-script.yaml (registry images, GenAxis Ingress)
□ Smoke: GenAxis → promo-api → polecat → madeye-proxy → MadEye
```

### What infra does not change

| Path | Owner |
|------|-------|
| `tools/finish/`, `repos/bte-promo-script-repo/` | App (baked into polecat-agent) |
| `api/`, `madeye-proxy/` Go code | App |
| Dockerfiles | App builds; infra only needs pull access |
| Operator Go code (`internal/`, `pkg/`) | App release |
