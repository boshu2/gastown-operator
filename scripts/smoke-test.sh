#!/bin/bash
# Smoke Test — Verify published release artifacts in Kind
#
# Quick validation that GHCR image + Helm chart deploy and work.
# For full release validation (community + FIPS), use release-validation.sh.
#
# Usage:
#   ./scripts/smoke-test.sh --version 0.5.0
#
# Options:
#   --version VERSION    Version to test (required)
#   --skip-cleanup       Don't delete Kind cluster after tests
#   --image IMAGE        Override operator image
#   --verbose            Enable verbose output
#
# Prerequisites:
#   - docker (running)
#   - kind
#   - helm
#   - kubectl

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
CLUSTER_NAME="gastown-smoke-test"
GHCR_REGISTRY="ghcr.io/boshu2"
HELM_CHART="oci://${GHCR_REGISTRY}/charts/gastown-operator"
NAMESPACE="gastown-system"
SKIP_CLEANUP=false
VERBOSE=false
VERSION=""
OPERATOR_IMAGE=""
PASS_COUNT=0
FAIL_COUNT=0

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --skip-cleanup)
      SKIP_CLEANUP=true
      shift
      ;;
    --image)
      OPERATOR_IMAGE="$2"
      shift 2
      ;;
    --verbose)
      VERBOSE=true
      shift
      ;;
    -h|--help)
      head -18 "$0" | tail -16
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

if [ -z "$VERSION" ]; then
  echo -e "${RED}Error: --version is required${NC}"
  echo "Usage: $0 --version 0.5.0"
  exit 1
fi

if [ -z "$OPERATOR_IMAGE" ]; then
  OPERATOR_IMAGE="${GHCR_REGISTRY}/gastown-operator:${VERSION}"
fi

# Utility functions
log_info()    { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; PASS_COUNT=$((PASS_COUNT + 1)); }
log_error()   { echo -e "${RED}[FAIL]${NC} $1"; FAIL_COUNT=$((FAIL_COUNT + 1)); }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }

log_section() {
  echo ""
  echo -e "${BLUE}========================================${NC}"
  echo -e "${BLUE}  $1${NC}"
  echo -e "${BLUE}========================================${NC}"
  echo ""
}

cleanup() {
  if [ "$SKIP_CLEANUP" = true ]; then
    log_warning "Skipping cleanup (--skip-cleanup)"
    log_warning "Cluster '$CLUSTER_NAME' is still running"
    log_warning "Delete with: kind delete cluster --name $CLUSTER_NAME"
    return
  fi

  log_section "Cleanup"
  log_info "Deleting Kind cluster..."
  kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
  log_info "Cleanup complete"
}

trap cleanup EXIT

# ============================================================================
# Banner
# ============================================================================

echo -e "${BLUE}========================================"
echo "  Gas Town Operator Smoke Test"
echo "========================================${NC}"
echo ""
echo "Version:  $VERSION"
echo "Image:    $OPERATOR_IMAGE"
echo "Chart:    $HELM_CHART"
echo "Cluster:  $CLUSTER_NAME"
echo ""

# ============================================================================
# Prerequisites
# ============================================================================

log_section "Prerequisites"

for cmd in docker kind helm kubectl; do
  if command -v "$cmd" &> /dev/null; then
    log_success "$cmd installed"
  else
    log_error "$cmd not found"
    exit 1
  fi
done

if ! docker info &> /dev/null; then
  log_error "Docker daemon is not running"
  exit 1
fi
log_success "Docker daemon running"

# ============================================================================
# Kind Cluster
# ============================================================================

log_section "Kind Cluster"

kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true
log_info "Creating Kind cluster: $CLUSTER_NAME"
kind create cluster --name "$CLUSTER_NAME" --wait 60s
log_success "Kind cluster created"

kubectl cluster-info --context "kind-${CLUSTER_NAME}" > /dev/null 2>&1
log_success "kubectl connected to cluster"

# ============================================================================
# Helm Install
# ============================================================================

log_section "Helm Install"

log_info "Installing gastown-operator ${VERSION} from GHCR..."
helm install gastown-operator "$HELM_CHART" \
  --version "$VERSION" \
  --namespace "$NAMESPACE" \
  --create-namespace \
  --set image.tag="$VERSION" \
  --wait \
  --timeout 120s

log_success "Helm install completed"

# ============================================================================
# CRD Verification
# ============================================================================

log_section "CRD Verification"

