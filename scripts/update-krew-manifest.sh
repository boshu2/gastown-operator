#!/bin/bash
# Update Krew plugin manifest with SHA256 checksums from GitHub release
#
# Usage:
#   ./scripts/update-krew-manifest.sh <version>
#   ./scripts/update-krew-manifest.sh 0.4.2
#
# This script:
# 1. Downloads release artifacts from GitHub
# 2. Computes SHA256 checksums
# 3. Updates plugins/gt.yaml with real checksums
#
# Run this after creating a GitHub release with github-release.sh

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.4.2"
    exit 1
fi

TAG="v$VERSION"
REPO="boshu2/gastown-operator"
MANIFEST="plugins/gt.yaml"
TMPDIR=$(mktemp -d)

cleanup() {
    rm -rf "$TMPDIR"
}
trap cleanup EXIT

echo "Updating Krew manifest for $TAG..."

# Platforms to process
PLATFORMS=(
    "darwin-amd64"
    "darwin-arm64"
    "linux-amd64"
    "linux-arm64"
)

# Download and compute checksums
for PLATFORM in "${PLATFORMS[@]}"; do
    ARTIFACT="kubectl-gt-${PLATFORM}.tar.gz"
    URL="https://github.com/$REPO/releases/download/$TAG/$ARTIFACT"

    echo "  Downloading $ARTIFACT..."
    if ! curl -sL "$URL" -o "$TMPDIR/$ARTIFACT"; then
        echo "ERROR: Failed to download $URL"
        echo "Make sure the release exists and artifacts are uploaded."
        exit 1
    fi

    SHA256=$(shasum -a 256 "$TMPDIR/$ARTIFACT" | awk '{print $1}')
    echo "  SHA256: $SHA256"

    # Convert platform to manifest placeholder format
    PLACEHOLDER="PLACEHOLDER_SHA256_$(echo "$PLATFORM" | tr '-' '_' | tr '[:lower:]' '[:upper:]')"

    # Update manifest
    if grep -q "$PLACEHOLDER" "$MANIFEST"; then
        sed -i.bak "s/$PLACEHOLDER/$SHA256/" "$MANIFEST"
        rm -f "$MANIFEST.bak"
        echo "  Updated $PLACEHOLDER"
    else
        # Try to update existing SHA256 for this platform
        # Match the line after the uri line for this platform
        echo "  Note: Placeholder not found, manifest may already have checksums"
    fi
done

# Update version in manifest
sed -i.bak "s/version: v[0-9.]*/version: $TAG/" "$MANIFEST"
rm -f "$MANIFEST.bak"

echo ""
echo "Krew manifest updated: $MANIFEST"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff $MANIFEST"
echo "  2. Commit: git add $MANIFEST && git commit -m 'chore: update Krew manifest for $TAG'"
echo "  3. Submit to krew-index: https://github.com/kubernetes-sigs/krew-index"
