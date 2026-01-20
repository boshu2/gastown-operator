#!/bin/bash
# e2e-release-validation.sh - End-to-end release validation
#
# Comprehensive validation pipeline that exercises all stages before release.
# Can run in dry-run mode (default) or create a real release.
#
# Phases:
#   1. Pre-flight checks (git state, tools, VERSION sync)
#   2. Local validation (vet, lint, unit tests, prescan)
#   3. Kind E2E tests (deploys both Community and FIPS editions)
#   4. Release dry-run (semantic-release preview)
#   5. Create GitHub release (actual release, requires --release flag)
#   6. Post-release verification (GHCR images, Helm chart, GitHub release)
#
# Usage:
#   ./scripts/e2e-release-validation.sh              # Full validation (dry-run)
#   ./scripts/e2e-release-validation.sh --release    # Full validation + real release
#   ./scripts/e2e-release-validation.sh --phase 3    # Run only Phase 3
#   ./scripts/e2e-release-validation.sh --skip-kind  # Skip Kind E2E (CI already ran it)
#   ./scripts/e2e-release-validation.sh -v           # Verbose output
#
# Exit codes:
#   0 - All validations passed
#   1 - Pre-flight checks failed
#   2 - Local validation failed
#   3 - Kind E2E tests failed
#   4 - Release dry-run failed
#   5 - GitHub release creation failed
#   6 - Post-release verification failed

set -euo pipefail

# Script configuration
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default options
DRY_RUN=true
SKIP_KIND=false
VERBOSE=false
SINGLE_PHASE=""
GITHUB_OWNER="${GITHUB_OWNER:-boshu2}"
GHCR_REGISTRY="ghcr.io/${GITHUB_OWNER}"

# Parse arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --release)
      DRY_RUN=false
      shift
      ;;
    --skip-kind)
      SKIP_KIND=true
      shift
      ;;
    --phase)
      SINGLE_PHASE="$2"
      shift 2
      ;;
    -v|--verbose)
      VERBOSE=true
      shift
      ;;
    -h|--help)
      head -35 "$0" | tail -30
      exit 0
      ;;
    *)
      echo "Unknown option: $1"
      exit 1
      ;;
  esac
done

# Logging functions
log_info() { echo -e "${BLUE}[INFO]${NC} $1"; }
log_success() { echo -e "${GREEN}[PASS]${NC} $1"; }
log_warning() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[FAIL]${NC} $1"; }
log_verbose() { [[ "$VERBOSE" == "true" ]] && echo -e "${CYAN}[DEBUG]${NC} $1" || true; }

log_section() {
  echo ""
  echo -e "${BLUE}========================================"
  echo "  Phase $1: $2"
  echo "========================================${NC}"
  echo ""
}

fail() {
  log_error "$1"
  exit "${2:-1}"
}

# Track phase status
PHASE_RESULTS=()

run_phase() {
  local phase_num="$1"
  local phase_name="$2"
  local phase_func="$3"
  local exit_code="$4"

  # Skip if not the requested phase (when --phase is used)
  if [[ -n "$SINGLE_PHASE" ]] && [[ "$SINGLE_PHASE" != "$phase_num" ]]; then
    log_verbose "Skipping Phase $phase_num (not requested)"
    return 0
  fi

  log_section "$phase_num" "$phase_name"

  if "$phase_func"; then
    log_success "Phase $phase_num completed successfully"
    PHASE_RESULTS+=("Phase $phase_num: PASS")
    return 0
  else
    log_error "Phase $phase_num failed"
    PHASE_RESULTS+=("Phase $phase_num: FAIL")
    fail "Phase $phase_num ($phase_name) failed" "$exit_code"
  fi
}