EXPECTED_CRDS=("beadstores" "convoys" "polecats" "refineries" "rigs" "witnesses")
for crd in "${EXPECTED_CRDS[@]}"; do
  if kubectl get crd "gastown.gastown.io_${crd}" &> /dev/null 2>&1 || \
     kubectl get crd "${crd}.gastown.gastown.io" &> /dev/null 2>&1; then
    log_success "CRD: $crd"
  else
    log_error "CRD missing: $crd"
  fi
done

# ============================================================================
# Controller Health
# ============================================================================

log_section "Controller Health"

log_info "Waiting for controller-manager pod..."
if kubectl wait --for=condition=Ready pod \
  -l control-plane=controller-manager \
  -n "$NAMESPACE" \
  --timeout=60s &> /dev/null; then
  log_success "Controller pod is Ready"
else
  log_error "Controller pod not ready after 60s"
  kubectl get pods -n "$NAMESPACE" 2>/dev/null || true
fi

POD_NAME=$(kubectl get pods -n "$NAMESPACE" -l control-plane=controller-manager -o jsonpath='{.items[0].metadata.name}' 2>/dev/null)
if [ -n "$POD_NAME" ]; then
  POD_STATUS=$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.phase}')
  if [ "$POD_STATUS" = "Running" ]; then
    log_success "Controller pod status: $POD_STATUS"
  else
    log_error "Controller pod status: $POD_STATUS (expected Running)"
  fi

  RESTART_COUNT=$(kubectl get pod "$POD_NAME" -n "$NAMESPACE" -o jsonpath='{.status.containerStatuses[0].restartCount}' 2>/dev/null || echo "unknown")
  if [ "$RESTART_COUNT" = "0" ]; then
    log_success "No container restarts"
  else
    log_warning "Container restarts: $RESTART_COUNT"
  fi
fi

# ============================================================================
# Rig Lifecycle
# ============================================================================

log_section "Rig Lifecycle"

log_info "Creating test Rig..."
kubectl apply -f - <<'EOF'
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: smoke-test-rig
  namespace: default
spec:
  gitURL: "https://github.com/example/test-repo.git"
  description: "Smoke test rig"
EOF

log_info "Waiting for Rig reconciliation (30s)..."
sleep 5

# Check Witness auto-created
WITNESS_FOUND=false
for i in $(seq 1 6); do
  if kubectl get witness smoke-test-rig-witness -n default &> /dev/null 2>&1; then
    WITNESS_FOUND=true
    break
  fi
  sleep 5
done

if [ "$WITNESS_FOUND" = true ]; then
  log_success "Witness auto-created: smoke-test-rig-witness"
else
  log_error "Witness not auto-created after 30s"
fi

# Check Refinery auto-created
REFINERY_FOUND=false
for i in $(seq 1 6); do
  if kubectl get refinery smoke-test-rig-refinery -n default &> /dev/null 2>&1; then
    REFINERY_FOUND=true
    break
  fi
  sleep 5
done

if [ "$REFINERY_FOUND" = true ]; then
  log_success "Refinery auto-created: smoke-test-rig-refinery"
else
  log_error "Refinery not auto-created after 30s"
fi

# Test cascade delete
log_info "Testing cascade delete..."
kubectl delete rig smoke-test-rig -n default --timeout=30s 2>/dev/null || true
sleep 5

if ! kubectl get witness smoke-test-rig-witness -n default &> /dev/null 2>&1; then
  log_success "Witness cascade deleted"
else
  log_warning "Witness still exists after Rig deletion"
fi

if ! kubectl get refinery smoke-test-rig-refinery -n default &> /dev/null 2>&1; then
  log_success "Refinery cascade deleted"
else
  log_warning "Refinery still exists after Rig deletion"
fi

# ============================================================================
# Summary
# ============================================================================

log_section "Summary"

TOTAL=$((PASS_COUNT + FAIL_COUNT))
echo "Results: ${PASS_COUNT}/${TOTAL} passed"
echo ""

if [ "$FAIL_COUNT" -gt 0 ]; then
  echo -e "${RED}SMOKE TEST FAILED${NC} ($FAIL_COUNT failures)"

  if [ "$VERBOSE" = true ] && [ -n "$POD_NAME" ]; then
    echo ""
    log_info "Controller logs (last 50 lines):"
    kubectl logs "$POD_NAME" -n "$NAMESPACE" --tail=50 2>/dev/null || true
  fi

  exit 1
else
  echo -e "${GREEN}SMOKE TEST PASSED${NC}"
  exit 0
fi
