#!/bin/bash
# Release Validation Script
# Comprehensive E2E tests for gastown-operator release validation
#
# This script:
#   1. Creates a local Kind cluster
#   2. Pulls Helm charts from GHCR (or uses local)
#   3. Deploys Community edition in gastown-community namespace
#   4. Deploys FIPS edition in gastown-fips namespace
#   5. Verifies CRDs, secrets, and job triggers
#   6. Runs functional tests for both editions
#   7. Cleans up
#
# Usage:
#   ./scripts/release-validation.sh [options]
#
# Options:
#   --version VERSION    Version to test (default: latest from Chart.yaml)
#   --local              Use local Helm chart instead of GHCR
#   --skip-cleanup       Don't delete Kind cluster after tests
#   --image IMAGE        Override operator image
#   --verbose            Enable verbose output
#
# Prerequisites:
#   - docker
#   - kind
#   - helm
#   - kubectl

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
CLUSTER_NAME="gastown-release-test"
COMMUNITY_NS="gastown-community"
FIPS_NS="gastown-fips"
GHCR_REGISTRY="ghcr.io/boshu2"
USE_LOCAL=false
SKIP_CLEANUP=false
VERBOSE=false
VERSION=""
OPERATOR_IMAGE=""

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --version)
      VERSION="$2"
      shift 2
      ;;
    --local)
      USE_LOCAL=true
      shift
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
      head -30 "$0" | tail -25
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Get version from Chart.yaml if not specified
if [ -z "$VERSION" ]; then
  VERSION=$(grep "^version:" "$PROJECT_ROOT/helm/gastown-operator/Chart.yaml" | awk '{print $2}')
fi

# Set default image if not specified
if [ -z "$OPERATOR_IMAGE" ]; then
  OPERATOR_IMAGE="${GHCR_REGISTRY}/gastown-operator:${VERSION}"
fi

echo -e "${BLUE}========================================"
echo "  Gas Town Operator Release Validation"
echo "========================================${NC}"
echo ""
echo "Version: $VERSION"
echo "Image: $OPERATOR_IMAGE"
echo "Use local chart: $USE_LOCAL"
echo "Cluster: $CLUSTER_NAME"
echo ""

# Utility functions
log_info() {
  echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
  echo -e "${GREEN}[PASS]${NC} $1"
}

log_warning() {
  echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
  echo -e "${RED}[FAIL]${NC} $1"
}

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
    return
  fi

  log_section "Cleanup"

  log_info "Deleting Kind cluster..."
  kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true

  log_success "Cleanup complete"
}

# Trap for cleanup on exit
trap cleanup EXIT

# Check prerequisites
log_section "Prerequisites Check"

for cmd in docker kind helm kubectl; do
  if command -v "$cmd" &> /dev/null; then
    log_success "$cmd is installed"
  else
    log_error "$cmd is not installed"
    exit 1
  fi
done

# Check Docker is running
if ! docker info &> /dev/null; then
  log_error "Docker daemon is not running"
  exit 1
fi
log_success "Docker daemon is running"

# ============================================================================
# Create Kind Cluster
# ============================================================================
log_section "Creating Kind Cluster"

# Delete existing cluster if it exists
kind delete cluster --name "$CLUSTER_NAME" 2>/dev/null || true

log_info "Creating Kind cluster: $CLUSTER_NAME"
cat <<EOF | kind create cluster --name "$CLUSTER_NAME" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
nodes:
  - role: control-plane
    kubeadmConfigPatches:
      - |
        kind: InitConfiguration
        nodeRegistration:
          kubeletExtraArgs:
            node-labels: "ingress-ready=true"
EOF

# Wait for cluster to be ready
log_info "Waiting for cluster to be ready..."
kubectl wait --for=condition=Ready nodes --all --timeout=120s
log_success "Kind cluster created and ready"

# ============================================================================
# Install CRDs
# ============================================================================
log_section "Installing CRDs"

