#!/bin/bash
# Cluster Test Script for Gas Town Operator
# Deploys and tests the operator in an existing Kubernetes/OpenShift cluster
#
# Prerequisites:
#   - kubectl configured for target cluster
#   - helm 3.x installed
#   - ~/.claude/ directory with credentials.json and settings.json
#   - SSH key for git operations (optional, for kubernetes mode testing)
#
# Usage:
#   ./scripts/cluster-test.sh [options]
#
# Options:
#   --namespace NS       Target namespace (default: gastown-system)
#   --image IMAGE        Operator image (default: ghcr.io/boshu2/gastown-operator:0.3.2)
#   --skip-secrets       Don't create secrets (use existing)
#   --skip-deploy        Don't deploy operator (use existing)
#   --cleanup            Remove all resources and exit
#   --test-polecat       Create and verify a test polecat CR
#   --dry-run            Print what would be done without executing
#   --verbose            Enable verbose output

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Defaults
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
NAMESPACE="gastown-system"
IMAGE="ghcr.io/boshu2/gastown-operator:0.3.2"
SKIP_SECRETS=false
SKIP_DEPLOY=false
CLEANUP=false
TEST_POLECAT=false
DRY_RUN=false
VERBOSE=false

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --namespace)
      NAMESPACE="$2"
      shift 2
      ;;
    --image)
      IMAGE="$2"
      shift 2
      ;;
    --skip-secrets)
      SKIP_SECRETS=true
      shift
      ;;
    --skip-deploy)
      SKIP_DEPLOY=true
      shift
      ;;
    --cleanup)
      CLEANUP=true
      shift
      ;;
    --test-polecat)
      TEST_POLECAT=true
      shift
      ;;
    --dry-run)
      DRY_RUN=true
      shift
      ;;
    --verbose)
      VERBOSE=true
      shift
      ;;
    -h|--help)
      head -25 "$0" | tail -22
      exit 0
      ;;
    *)
      echo -e "${RED}Unknown option: $1${NC}"
      exit 1
      ;;
  esac
done

log() {
  echo -e "${BLUE}[$(date '+%H:%M:%S')]${NC} $1"
}

success() {
  echo -e "${GREEN}✓${NC} $1"
}

warn() {
  echo -e "${YELLOW}⚠${NC} $1"
}

error() {
  echo -e "${RED}✗${NC} $1"
}

run() {
  if [ "$DRY_RUN" = true ]; then
    echo -e "${YELLOW}[DRY-RUN]${NC} $*"
  else
    if [ "$VERBOSE" = true ]; then
      echo -e "${BLUE}[RUN]${NC} $*"
    fi
    "$@"
  fi
}

# Check prerequisites
check_prereqs() {
  log "Checking prerequisites..."

  if ! command -v kubectl &> /dev/null; then
    error "kubectl not found"
    exit 1
  fi

  if ! command -v helm &> /dev/null; then
    error "helm not found"
    exit 1
  fi

  # Verify cluster access
  if ! kubectl cluster-info &> /dev/null; then
    error "Cannot connect to Kubernetes cluster"
    exit 1
  fi

  CONTEXT=$(kubectl config current-context)
  success "Connected to cluster: $CONTEXT"
}

# Cleanup resources
cleanup() {
  log "Cleaning up resources in namespace: $NAMESPACE"

  # Delete test polecat if exists
  run kubectl delete polecat test-worker -n "$NAMESPACE" --ignore-not-found=true

  # Uninstall helm release
  if helm status gastown-operator -n "$NAMESPACE" &> /dev/null; then
    run helm uninstall gastown-operator -n "$NAMESPACE"
    success "Helm release uninstalled"
  fi

  # Delete secrets
  run kubectl delete secret claude-creds -n "$NAMESPACE" --ignore-not-found=true
  run kubectl delete secret git-creds -n "$NAMESPACE" --ignore-not-found=true

  # Delete namespace (optional - commented out for safety)
  # run kubectl delete namespace "$NAMESPACE" --ignore-not-found=true

  success "Cleanup complete"
}

# Create namespace
create_namespace() {
  log "Creating namespace: $NAMESPACE"

  if kubectl get namespace "$NAMESPACE" &> /dev/null; then
    success "Namespace already exists"
  else
    run kubectl create namespace "$NAMESPACE"
    success "Namespace created"
  fi
}

