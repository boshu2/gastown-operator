#!/usr/bin/env bash
# shellcheck shell=bash
require_k8s_secret() {
  local ns="$1"
  local name="$2"
  if ! kubectl -n "${ns}" get secret "${name}" >/dev/null 2>&1; then
    echo "ERROR: missing secret ${name} in namespace ${ns}."
    echo "  Run: ./bte-promo-script/deploy/apply-secrets.sh"
    exit 1
  fi
}

require_promo_secrets() {
  local ns="${1:-gastown-system}"
  require_k8s_secret "${ns}" madeye-proxy-secrets
  require_k8s_secret "${ns}" genaxis-auth
}