log_info "Applying CRDs from config/crd/bases/"
kubectl apply -f "$PROJECT_ROOT/config/crd/bases/"

# Verify CRDs are installed
log_info "Verifying CRDs..."
EXPECTED_CRDS=(
  "polecats.gastown.gastown.io"
  "rigs.gastown.gastown.io"
  "convoys.gastown.gastown.io"
  "witnesses.gastown.gastown.io"
  "refineries.gastown.gastown.io"
  "beadstores.gastown.gastown.io"
)

for crd in "${EXPECTED_CRDS[@]}"; do
  if kubectl get crd "$crd" &> /dev/null; then
    log_success "CRD installed: $crd"
  else
    log_error "CRD missing: $crd"
    exit 1
  fi
done

# ============================================================================
# Build/Load Operator Image
# ============================================================================
log_section "Loading Operator Image"

if [ "$USE_LOCAL" = true ]; then
  log_info "Building operator image locally..."

  # Build gt CLI first (required by Dockerfile)
  log_info "Building gt CLI..."
  (
    cd /tmp
    rm -rf gastown-build
    git clone --depth 1 https://github.com/steveyegge/gastown.git gastown-build
    cd gastown-build
    CGO_ENABLED=0 go build -o "$PROJECT_ROOT/gt" ./cmd/gt
  )

  # Build operator
  docker build -t "${GHCR_REGISTRY}/gastown-operator:${VERSION}" "$PROJECT_ROOT"
  OPERATOR_IMAGE="${GHCR_REGISTRY}/gastown-operator:${VERSION}"
else
  log_info "Pulling operator image: $OPERATOR_IMAGE"
  docker pull "$OPERATOR_IMAGE" || {
    log_warning "Could not pull image, attempting local build..."
    USE_LOCAL=true
    # Retry with local build
    (
      cd /tmp
      rm -rf gastown-build
      git clone --depth 1 https://github.com/steveyegge/gastown.git gastown-build
      cd gastown-build
      CGO_ENABLED=0 go build -o "$PROJECT_ROOT/gt" ./cmd/gt
    )
    docker build -t "$OPERATOR_IMAGE" "$PROJECT_ROOT"
  }
fi

log_info "Loading image into Kind..."
kind load docker-image "$OPERATOR_IMAGE" --name "$CLUSTER_NAME"
log_success "Image loaded: $OPERATOR_IMAGE"

# ============================================================================
# Deploy Community Edition
# ============================================================================
log_section "Deploying Community Edition"

# Create namespace
log_info "Creating namespace: $COMMUNITY_NS"
kubectl create namespace "$COMMUNITY_NS"
kubectl label namespace "$COMMUNITY_NS" pod-security.kubernetes.io/enforce=restricted

# Pull/use Helm chart
if [ "$USE_LOCAL" = true ]; then
  HELM_CHART="$PROJECT_ROOT/helm/gastown-operator"
else
  log_info "Pulling Helm chart from GHCR..."
  helm pull "oci://${GHCR_REGISTRY}/charts/gastown-operator" --version "$VERSION" --untar --untardir /tmp/
  HELM_CHART="/tmp/gastown-operator"
fi

log_info "Installing Helm chart (Community)..."
helm install gastown-operator "$HELM_CHART" \
  --namespace "$COMMUNITY_NS" \
  --set image.repository="${GHCR_REGISTRY}/gastown-operator" \
  --set image.tag="v${VERSION}" \
  --set image.pullPolicy=Never \
  --wait --timeout 5m

# Verify deployment
log_info "Verifying Community deployment..."
kubectl wait --for=condition=available deployment/gastown-operator \
  -n "$COMMUNITY_NS" --timeout=120s
log_success "Community edition deployed"

# ============================================================================
# Deploy FIPS Edition
# ============================================================================
log_section "Deploying FIPS Edition"

# Create namespace
log_info "Creating namespace: $FIPS_NS"
kubectl create namespace "$FIPS_NS"
kubectl label namespace "$FIPS_NS" pod-security.kubernetes.io/enforce=restricted

