#!/bin/sh
set -e

MADEYE_API_BASE="${MADEYE_API_BASE:-https://madeye.internal.pocketfm.org/v1}"
MADEYE_USER_EMAIL="${MADEYE_USER_EMAIL:?MADEYE_USER_EMAIL is required}"

export LITELLM_USE_CHAT_COMPLETIONS_URL_FOR_ANTHROPIC_MESSAGES=true

# Substitute only the placeholders LiteLLM cannot resolve via os.environ/
sed \
  -e "s|\${MADEYE_API_BASE}|${MADEYE_API_BASE}|g" \
  -e "s|\${MADEYE_USER_EMAIL}|${MADEYE_USER_EMAIL}|g" \
  /app/config.template.yaml > /tmp/config.yaml

echo "Starting LiteLLM → ${MADEYE_API_BASE} (user=${MADEYE_USER_EMAIL})"
echo "Chat completions mode for Anthropic messages: ON"

# LiteLLM on 4001; edge proxy on 4000 strips Claude Code assistant prefills
litellm --config /tmp/config.yaml --port 4001 --host 127.0.0.1 &
LITELLM_PID=$!

cleanup() {
  kill "${LITELLM_PID}" 2>/dev/null || true
}
trap cleanup EXIT INT TERM

python - <<'PY'
import time, urllib.request
for _ in range(90):
    try:
        urllib.request.urlopen("http://127.0.0.1:4001/health/liveliness", timeout=2)
        break
    except Exception:
        time.sleep(1)
else:
    raise SystemExit("LiteLLM failed to become ready on :4001")
PY

export LITELLM_UPSTREAM=http://127.0.0.1:4001
export PREFILL_PROXY_PORT=4000
exec python /app/prefill_proxy.py
