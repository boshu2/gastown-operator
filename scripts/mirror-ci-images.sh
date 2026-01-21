#!/bin/bash
# Mirror CI images to internal DPR registry using skopeo
#
# Usage: ./scripts/mirror-ci-images.sh [target-registry]
# Example: ./scripts/mirror-ci-images.sh dprusocplvjmp01.deepsky.lab:5000
#
# Environment Variables:
#   DEST_TLS_VERIFY - Enable TLS verification (default: false for internal registry)
#
# Exit Codes:
#   0 - Success
#   1 - Argument error
#   2 - Missing dependency
#   3 - Partial failure (some images failed)

set -euo pipefail

trap 'echo "Error on line $LINENO. Exit code: $?" >&2' ERR

# ============================================================================
# IMAGE VERSIONS - Keep in sync with .gitlab-ci.yml
# ============================================================================

VERSION_GO="1.25"
VERSION_GOLANGCI_LINT="v2.8-alpine"
VERSION_YAMLLINT="latest"
VERSION_KUBECTL="1.32.11"

# ============================================================================
# CONFIGURATION
# ============================================================================

TARGET_REGISTRY="${1:-dprusocplvjmp01.deepsky.lab:5000}"
DEST_TLS_VERIFY="${DEST_TLS_VERIFY:-false}"

# ============================================================================
# IMAGE LIST - Format: source-image|target-path
# ============================================================================

IMAGES=(
  # Go build images
  "docker.io/library/golang:${VERSION_GO}|library/golang:${VERSION_GO}"

  # Linting
  "docker.io/golangci/golangci-lint:${VERSION_GOLANGCI_LINT}|golangci/golangci-lint:${VERSION_GOLANGCI_LINT}"
  "docker.io/cytopia/yamllint:${VERSION_YAMLLINT}|cytopia/yamllint:${VERSION_YAMLLINT}"

  # Kubernetes
  "docker.io/alpine/k8s:${VERSION_KUBECTL}|alpine/k8s:${VERSION_KUBECTL}"
)

# ============================================================================
# FUNCTIONS
# ============================================================================

mirror_image() {
  local source="$1"
  local target_path="$2"
  local target="docker://${TARGET_REGISTRY}/${target_path}"

  echo "Mirroring: $source"
  echo "       -> $target"

  local tls_flag=""
  if [[ "$DEST_TLS_VERIFY" == "false" ]]; then
    tls_flag="--dest-tls-verify=false"
  fi

  if skopeo copy --all $tls_flag "docker://${source}" "$target" 2>&1; then
    echo "  ✓ Success"
  else
    echo "  ✗ Failed"
    return 1
  fi
  echo ""
}

# ============================================================================
# MAIN
# ============================================================================

echo "============================================"
echo "Labyrinth CI Image Mirror"
echo "Target Registry: $TARGET_REGISTRY"
echo "TLS Verify: $DEST_TLS_VERIFY"
echo "============================================"
echo ""
echo "Images to mirror:"
echo "  golang:${VERSION_GO}"
echo "  golangci-lint:${VERSION_GOLANGCI_LINT}"
echo "  yamllint:${VERSION_YAMLLINT}"
echo "  kubectl:${VERSION_KUBECTL}"
echo ""

# Check skopeo is available
if ! command -v skopeo &> /dev/null; then
  echo "ERROR: skopeo not found. Install with: brew install skopeo" >&2
  exit 2
fi

# Mirror all images
FAILED=()
for entry in "${IMAGES[@]}"; do
  source="${entry%%|*}"
  target_path="${entry##*|}"

  if ! mirror_image "$source" "$target_path"; then
    FAILED+=("$source")
  fi
done

echo "============================================"
echo "Summary"
echo "============================================"
if [[ ${#FAILED[@]} -eq 0 ]]; then
  echo "All images mirrored successfully!"
  echo ""
  echo "Pipeline should now work with DPR mirror."
else
  echo "Failed images:" >&2
  for img in "${FAILED[@]}"; do
    echo "  - $img" >&2
  done
  exit 3
fi