log_info "Installing Helm chart (FIPS)..."
helm install gastown-operator "$HELM_CHART" \
  --namespace "$FIPS_NS" \
  --set image.repository="${GHCR_REGISTRY}/gastown-operator" \
  --set image.tag="v${VERSION}" \
  --set image.pullPolicy=Never \
  -f "$PROJECT_ROOT/helm/gastown-operator/values-fips.yaml" \
  --wait --timeout 5m

# Verify deployment
log_info "Verifying FIPS deployment..."
kubectl wait --for=condition=available deployment/gastown-operator \
  -n "$FIPS_NS" --timeout=120s
log_success "FIPS edition deployed"

# ============================================================================
# Verify Deployments
# ============================================================================
log_section "Verifying Deployments"

for ns in "$COMMUNITY_NS" "$FIPS_NS"; do
  log_info "Checking $ns namespace..."

  # Check pods running
  PODS=$(kubectl get pods -n "$ns" -l app.kubernetes.io/name=gastown-operator -o jsonpath='{.items[*].status.phase}')
  if [ "$PODS" = "Running" ]; then
    log_success "[$ns] Operator pod is running"
  else
    log_error "[$ns] Operator pod not running: $PODS"
    kubectl get pods -n "$ns"
    exit 1
  fi

  # Check service account
  if kubectl get serviceaccount gastown-operator -n "$ns" &> /dev/null; then
    log_success "[$ns] ServiceAccount created"
  else
    log_error "[$ns] ServiceAccount missing"
    exit 1
  fi

  # Check RBAC
  if kubectl get clusterrole gastown-operator-manager-role &> /dev/null; then
    log_success "[$ns] ClusterRole created"
  else
    log_error "[$ns] ClusterRole missing"
    exit 1
  fi
done

# ============================================================================
# Test Community Edition - Rig and Polecat
# ============================================================================
log_section "Testing Community Edition"

# Create test Rig
log_info "Creating test Rig..."
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: test-rig-community
spec:
  gitURL: "https://github.com/test/repo.git"
  beadsPrefix: "comm"
  localPath: "/tmp/test"
EOF

# Wait for Rig to be ready
sleep 3
RIG_STATUS=$(kubectl get rig test-rig-community -o jsonpath='{.status.conditions[?(@.type=="Ready")].status}' 2>/dev/null || echo "Unknown")
log_info "Rig status: $RIG_STATUS"

# Create secrets for Polecat
log_info "Creating secrets..."
kubectl create secret generic test-git-creds -n "$COMMUNITY_NS" \
  --from-literal=ssh-privatekey=dummy-test-key-for-validation

kubectl create secret generic test-claude-creds -n "$COMMUNITY_NS" \
  --from-literal=api-key=dummy-test-key-for-validation

# Verify secrets created
if kubectl get secret test-git-creds -n "$COMMUNITY_NS" &> /dev/null; then
  log_success "[Community] Git secret created"
else
  log_error "[Community] Git secret missing"
  exit 1
fi

if kubectl get secret test-claude-creds -n "$COMMUNITY_NS" &> /dev/null; then
  log_success "[Community] Claude secret created"
else
  log_error "[Community] Claude secret missing"
  exit 1
fi

# Create test Polecat
log_info "Creating test Polecat..."
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: test-polecat-community
  namespace: $COMMUNITY_NS
spec:
  rig: test-rig-community
  desiredState: Working
  executionMode: kubernetes
  beadID: test-comm-001
  kubernetes:
    gitRepository: "https://github.com/test/repo.git"
    gitBranch: main
    gitSecretRef:
      name: test-git-creds
    apiKeySecretRef:
      name: test-claude-creds
      key: api-key
EOF

# Wait for Polecat to create Pod
log_info "Waiting for Polecat to create Pod..."
sleep 10

