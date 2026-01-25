#!/bin/bash
# Build multi-arch images locally (no push yet)
#
# Usage:
#   ./scripts/build-local.sh [version]
#   make ci-build
#
# This builds locally using docker buildx but does NOT push.
# Run 'make ci-push' to push after successful build.

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    VERSION=$(cat VERSION 2>/dev/null || echo "0.0.0")
fi

PLATFORMS="linux/amd64,linux/arm64"
BUILDER_NAME="gastown-multiarch"
REGISTRY="ghcr.io/boshu2"
IMAGE="gastown-operator"

echo "========================================"
echo "  Building Multi-Arch Image"
echo "========================================"
echo ""
echo "Version:   $VERSION"
echo "Platforms: $PLATFORMS"
echo "Image:     $REGISTRY/$IMAGE:$VERSION (local)"
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

# Create/use buildx builder
if ! docker buildx inspect $BUILDER_NAME &> /dev/null; then
    echo "Creating buildx builder: $BUILDER_NAME"
    docker buildx create --name $BUILDER_NAME --platform $PLATFORMS --use
else
    docker buildx use $BUILDER_NAME
fi

echo ""
echo "Building (local cache, no push)..."
echo ""

# Build multi-arch with local cache
# Uses --load is not supported for multi-platform, so we use --output=type=cacheonly
# The image will be pushed separately by ci-push
docker buildx build \
    --platform $PLATFORMS \
    -t $REGISTRY/$IMAGE:$VERSION \
    -t $REGISTRY/$IMAGE:latest \
    --cache-to type=local,dest=/tmp/buildx-cache,mode=max \
    --cache-from type=local,src=/tmp/buildx-cache \
    .

echo ""
echo "========================================"
echo "  Build Complete"
echo "========================================"
echo ""
echo "Built: $REGISTRY/$IMAGE:$VERSION for $PLATFORMS"
echo ""
echo "Next: Run 'make ci-push' to push to GHCR"
echo ""