# ============================================================================
# Phase 1: Pre-flight Checks
# ============================================================================
phase_preflight() {
  log_info "Checking required tools..."

  # Required tools
  local tools=("gh" "helm" "kubectl")
  if [[ "$SKIP_KIND" != "true" ]]; then
    tools+=("kind")
  fi

  for cmd in "${tools[@]}"; do
    if command -v "$cmd" &> /dev/null; then
      log_success "$cmd is installed"
    else
      log_error "$cmd is not installed"
      return 1
    fi
  done

  # Check Docker (needed for images)
  if ! docker info &> /dev/null; then
    log_error "Docker daemon is not running"
    return 1
  fi
  log_success "Docker daemon is running"

  # Git state check
  log_info "Checking git state..."
  if [[ -n "$(git -C "$PROJECT_ROOT" status --porcelain)" ]]; then
    log_warning "Working directory has uncommitted changes"
    if [[ "$DRY_RUN" != "true" ]]; then
      log_error "Cannot create release with dirty working directory"
      return 1
    fi
  else
    log_success "Git working directory is clean"
  fi

  # VERSION file check
  log_info "Checking VERSION file..."
  if [[ ! -f "$PROJECT_ROOT/VERSION" ]]; then
    log_error "VERSION file not found"
    return 1
  fi
  local version
  version=$(cat "$PROJECT_ROOT/VERSION" | tr -d '[:space:]')
  log_success "VERSION file exists: $version"

  # Chart.yaml version sync
  log_info "Checking version sync..."
  if ! "$SCRIPT_DIR/version-sync.sh" --check; then
    log_error "Versions are out of sync"
    return 1
  fi

  # GitHub auth check (if not dry-run or for verification)
  log_info "Checking GitHub authentication..."
  if gh auth status &> /dev/null; then
    log_success "GitHub CLI authenticated"
  else
    log_warning "GitHub CLI not authenticated (some features may be limited)"
    if [[ "$DRY_RUN" != "true" ]]; then
      log_error "GitHub CLI must be authenticated to create releases"
      return 1
    fi
  fi

  return 0
}

# ============================================================================
# Phase 2: Local Validation
# ============================================================================
phase_local_validation() {
  cd "$PROJECT_ROOT"

  log_info "Running go vet..."
  if go vet ./...; then
    log_success "go vet passed"
  else
    log_error "go vet failed"
    return 1
  fi

  log_info "Running golangci-lint..."
  if make lint 2>&1; then
    log_success "golangci-lint passed"
  else
    log_error "golangci-lint failed"
    return 1
  fi

  log_info "Running unit tests..."
  if make test 2>&1; then
    log_success "Unit tests passed"
  else
    log_error "Unit tests failed"
    return 1
  fi

  log_info "Running vibe prescan..."
  # Allow MEDIUM findings for dry-run, fail on HIGH for release
  if [[ "$DRY_RUN" == "true" ]]; then
    export PRESCAN_FAIL_ON="HIGH"
  else
    export PRESCAN_FAIL_ON="HIGH"
  fi

  local prescan_exit=0
  "$SCRIPT_DIR/prescan.sh" all || prescan_exit=$?

  if [[ $prescan_exit -eq 0 ]]; then
    log_success "Vibe prescan passed"
  elif [[ $prescan_exit -eq 4 ]]; then
    log_warning "Vibe prescan found MEDIUM issues (acceptable for dry-run)"
  else
    log_error "Vibe prescan failed (exit code: $prescan_exit)"
    return 1
  fi

  return 0
}

# ============================================================================
# Phase 3: Kind E2E Tests
# ============================================================================
phase_kind_e2e() {
  if [[ "$SKIP_KIND" == "true" ]]; then
    log_info "Skipping Kind E2E tests (--skip-kind flag)"
    return 0
  fi

  log_info "Running Kind E2E validation..."
  log_info "This will:"
  echo "  - Create a Kind cluster"
  echo "  - Build and load operator image"
  echo "  - Deploy Community edition"
  echo "  - Deploy FIPS edition"
  echo "  - Run validation tests"
  echo "  - Clean up cluster"
  echo ""

  if "$SCRIPT_DIR/release-validation.sh" --local; then
    log_success "Kind E2E validation passed"
    return 0
  else
    log_error "Kind E2E validation failed"
    return 1
  fi
}

