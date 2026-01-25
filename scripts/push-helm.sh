#!/bin/bash
# Push Helm chart to GHCR OCI registry
#
# Usage:
#   ./scripts/push-helm.sh <version>
#   make ci-push (calls this after image push)
#
# Prerequisites:
#   - helm v3.8+ (OCI support)
#   - GITHUB_TOKEN environment variable set

set -euo pipefail

VERSION="${1:-}"
if [ -z "$VERSION" ]; then
    echo "Usage: $0 <version>"
    echo "Example: $0 0.3.3"
    exit 1
fi

REGISTRY="ghcr.io/boshu2"
CHART_DIR="helm/gastown-operator"

echo "========================================"
echo "  Pushing Helm Chart to GHCR"
echo "========================================"
echo ""
echo "Version: $VERSION"
echo "Chart:   $CHART_DIR"
echo "Target:  oci://$REGISTRY/charts"
echo ""

# Check helm is available
if ! command -v helm &> /dev/null; then
    echo "ERROR: helm not found"
    exit 1
fi

# Check helm version supports OCI (3.8+)
HELM_VERSION=$(helm version --template='{{.Version}}' | sed 's/v//')
HELM_MAJOR=$(echo "$HELM_VERSION" | cut -d. -f1)
HELM_MINOR=$(echo "$HELM_VERSION" | cut -d. -f2)
if [ "$HELM_MAJOR" -lt 3 ] || ([ "$HELM_MAJOR" -eq 3 ] && [ "$HELM_MINOR" -lt 8 ]); then
    echo "ERROR: helm v3.8+ required for OCI support (found v$HELM_VERSION)"
    exit 1
fi

# Check GITHUB_TOKEN
if [ -z "${GITHUB_TOKEN:-}" ]; then
    echo "ERROR: GITHUB_TOKEN not set"
    echo "Run: export GITHUB_TOKEN=<your-token>"
    exit 1
fi

# Login to GHCR
echo "Logging into GHCR..."
echo "$GITHUB_TOKEN" | helm registry login ghcr.io -u boshu2 --password-stdin

# Update Chart.yaml with version
echo "Updating Chart.yaml to version $VERSION..."
sed -i.bak "s/^version:.*/version: $VERSION/" "$CHART_DIR/Chart.yaml"
sed -i.bak "s/^appVersion:.*/appVersion: \"$VERSION\"/" "$CHART_DIR/Chart.yaml"
rm -f "$CHART_DIR/Chart.yaml.bak"

# Package chart
echo "Packaging chart..."
helm package "$CHART_DIR" --destination /tmp/

# Push to GHCR
echo "Pushing to GHCR..."
helm push "/tmp/gastown-operator-$VERSION.tgz" "oci://$REGISTRY/charts"

# Cleanup
rm -f "/tmp/gastown-operator-$VERSION.tgz"

echo ""
echo "========================================"
echo "  Helm Chart Pushed"
echo "========================================"
echo ""
echo "Published: oci://$REGISTRY/charts/gastown-operator:$VERSION"
echo ""
echo "Install with:"
echo "  helm install gastown-operator oci://$REGISTRY/charts/gastown-operator \\"
echo "    --version $VERSION \\"
echo "    -n gastown-system --create-namespace"
echo ""
