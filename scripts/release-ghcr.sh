#!/bin/bash
# Build and push multi-arch image to GHCR
#
# Usage:
#   ./scripts/release-ghcr.sh <version>
#   make release-ghcr VERSION=0.3.3
#
# Prerequisites:
#   - Docker with buildx (Docker Desktop includes this)
#   - GHCR login: echo $GITHUB_TOKEN | docker login ghcr.io -u boshu2 --password-stdin
#
# This builds locally on your Mac which is fast:
#   - arm64: native (~30s)
#   - amd64: Rosetta emulation (~1-2min)

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.3.3"
    exit 1
fi

REGISTRY="ghcr.io/boshu2"
IMAGE="gastown-operator"
PLATFORMS="linux/amd64,linux/arm64"

echo "========================================"
echo "  Multi-Arch Release to GHCR"
echo "========================================"
echo ""
echo "Version:   $VERSION"
echo "Platforms: $PLATFORMS"
echo "Image:     $REGISTRY/$IMAGE:$VERSION"
echo ""

# Check docker is available
if ! command -v docker &> /dev/null; then
    echo "ERROR: docker not found"
    exit 1
fi

# Check buildx is available
if ! docker buildx version &> /dev/null; then
    echo "ERROR: docker buildx not available"
    echo "Install Docker Desktop or enable buildx"
    exit 1
fi

# Check logged into GHCR (verify credentials file or try gh auth)
if ! grep -q "ghcr.io" ~/.docker/config.json 2>/dev/null; then
    echo "Not logged into GHCR. Run:"
    echo "  gh auth token | docker login ghcr.io -u boshu2 --password-stdin"
    exit 1
fi

# Create/use buildx builder
BUILDER_NAME="gastown-multiarch"
if ! docker buildx inspect $BUILDER_NAME &> /dev/null; then
    echo "Creating buildx builder: $BUILDER_NAME"
    docker buildx create --name $BUILDER_NAME --platform $PLATFORMS --use
else
    docker buildx use $BUILDER_NAME
fi

echo ""
echo "Building and pushing..."
echo ""

# Build and push multi-arch
docker buildx build --push \
    --platform $PLATFORMS \
    -t $REGISTRY/$IMAGE:$VERSION \
    -t $REGISTRY/$IMAGE:latest \
    .

echo ""
echo "========================================"
echo "  Release Complete"
echo "========================================"
echo ""
echo "Published:"
echo "  - $REGISTRY/$IMAGE:$VERSION"
echo "  - $REGISTRY/$IMAGE:latest"
echo ""
echo "Platforms: $PLATFORMS"
echo ""
echo "Verify with:"
echo "  docker manifest inspect $REGISTRY/$IMAGE:$VERSION"