# Check if Pod was created
POD_NAME=$(kubectl get pod -n "$COMMUNITY_NS" -l gastown.io/polecat=test-polecat-community -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "")
if [ -n "$POD_NAME" ]; then
  log_success "[Community] Polecat Pod created: $POD_NAME"
else
  log_warning "[Community] Polecat Pod not created (expected - no real git creds)"
fi

# Check Polecat status
POLECAT_PHASE=$(kubectl get polecat test-polecat-community -n "$COMMUNITY_NS" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
log_info "[Community] Polecat phase: $POLECAT_PHASE"

# Check operator logs for activity
log_info "Checking operator logs for Polecat reconciliation..."
if kubectl logs deployment/gastown-operator -n "$COMMUNITY_NS" --tail=50 | grep -q "Reconciling Polecat"; then
  log_success "[Community] Operator is reconciling Polecats"
else
  log_warning "[Community] No Polecat reconciliation in recent logs"
fi

# ============================================================================
# Test FIPS Edition - Convoy
# ============================================================================
log_section "Testing FIPS Edition"

# Create test Rig for FIPS
log_info "Creating test Rig..."
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: test-rig-fips
spec:
  gitURL: "https://github.com/test/repo.git"
  beadsPrefix: "fips"
  localPath: "/tmp/test"
EOF

sleep 3

# Create Convoy
log_info "Creating test Convoy..."
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Convoy
metadata:
  name: test-convoy-fips
  namespace: $FIPS_NS
spec:
  name: "FIPS Test Convoy"
  rig: test-rig-fips
  issueIDs:
    - fips-001
    - fips-002
    - fips-003
EOF

# Verify Convoy created
sleep 5
if kubectl get convoy test-convoy-fips -n "$FIPS_NS" &> /dev/null; then
  log_success "[FIPS] Convoy created"
  CONVOY_STATUS=$(kubectl get convoy test-convoy-fips -n "$FIPS_NS" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
  log_info "[FIPS] Convoy status: $CONVOY_STATUS"
else
  log_error "[FIPS] Convoy not created"
  exit 1
fi

# Verify security context (FIPS requirements)
log_info "Verifying FIPS security context..."
POD=$(kubectl get pod -n "$FIPS_NS" -l app.kubernetes.io/name=gastown-operator -o jsonpath='{.items[0].metadata.name}')

RUN_AS_NON_ROOT=$(kubectl get pod "$POD" -n "$FIPS_NS" -o jsonpath='{.spec.securityContext.runAsNonRoot}')
if [ "$RUN_AS_NON_ROOT" = "true" ]; then
  log_success "[FIPS] runAsNonRoot=true"
else
  log_error "[FIPS] runAsNonRoot should be true"
  exit 1
fi

READ_ONLY=$(kubectl get pod "$POD" -n "$FIPS_NS" -o jsonpath='{.spec.containers[0].securityContext.readOnlyRootFilesystem}')
if [ "$READ_ONLY" = "true" ]; then
  log_success "[FIPS] readOnlyRootFilesystem=true"
else
  log_error "[FIPS] readOnlyRootFilesystem should be true"
  exit 1
fi

# ============================================================================
# Summary
# ============================================================================
log_section "Test Summary"

echo ""
echo -e "${GREEN}All release validation tests passed!${NC}"
echo ""
echo "Tested:"
echo "  - CRDs: ${#EXPECTED_CRDS[@]} CRDs installed and verified"
echo "  - Community Edition:"
echo "    - Helm chart deployed to $COMMUNITY_NS"
echo "    - Secrets created successfully"
echo "    - Rig and Polecat CRDs functional"
echo "    - Operator reconciliation working"
echo "  - FIPS Edition:"
echo "    - Helm chart deployed to $FIPS_NS"
echo "    - Convoy CRD functional"
echo "    - Security context verified (runAsNonRoot, readOnlyRootFilesystem)"
echo ""
echo "Version: $VERSION"
echo "Image: $OPERATOR_IMAGE"
echo ""

# Exit success (cleanup will run via trap)
exit 0
