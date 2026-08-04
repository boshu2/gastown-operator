#!/bin/sh
set -e

# MadEye speaks OpenAI chat/completions only. Force LiteLLM off the Responses API
# path when Claude Code calls Anthropic /v1/messages (critical for prefill errors).
export LITELLM_USE_CHAT_COMPLETIONS_URL_FOR_ANTHROPIC_MESSAGES=true

# Inject secrets into config (more reliable than LiteLLM os.environ/ syntax)
sed \
  -e "s|MADEYE_API_KEY_PLACEHOLDER|${MADEYE_API_KEY}|g" \
  -e "s|MADEYE_USER_EMAIL_PLACEHOLDER|${MADEYE_USER_EMAIL}|g" \
  -e "s|LITELLM_MASTER_KEY_PLACEHOLDER|${LITELLM_MASTER_KEY}|g" \
  /app/config.template.yaml > /tmp/config.yaml

# LiteLLM internally; edge proxy on :4000 strips Claude Code prefills
litellm --config /tmp/config.yaml --port 4001 --host 127.0.0.1 &
LITELLM_PID=$!
trap 'kill ${LITELLM_PID} 2>/dev/null || true' EXIT INT TERM

python -c 'import time,urllib.request
for _ in range(90):
  try:
    urllib.request.urlopen("http://127.0.0.1:4001/health/liveliness", timeout=2); break
  except Exception:
    time.sleep(1)
else:
  raise SystemExit("LiteLLM not ready on :4001")'

export LITELLM_UPSTREAM=http://127.0.0.1:4001
export PREFILL_PROXY_PORT=4000
exec python /app/prefill_proxy.py
