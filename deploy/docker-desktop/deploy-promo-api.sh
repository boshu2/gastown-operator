#!/usr/bin/env bash
# Build and deploy promo-api to Docker Desktop Kubernetes.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"

kubectl config use-context docker-desktop >/dev/null

echo "==> Building promo-api:local (load into Docker Desktop image store)"
docker buildx build --load -t promo-api:local -f cmd/promo-api/Dockerfile .

echo "==> Applying manifests"
kubectl apply -f deploy/docker-desktop/promo-api.yaml
kubectl -n gastown-system delete pod -l app=promo-api --ignore-not-found
kubectl -n gastown-system rollout status deploy/promo-api --timeout=120s

echo
echo "promo-api ready"
echo "  Health:   curl http://localhost:30080/health"
echo "  Generate: curl -X POST http://localhost:30080/v1/promo/generate -H 'Content-Type: application/json' -d '{\"prompt\":\"Write a short promo script for a comedy podcast\"}'"
echo "  Status:   curl http://localhost:30080/v1/promo/jobs/<job_name>"