# Create secrets
create_secrets() {
  log "Creating secrets..."

  # Claude credentials - try multiple sources
  # Priority: 1) API key env var, 2) macOS Keychain, 3) file-based

  CLAUDE_CREDS=""

  # Option 1: API key from environment
  if [ -n "${ANTHROPIC_API_KEY:-}" ]; then
    log "Using ANTHROPIC_API_KEY from environment"
    if kubectl get secret claude-creds -n "$NAMESPACE" &> /dev/null; then
      warn "Secret claude-creds already exists, updating..."
      run kubectl delete secret claude-creds -n "$NAMESPACE"
    fi
    run kubectl create secret generic claude-creds \
      --from-literal=api-key="$ANTHROPIC_API_KEY" \
      -n "$NAMESPACE"
    success "Created claude-creds secret (API key mode)"
    CLAUDE_CREDS="api-key"
  fi

  # Option 2: macOS Keychain (Claude Code OAuth tokens)
  if [ -z "$CLAUDE_CREDS" ] && command -v security &> /dev/null; then
    KEYCHAIN_CREDS=$(security find-generic-password -s "Claude Code-credentials" -w 2>/dev/null || true)
    if [ -n "$KEYCHAIN_CREDS" ]; then
      log "Found credentials in macOS Keychain"
      if kubectl get secret claude-creds -n "$NAMESPACE" &> /dev/null; then
        warn "Secret claude-creds already exists, updating..."
        run kubectl delete secret claude-creds -n "$NAMESPACE"
      fi
      run kubectl create secret generic claude-creds \
        --from-literal=.credentials.json="$KEYCHAIN_CREDS" \
        -n "$NAMESPACE"
      success "Created claude-creds secret (OAuth from Keychain)"
      CLAUDE_CREDS="oauth"
    fi
  fi

  # Option 3: File-based credentials
  if [ -z "$CLAUDE_CREDS" ]; then
    CLAUDE_DIR="$HOME/.claude"
    if [ -f "$CLAUDE_DIR/.credentials.json" ]; then
      log "Found file-based credentials at $CLAUDE_DIR"
      if kubectl get secret claude-creds -n "$NAMESPACE" &> /dev/null; then
        warn "Secret claude-creds already exists, updating..."
        run kubectl delete secret claude-creds -n "$NAMESPACE"
      fi
      run kubectl create secret generic claude-creds \
        --from-file=.credentials.json="$CLAUDE_DIR/.credentials.json" \
        -n "$NAMESPACE"
      success "Created claude-creds secret (file-based OAuth)"
      CLAUDE_CREDS="file"
    fi
  fi

  if [ -z "$CLAUDE_CREDS" ]; then
    error "No Claude credentials found!"
    error "Options:"
    error "  1. Set ANTHROPIC_API_KEY environment variable"
    error "  2. Run 'claude login' on macOS (stores in Keychain)"
    error "  3. Create ~/.claude/.credentials.json manually"
    exit 1
  fi

  # Git credentials (optional - for kubernetes mode)
  SSH_KEY="$HOME/.ssh/id_rsa"
  if [ -f "$SSH_KEY" ]; then
    if kubectl get secret git-creds -n "$NAMESPACE" &> /dev/null; then
      warn "Secret git-creds already exists, updating..."
      run kubectl delete secret git-creds -n "$NAMESPACE"
    fi

    run kubectl create secret generic git-creds \
      --from-file=ssh-privatekey="$SSH_KEY" \
      --type=kubernetes.io/ssh-auth \
      -n "$NAMESPACE"
    success "Created git-creds secret"
  else
    warn "SSH key not found at $SSH_KEY - git-creds secret not created"
    warn "Kubernetes execution mode will not work without git credentials"
  fi
}

# Deploy operator
deploy_operator() {
  log "Deploying operator..."

  # Extract image and tag
  IMAGE_REPO="${IMAGE%:*}"
  IMAGE_TAG="${IMAGE##*:}"

  # Check if already deployed
  if helm status gastown-operator -n "$NAMESPACE" &> /dev/null; then
    warn "Helm release already exists, upgrading..."
    ACTION="upgrade"
  else
    ACTION="install"
  fi

  # Deploy with helm
  run helm $ACTION gastown-operator "$PROJECT_ROOT/helm/gastown-operator" \
    -n "$NAMESPACE" \
    --set image.repository="$IMAGE_REPO" \
    --set image.tag="$IMAGE_TAG" \
    --set volumes.enabled=false \
    --wait --timeout 120s

  success "Operator deployed (image: $IMAGE)"

  # Wait for deployment
  log "Waiting for operator to be ready..."
  run kubectl rollout status deployment/gastown-operator -n "$NAMESPACE" --timeout=120s
  success "Operator is ready"
}

