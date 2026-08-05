# LiteLLM + MadEye

Route Polecat Claude Code traffic through LiteLLM to PocketFM MadEye
(`OpenAI /v1/chat/completions` only).

```
Polecat (claude)
  Anthropic /v1/messages + ANTHROPIC_BASE_URL
        ↓
LiteLLM (shared Deployment / docker-compose)
  translates tools/messages
        ↓
MadEye https://madeye.internal.pocketfm.org/v1/chat/completions
  Bearer token + metadata.user_email
```

## Prerequisites

- MadEye reachable from your machine (VPN / internal DNS)
- MadEye Bearer token
- Your PocketFM email for MadEye `metadata.user_email`

---

## 1. Taste LiteLLM locally (no Kubernetes)

```bash
cd deploy/litellm

export MADEYE_API_KEY='your-madeye-bearer-token'
export MADEYE_USER_EMAIL='priyanshu.rajput@pocketfm.com'
export LITELLM_MASTER_KEY='sk-local-litellm'

chmod +x entrypoint.sh
docker compose up -d
```

Smoke-test Anthropic-format call through LiteLLM:

```bash
curl -s http://localhost:4000/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer ${LITELLM_MASTER_KEY}" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 64,
    "messages": [{"role":"user","content":"Hello! which model are you using?"}]
  }'
```

If that returns a model reply, the adapter works. Claude Code uses the same path.

Optional — point local Claude Code at it:

```bash
export ANTHROPIC_BASE_URL=http://localhost:4000
export ANTHROPIC_AUTH_TOKEN="${LITELLM_MASTER_KEY}"
export ANTHROPIC_MODEL=claude-opus-4-8
claude --print "Hello! which model are you using?"
```

---

## 2. Deploy LiteLLM on Docker Desktop Kubernetes

**Enable the cluster first** (one-time):

1. Open **Docker Desktop → Settings → Kubernetes**
2. Check **Enable Kubernetes**
3. **Apply & Restart** and wait until Kubernetes shows green
4. Verify:

```bash
kubectl config use-context docker-desktop
kubectl get nodes
```

Then deploy LiteLLM:

```bash
cd deploy/litellm
cp secret.env.example secret.env
# edit MADEYE_API_KEY (and email if needed)

chmod +x apply-docker-desktop.sh
./apply-docker-desktop.sh
```

Smoke test (NodePort **30040** on Docker Desktop):

```bash
curl -s http://localhost:30040/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-local-litellm" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 64,
    "messages": [{"role":"user","content":"Hello! which model are you using?"}]
  }'
```

MadEye must be reachable from Docker Desktop (VPN on the host usually works for outbound pod traffic).

---

## 3. Deploy LiteLLM in any Kubernetes

```bash
# Edit MadEye token + email in the Secret
vim deploy/litellm/k8s.yaml

kubectl apply -f deploy/litellm/k8s.yaml
kubectl -n gastown-system rollout status deploy/litellm
kubectl -n gastown-system port-forward svc/litellm 4000:4000
```

Re-run the same `curl` against `http://localhost:4000`.

---

## 4. Polecat agentConfig (operator wiring)

The pod builder injects:

| `agentConfig` field | Container env |
|---------------------|---------------|
| `modelProvider.endpoint` | `ANTHROPIC_BASE_URL` |
| `modelProvider.apiKeySecretRef` | `ANTHROPIC_AUTH_TOKEN` |
| `model` | `ANTHROPIC_MODEL` |
| `env` | passthrough |

Sample: `config/samples/litellm/polecat-madeye.yaml`

```yaml
agentConfig:
  provider: litellm
  model: claude-opus-4-8
  modelProvider:
    endpoint: http://litellm.gastown-system.svc:4000
    apiKeySecretRef:
      name: litellm-auth
      key: master-key
```

---

## Local Kind path (operator + LiteLLM)

Prefer Docker Desktop full stack: see [bte-promo-script/README.md](../bte-promo-script/README.md) or `make demo-docker-desktop`.

```bash
# 1. Build/load operator into Kind (or make demo)
make demo

# 2. Deploy LiteLLM
kubectl apply -f deploy/litellm/k8s.yaml

# 3. Apply sample (after editing git repo/secrets as needed)
kubectl apply -f config/samples/litellm/polecat-madeye.yaml

# 4. Watch
kubectl -n gastown-system logs -f deploy/litellm
kubectl gt polecat logs smoke/madeye-smoke -f -n gastown-system
```

Start with step 1 (docker compose + curl) before involving the operator.

---

## Troubleshooting

| Symptom | Check |
|---------|--------|
| curl to LiteLLM fails DNS to MadEye | VPN / can you curl MadEye directly? |
| 401 from MadEye | `MADEYE_API_KEY` Bearer token |
| MadEye rejects request | `user_email` in config / Secret |
| `does not support assistant message prefill` | LiteLLM was routing Claude Code through OpenAI **Responses API**. Set `LITELLM_USE_CHAT_COMPLETIONS_URL_FOR_ANTHROPIC_MESSAGES=true` and redeploy. Also ensure `prefill_proxy` strips trailing assistant turns. |
| Polecat still hits api.anthropic.com | `ANTHROPIC_BASE_URL` missing — confirm `agentConfig` on Polecat |
| Tool-call errors | LiteLLM version; check LiteLLM logs for translation failures |
