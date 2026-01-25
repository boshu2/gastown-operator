#!/bin/bash
# Build and release polecat-agent container image with SBOM and security scanning
#
# Usage:
#   ./scripts/release-polecat-agent.sh <version>
#   make release-polecat-agent VERSION=0.4.0
#
# Prerequisites:
#   - Docker with buildx
#   - Trivy (brew install trivy)
#   - GHCR login (gh auth token | docker login ghcr.io -u boshu2 --password-stdin)
#
# Outputs:
#   - Multi-arch image pushed to GHCR
#   - SBOM attached to image (buildx --sbom)
#   - Trivy scan results in dist/trivy-polecat-agent-<version>.json

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.4.0"
    exit 1
fi

REGISTRY="ghcr.io/boshu2"
IMAGE="polecat-agent"
PLATFORMS="linux/amd64,linux/arm64"
BUILDER_NAME="gastown-multiarch"
DOCKERFILE="images/polecat-agent/Dockerfile"
DIST_DIR="dist"

echo "========================================"
echo "  Polecat Agent Release"
echo "========================================"
echo ""
echo "Version:   $VERSION"
echo "Platforms: $PLATFORMS"
echo "Image:     $REGISTRY/$IMAGE:$VERSION"
echo ""

# Check prerequisites
command -v docker &> /dev/null || { echo "ERROR: docker not found"; exit 1; }
command -v trivy &> /dev/null || { echo "ERROR: trivy not found (brew install trivy)"; exit 1; }
docker buildx version &> /dev/null || { echo "ERROR: docker buildx not available"; exit 1; }

# Check GHCR auth
if ! grep -q "ghcr.io" ~/.docker/config.json 2>/dev/null; then
    echo "Not logged into GHCR. Run:"
    echo "  gh auth token | docker login ghcr.io -u boshu2 --password-stdin"
    exit 1
fi

# Create dist directory
mkdir -p "$DIST_DIR"

# Create/use buildx builder
if ! docker buildx inspect $BUILDER_NAME &> /dev/null; then
    echo "Creating buildx builder: $BUILDER_NAME"
    docker buildx create --name $BUILDER_NAME --platform $PLATFORMS --use
else
    docker buildx use $BUILDER_NAME
fi

echo ""
echo "[1/4] Building multi-arch image with SBOM..."
echo ""

# Build and push with SBOM and provenance attestations
docker buildx build \
    --platform $PLATFORMS \
    --sbom=true \
    --provenance=true \
    -t "$REGISTRY/$IMAGE:$VERSION" \
    -t "$REGISTRY/$IMAGE:latest" \
    -f "$DOCKERFILE" \
    --push \
    .

echo ""
echo "[2/4] Running Trivy vulnerability scan..."
echo ""

# Scan for vulnerabilities
TRIVY_OUTPUT="$DIST_DIR/trivy-$IMAGE-$VERSION.json"
trivy image \
    --format json \
    --output "$TRIVY_OUTPUT" \
    --severity HIGH,CRITICAL \
    "$REGISTRY/$IMAGE:$VERSION"

# Also output human-readable summary
echo ""
echo "Trivy Scan Summary:"
trivy image \
    --severity HIGH,CRITICAL \
    "$REGISTRY/$IMAGE:$VERSION" || true

echo ""
echo "[3/4] Extracting SBOM..."
echo ""

# Extract SBOM from the image (buildx attaches it automatically)
SBOM_OUTPUT="$DIST_DIR/sbom-$IMAGE-$VERSION.json"
docker buildx imagetools inspect "$REGISTRY/$IMAGE:$VERSION" --format '{{json .SBOM}}' > "$SBOM_OUTPUT" 2>/dev/null || {
    echo "Note: SBOM extraction requires newer buildx. SBOM is still attached to image."
    echo "{\"note\": \"SBOM attached to image attestation\"}" > "$SBOM_OUTPUT"
}

echo ""
echo "[4/4] Generating version manifest..."
echo ""

# Create version manifest with all component versions
MANIFEST_OUTPUT="$DIST_DIR/manifest-$IMAGE-$VERSION.json"
cat > "$MANIFEST_OUTPUT" << EOF
{
  "image": "$REGISTRY/$IMAGE:$VERSION",
  "version": "$VERSION",
  "platforms": ["linux/amd64", "linux/arm64"],
  "built": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "components": {
    "claude_code": "$(grep 'ARG CLAUDE_CODE_VERSION=' $DOCKERFILE | cut -d= -f2)",
    "gt_cli": "$(grep 'ARG GT_VERSION=' $DOCKERFILE | cut -d= -f2)",
    "base_image": "debian:$(grep 'ARG DEBIAN_VERSION=' $DOCKERFILE | cut -d= -f2)"
  },
  "security": {
    "trivy_scan": "$TRIVY_OUTPUT",
    "sbom": "$SBOM_OUTPUT"
  }
}
EOF

echo ""
echo "========================================"
echo "  Polecat Agent Release Complete"
echo "========================================"
echo ""
echo "Published:"
echo "  - $REGISTRY/$IMAGE:$VERSION"
echo "  - $REGISTRY/$IMAGE:latest"
echo ""
echo "Platforms: $PLATFORMS"
echo ""
echo "Security Artifacts:"
echo "  - Trivy scan: $TRIVY_OUTPUT"
echo "  - SBOM: $SBOM_OUTPUT"
echo "  - Manifest: $MANIFEST_OUTPUT"
echo ""
echo "Verify with:"
echo "  docker manifest inspect $REGISTRY/$IMAGE:$VERSION"
echo "  trivy image $REGISTRY/$IMAGE:$VERSION"
echo ""
