# BTE Gastown — production Kubernetes manifests
#
# Namespace: bte-gastown
# Architecture: Pod 1 (infra-monolith) + Pod 2 per job (polecat/claude)
#
# Prerequisites (infra team):
#   - EKS cluster + IRSA role bte-gastown-role
#   - AWS Secrets Manager path bte-gastown/prod/bte-gastown
#   - External Secrets Operator installed
#   - gastown-operator CRDs installed (make install / helm)
#   - ECR images: bte-gastown, promo-api, madeye-proxy, polecat-agent
#   - DNS: bte-gastown.pocketfm.org → ingress
#
# Apply order:
#   1. kubectl apply -f namespace.yaml
#   2. make install   # CRDs + gastown-operator-manager-role (from repo root)
#   3. kubectl apply -f serviceaccount.yaml
#   4. kubectl apply -f secret-store.yaml
#   5. kubectl apply -f external-secret.yaml   # wait for secrets
#   6. kubectl apply -f rbac.yaml
#   7. Edit infra-monolith.yaml — set <ECR> image URLs
#   8. kubectl apply -f infra-monolith.yaml
#   9. kubectl apply -f services.yaml
#  10. kubectl apply -f ingress.yaml
#  11. kubectl apply -f rig.yaml
#
# Skipped (later):
#   - S3 upload + CloudFront URL in promo-api
#   - HPA (Pod 1 min 1 max 2)
#   - KEDA ScaledObject (Pod 2 max 4 parallel jobs)
#
# Resource budget (from load test):
#   Pod 1 infra-monolith: 1 CPU / 1Gi total
#   Pod 2 polecat job:    2 CPU / 2Gi (set in promo-api Polecat CR)
