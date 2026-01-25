#!/bin/bash
# Create GitHub release using gh CLI
#
# Usage:
#   ./scripts/github-release.sh <version>
#   make ci-publish
#
# Prerequisites:
#   - gh CLI authenticated (gh auth login)
#   - Images already pushed to GHCR (make ci-push)

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.3.3"
    exit 1
fi

TAG="v$VERSION"
REPO="boshu2/gastown-operator"
REGISTRY="ghcr.io/boshu2"

echo "========================================"
echo "  Creating GitHub Release"
echo "========================================"
echo ""
echo "Version: $VERSION"
echo "Tag:     $TAG"
echo "Repo:    $REPO"
echo ""

# Check gh is available
if ! command -v gh &> /dev/null; then
    echo "ERROR: gh CLI not found"
    echo "Install: brew install gh"
    exit 1
fi

# Check gh is authenticated
if ! gh auth status &> /dev/null; then
    echo "ERROR: gh CLI not authenticated"
    echo "Run: gh auth login"
    exit 1
fi

# Create release body
BODY=$(cat <<EOF
## Release $TAG

### Installation

**Container Image (multi-arch: amd64 + arm64):**
\`\`\`bash
docker pull $REGISTRY/gastown-operator:$VERSION
\`\`\`

**Helm Chart:**
\`\`\`bash
helm install gastown-operator oci://$REGISTRY/charts/gastown-operator \\
  --version $VERSION \\
  -n gastown-system --create-namespace
\`\`\`

### Artifacts

| Artifact | Location |
|----------|----------|
| Image | \`$REGISTRY/gastown-operator:$VERSION\` |
| Helm | \`oci://$REGISTRY/charts/gastown-operator:$VERSION\` |

---
*Released via local Make CI/CD pipeline.*
EOF
)

# Tag if not exists
echo "Creating tag $TAG..."
git tag -a "$TAG" -m "Release $TAG" 2>/dev/null || echo "Tag $TAG already exists"

# Push tag to origin (GitLab)
echo "Pushing tag to origin..."
git push origin "$TAG" 2>/dev/null || echo "Tag already pushed to origin"

# Push to GitHub mirror
echo "Pushing to GitHub mirror..."
if ! git remote get-url github &> /dev/null; then
    echo "Adding github remote..."
    git remote add github "git@github.com:$REPO.git"
fi
git push github main 2>/dev/null || echo "main already up to date on GitHub"
git push github "$TAG" 2>/dev/null || echo "Tag already pushed to GitHub"

# Create release
echo "Creating GitHub release..."
gh release create "$TAG" \
    --title "$TAG" \
    --notes "$BODY" \
    --repo "$REPO" \
    2>/dev/null || {
        echo "Release $TAG already exists, updating..."
        gh release edit "$TAG" \
            --title "$TAG" \
            --notes "$BODY" \
            --repo "$REPO"
    }

echo ""
echo "========================================"
echo "  GitHub Release Complete"
echo "========================================"
echo ""
echo "Release: https://github.com/$REPO/releases/tag/$TAG"
echo ""
