# LiteLLM → MadEye

Local and in-cluster adapter so Claude Code (Anthropic API) can call MadEye
(OpenAI `/v1/chat/completions` only).

Full guide: [docs/LITE_LLM.md](../../docs/LITE_LLM.md)

## Docker Desktop Kubernetes (recommended for local taste)

1. **Docker Desktop → Settings → Kubernetes → Enable Kubernetes → Apply & Restart**
2. Confirm:

```bash
kubectl config use-context docker-desktop
kubectl get nodes
```

3. Deploy:

```bash
cp secret.env.example secret.env   # edit MADEYE_API_KEY
./apply-docker-desktop.sh
```

4. Smoke test:

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

## Docker Compose (no Kubernetes)

```bash
cp .env.example .env   # edit MADEYE_API_KEY + MADEYE_USER_EMAIL
docker compose --env-file .env up -d

curl -s http://localhost:4000/v1/messages \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer sk-local-litellm" \
  -H "anthropic-version: 2023-06-01" \
  -d '{
    "model": "claude-opus-4-8",
    "max_tokens": 64,
    "messages": [{"role":"user","content":"Hello! which model are you using?"}]
  }'
```