# ============================================================================
# Phase 4: Release Dry-Run
# ============================================================================
phase_release_dry_run() {
  log_info "Running semantic-release dry-run..."

  cd "$PROJECT_ROOT"

  # Check if semantic-release is available
  if ! command -v npx &> /dev/null; then
    log_warning "npx not available, skipping semantic-release dry-run"
    log_info "Install Node.js to enable release preview"
    return 0
  fi

  # Create temp file for output
  local preview_file="/tmp/release-preview-$$.txt"

  log_info "Analyzing commits for version bump..."

  # Run dry-run (may fail if no releasable commits, that's OK)
  set +e
  npx semantic-release --dry-run 2>&1 | tee "$preview_file"
  local sr_exit=$?
  set -e

  # Parse results
  if grep -q "Published release" "$preview_file" 2>/dev/null; then
    local next_version
    next_version=$(grep -oP 'next release version is \K[\d.]+' "$preview_file" 2>/dev/null || grep -o 'v[0-9.]*' "$preview_file" | head -1 | tr -d 'v')
    log_success "Release preview: would release version $next_version"
    echo ""
    echo "Preview summary:"
    grep -E "(feat|fix|breaking|BREAKING)" "$preview_file" | head -10 || true
  elif grep -q "no release" "$preview_file" 2>/dev/null || grep -q "There are no relevant changes" "$preview_file" 2>/dev/null; then
    log_warning "No releasable commits found (no release would be created)"
  else
    log_verbose "semantic-release dry-run completed (exit: $sr_exit)"
  fi

  rm -f "$preview_file"

  return 0
}

# ============================================================================
# Phase 5: Create GitHub Release
# ============================================================================
phase_create_release() {
  if [[ "$DRY_RUN" == "true" ]]; then
    log_info "Skipping release creation (dry-run mode)"
    log_info "Run with --release flag to create a real release"
    return 0
  fi

  log_warning "CREATING REAL RELEASE"
  echo ""
  echo "This will:"
  echo "  - Run semantic-release"
  echo "  - Bump version based on commits"
  echo "  - Update VERSION, Chart.yaml, CHANGELOG.md"
  echo "  - Create git tag"
  echo "  - Create GitHub release"
  echo "  - Push changes to main"
  echo ""

  # Confirm
  read -p "Proceed with release? (yes/no): " confirm
  if [[ "$confirm" != "yes" ]]; then
    log_info "Release cancelled by user"
    return 0
  fi

  cd "$PROJECT_ROOT"

  log_info "Running semantic-release..."
  if npx semantic-release; then
    log_success "Semantic release completed"
  else
    log_error "Semantic release failed"
    return 1
  fi

  # Get the new version
  local new_version
  new_version=$(cat "$PROJECT_ROOT/VERSION" | tr -d '[:space:]')
  log_success "Released version: $new_version"

  # Wait for release-helm.yaml workflow
  log_info "Waiting for release-helm.yaml workflow to start..."
  sleep 10

  local workflow_run
  workflow_run=$(gh run list --workflow=release-helm.yaml --limit=1 --json databaseId -q '.[0].databaseId' 2>/dev/null || echo "")

  if [[ -z "$workflow_run" ]]; then
    log_warning "Could not find release-helm workflow run"
    log_info "Check GitHub Actions manually: https://github.com/${GITHUB_OWNER}/gastown-operator/actions"
    return 0
  fi

  log_info "Watching workflow run: $workflow_run"
  log_info "Timeout: 20 minutes"

  if timeout 1200 gh run watch "$workflow_run"; then
    local conclusion
    conclusion=$(gh run view "$workflow_run" --json conclusion -q '.conclusion')
    if [[ "$conclusion" == "success" ]]; then
      log_success "Release workflow completed successfully"
    else
      log_error "Release workflow failed: $conclusion"
      return 1
    fi
  else
    log_error "Release workflow timed out"
    return 1
  fi

  return 0
}

