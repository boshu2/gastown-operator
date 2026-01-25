#!/bin/bash
# Create GitHub release with kubectl-gt binaries
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
    echo "Example: $0 0.4.1"
    exit 1
fi

TAG="v$VERSION"
REPO="boshu2/gastown-operator"
REGISTRY="ghcr.io/boshu2"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

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

# Build kubectl-gt binaries
echo "Building kubectl-gt binaries..."
cd "$PROJECT_ROOT/cmd/kubectl-gt"
make clean

# Cross-compile for all platforms
LDFLAGS="-ldflags \"-X github.com/olympus-cloud/gastown-operator/cmd/kubectl-gt/cmd.Version=$VERSION\""

echo "  Building linux-amd64..."
GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/olympus-cloud/gastown-operator/cmd/kubectl-gt/cmd.Version=$VERSION" -o bin/kubectl-gt-linux-amd64 .

echo "  Building linux-arm64..."
GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/olympus-cloud/gastown-operator/cmd/kubectl-gt/cmd.Version=$VERSION" -o bin/kubectl-gt-linux-arm64 .

echo "  Building darwin-amd64..."
GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/olympus-cloud/gastown-operator/cmd/kubectl-gt/cmd.Version=$VERSION" -o bin/kubectl-gt-darwin-amd64 .

echo "  Building darwin-arm64..."
GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/olympus-cloud/gastown-operator/cmd/kubectl-gt/cmd.Version=$VERSION" -o bin/kubectl-gt-darwin-arm64 .

echo "  Building windows-amd64..."
GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/olympus-cloud/gastown-operator/cmd/kubectl-gt/cmd.Version=$VERSION" -o bin/kubectl-gt-windows-amd64.exe .

cd "$PROJECT_ROOT"

# Create release body
BODY=$(cat <<EOF
## Release $TAG - kubectl-gt AI-Native CLI

### Highlights

- **kubectl-gt CLI**: AI-native kubectl plugin for managing Gas Town resources
- **JSON/YAML output**: Machine-parseable output for automation and scripting
- **Themed polecat naming**: Mad Max, minerals, and wasteland themes for memorable names
- **Native log streaming**: Direct pod log streaming via client-go
- **Wait for ready**: Block until polecat pod is running with \`--wait-ready\`

### Installation

**kubectl-gt Plugin (New!):**
\`\`\`bash
# macOS (Apple Silicon)
curl -LO https://github.com/$REPO/releases/download/$TAG/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt

# macOS (Intel)
curl -LO https://github.com/$REPO/releases/download/$TAG/kubectl-gt-darwin-amd64
chmod +x kubectl-gt-darwin-amd64 && sudo mv kubectl-gt-darwin-amd64 /usr/local/bin/kubectl-gt

# Linux
curl -LO https://github.com/$REPO/releases/download/$TAG/kubectl-gt-linux-amd64
chmod +x kubectl-gt-linux-amd64 && sudo mv kubectl-gt-linux-amd64 /usr/local/bin/kubectl-gt
\`\`\`

**Helm Chart:**
\`\`\`bash
helm install gastown-operator oci://$REGISTRY/charts/gastown-operator \\
  --version $VERSION \\
  -n gastown-system --create-namespace
\`\`\`

### Quick Start

\`\`\`bash
# Set up credentials
kubectl create secret generic git-creds -n gastown-system \\
  --from-file=ssh-privatekey=\$HOME/.ssh/id_ed25519
kubectl gt auth sync -n gastown-system

# Create rig and dispatch work
kubectl gt rig create my-project --git-url git@github.com:org/repo.git --prefix mp
kubectl gt sling issue-123 my-project --theme mad-max
kubectl gt polecat logs my-project/furiosa -f
\`\`\`

### Artifacts

| Artifact | Location |
|----------|----------|
| kubectl-gt (darwin-arm64) | [Download](https://github.com/$REPO/releases/download/$TAG/kubectl-gt-darwin-arm64) |
| kubectl-gt (darwin-amd64) | [Download](https://github.com/$REPO/releases/download/$TAG/kubectl-gt-darwin-amd64) |
| kubectl-gt (linux-amd64) | [Download](https://github.com/$REPO/releases/download/$TAG/kubectl-gt-linux-amd64) |
| kubectl-gt (linux-arm64) | [Download](https://github.com/$REPO/releases/download/$TAG/kubectl-gt-linux-arm64) |
| kubectl-gt (windows-amd64) | [Download](https://github.com/$REPO/releases/download/$TAG/kubectl-gt-windows-amd64.exe) |
| Container Image | \`$REGISTRY/gastown-operator:$VERSION\` |
| Helm Chart | \`oci://$REGISTRY/charts/gastown-operator:$VERSION\` |

---
*Released via local Make CI/CD pipeline.*
EOF
)

# Tag if not exists
echo "Creating tag $TAG..."
git tag -a "$TAG" -m "Release $TAG" 2>/dev/null || echo "Tag $TAG already exists"

# Push tag to origin
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

# Create release with binaries
echo "Creating GitHub release with binaries..."
gh release create "$TAG" \
    cmd/kubectl-gt/bin/kubectl-gt-darwin-arm64 \
    cmd/kubectl-gt/bin/kubectl-gt-darwin-amd64 \
    cmd/kubectl-gt/bin/kubectl-gt-linux-amd64 \
    cmd/kubectl-gt/bin/kubectl-gt-linux-arm64 \
    cmd/kubectl-gt/bin/kubectl-gt-windows-amd64.exe \
    --title "$TAG - kubectl-gt AI-Native CLI" \
    --notes "$BODY" \
    --repo "$REPO" \
    2>/dev/null || {
        echo "Release $TAG already exists, uploading assets..."
        gh release upload "$TAG" \
            cmd/kubectl-gt/bin/kubectl-gt-darwin-arm64 \
            cmd/kubectl-gt/bin/kubectl-gt-darwin-amd64 \
            cmd/kubectl-gt/bin/kubectl-gt-linux-amd64 \
            cmd/kubectl-gt/bin/kubectl-gt-linux-arm64 \
            cmd/kubectl-gt/bin/kubectl-gt-windows-amd64.exe \
            --repo "$REPO" \
            --clobber
    }

echo ""
echo "========================================"
echo "  GitHub Release Complete"
echo "========================================"
echo ""
echo "Release: https://github.com/$REPO/releases/tag/$TAG"
echo ""
echo "kubectl-gt binaries:"
echo "  - kubectl-gt-darwin-arm64"
echo "  - kubectl-gt-darwin-amd64"
echo "  - kubectl-gt-linux-amd64"
echo "  - kubectl-gt-linux-arm64"
echo "  - kubectl-gt-windows-amd64.exe"
echo ""