# Verify CRDs
verify_crds() {
  log "Verifying CRDs..."

  CRDS=("polecats.gastown.gastown.io" "convoys.gastown.gastown.io" "rigs.gastown.gastown.io"
        "refineries.gastown.gastown.io" "witnesses.gastown.gastown.io" "beadstores.gastown.gastown.io")

  ALL_FOUND=true
  for crd in "${CRDS[@]}"; do
    if kubectl get crd "$crd" &> /dev/null; then
      success "CRD: $crd"
    else
      error "CRD missing: $crd"
      ALL_FOUND=false
    fi
  done

  if [ "$ALL_FOUND" = false ]; then
    error "Some CRDs are missing"
    exit 1
  fi
}

# Test polecat CR
test_polecat() {
  log "Creating test polecat..."

  # Create test polecat CR
  cat <<EOF | run kubectl apply -f -
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: test-worker
  namespace: $NAMESPACE
spec:
  rig: test-rig
  beadID: go-test-001
  desiredState: Idle
  executionMode: local
EOF

  success "Test polecat created"

  # Wait for status
  log "Waiting for polecat status..."
  sleep 5

  # Check status
  STATUS=$(kubectl get polecat test-worker -n "$NAMESPACE" -o jsonpath='{.status.phase}' 2>/dev/null || echo "Unknown")
  log "Polecat status: $STATUS"

  # Show polecat details
  kubectl get polecat test-worker -n "$NAMESPACE" -o yaml

  success "Test polecat verification complete"
}

# Show status
show_status() {
  log "Current deployment status:"
  echo ""

  echo -e "${BLUE}=== Namespace ===${NC}"
  kubectl get namespace "$NAMESPACE" 2>/dev/null || echo "Namespace not found"
  echo ""

  echo -e "${BLUE}=== Helm Release ===${NC}"
  helm status gastown-operator -n "$NAMESPACE" 2>/dev/null || echo "Helm release not found"
  echo ""

  echo -e "${BLUE}=== Pods ===${NC}"
  kubectl get pods -n "$NAMESPACE" 2>/dev/null || echo "No pods"
  echo ""

  echo -e "${BLUE}=== Secrets ===${NC}"
  kubectl get secrets -n "$NAMESPACE" 2>/dev/null || echo "No secrets"
  echo ""

  echo -e "${BLUE}=== CRDs ===${NC}"
  kubectl get crd | grep gastown 2>/dev/null || echo "No gastown CRDs"
  echo ""

  echo -e "${BLUE}=== Polecats ===${NC}"
  kubectl get polecats -A 2>/dev/null || echo "No polecats"
}

# Main
main() {
  echo -e "${BLUE}========================================"
  echo "  Gas Town Operator - Cluster Test"
  echo "========================================${NC}"
  echo ""
  echo "Namespace: $NAMESPACE"
  echo "Image:     $IMAGE"
  echo ""

  check_prereqs

  if [ "$CLEANUP" = true ]; then
    cleanup
    exit 0
  fi

  # Create namespace
  create_namespace

  # Create secrets (unless skipped)
  if [ "$SKIP_SECRETS" = false ]; then
    create_secrets
  else
    log "Skipping secrets creation"
  fi

  # Deploy operator (unless skipped)
  if [ "$SKIP_DEPLOY" = false ]; then
    deploy_operator
    verify_crds
  else
    log "Skipping operator deployment"
  fi

  # Test polecat if requested
  if [ "$TEST_POLECAT" = true ]; then
    test_polecat
  fi

  # Show final status
  show_status

  echo ""
  success "Cluster test complete!"
  echo ""
  echo "Next steps:"
  echo "  - Create polecats: kubectl apply -f config/samples/polecat-kubernetes-sample.yaml"
  echo "  - View logs:       kubectl logs -f deployment/gastown-operator -n $NAMESPACE"
  echo "  - Cleanup:         $0 --cleanup"
}

main