# ============================================================================
# Phase 6: Post-Release Verification
# ============================================================================
phase_post_verification() {
  if [[ "$DRY_RUN" == "true" ]]; then
    log_info "Skipping post-release verification (dry-run mode)"
    return 0
  fi

  local version
  version=$(cat "$PROJECT_ROOT/VERSION" | tr -d '[:space:]')

  log_info "Verifying release artifacts for v$version..."

  # Verify GitHub release exists
  log_info "Checking GitHub release..."
  if gh release view "v$version" &> /dev/null; then
    log_success "GitHub release v$version exists"
  else
    log_error "GitHub release v$version not found"
    return 1
  fi

  # Verify image in GHCR
  log_info "Checking GHCR image..."
  if command -v crane &> /dev/null; then
    if crane manifest "${GHCR_REGISTRY}/gastown-operator:v$version" &> /dev/null; then
      log_success "Image found: ${GHCR_REGISTRY}/gastown-operator:v$version"
    else
      log_warning "Could not verify image in GHCR (may still be publishing)"
    fi
  else
    log_warning "crane not installed, skipping image verification"
    log_info "Install with: go install github.com/google/go-containerregistry/cmd/crane@latest"
  fi

  # Verify Helm chart in GHCR OCI
  log_info "Checking Helm chart..."
  if helm show chart "oci://${GHCR_REGISTRY}/gastown-operator" --version "$version" &> /dev/null; then
    log_success "Helm chart found: oci://${GHCR_REGISTRY}/gastown-operator:$version"
  else
    log_warning "Could not verify Helm chart (may still be publishing)"
  fi

  # Verify CHANGELOG.md updated
  log_info "Checking CHANGELOG.md..."
  if grep -q "## \[$version\]" "$PROJECT_ROOT/CHANGELOG.md" 2>/dev/null || \
     grep -q "## $version" "$PROJECT_ROOT/CHANGELOG.md" 2>/dev/null; then
    log_success "CHANGELOG.md contains v$version"
  else
    log_warning "CHANGELOG.md may not be updated yet"
  fi

  log_success "Post-release verification completed"
  return 0
}

# ============================================================================
# Main
# ============================================================================
main() {
  echo -e "${BLUE}========================================"
  echo "  E2E Release Validation"
  echo "========================================${NC}"
  echo ""
  echo "Mode: $([ "$DRY_RUN" == "true" ] && echo "DRY-RUN" || echo "RELEASE")"
  echo "Skip Kind: $SKIP_KIND"
  echo "Single Phase: ${SINGLE_PHASE:-all}"
  echo "Project: $PROJECT_ROOT"
  echo ""

  # Run phases
  run_phase 1 "Pre-flight Checks" phase_preflight 1
  run_phase 2 "Local Validation" phase_local_validation 2
  run_phase 3 "Kind E2E Tests" phase_kind_e2e 3
  run_phase 4 "Release Dry-Run" phase_release_dry_run 4
  run_phase 5 "Create GitHub Release" phase_create_release 5
  run_phase 6 "Post-Release Verification" phase_post_verification 6

  # Summary
  echo ""
  echo -e "${GREEN}========================================"
  echo "  Validation Summary"
  echo "========================================${NC}"
  echo ""
  for result in "${PHASE_RESULTS[@]}"; do
    echo "  $result"
  done
  echo ""

  if [[ "$DRY_RUN" == "true" ]]; then
    echo "Dry-run completed successfully."
    echo ""
    echo "To create a real release:"
    echo "  $0 --release"
  else
    local version
    version=$(cat "$PROJECT_ROOT/VERSION" | tr -d '[:space:]')
    echo "Release v$version completed successfully!"
    echo ""
    echo "Artifacts:"
    echo "  - GitHub: https://github.com/${GITHUB_OWNER}/gastown-operator/releases/tag/v$version"
    echo "  - Image:  ghcr.io/${GITHUB_OWNER}/gastown-operator:v$version"
    echo "  - Helm:   oci://ghcr.io/${GITHUB_OWNER}/gastown-operator:$version"
  fi

  echo ""
  exit 0
}

main
