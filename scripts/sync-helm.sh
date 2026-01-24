#!/bin/bash
# Sync Helm chart CRDs with generated manifests
#
# This script copies CRDs from config/crd/bases/ to helm/gastown-operator/crds/
# ensuring the Helm chart always uses the latest generated CRD schemas.
#
# Usage:
#   ./scripts/sync-helm.sh
#
# Run after: make manifests

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

CRD_SOURCE="$PROJECT_ROOT/config/crd/bases"
CRD_DEST="$PROJECT_ROOT/helm/gastown-operator/crds"

echo "========================================"
echo "  Syncing Helm Chart CRDs"
echo "========================================"
echo ""
echo "Source: $CRD_SOURCE"
echo "Dest:   $CRD_DEST"
echo ""

# Ensure source exists
if [ ! -d "$CRD_SOURCE" ]; then
    echo "ERROR: CRD source directory does not exist: $CRD_SOURCE"
    echo "Run 'make manifests' first."
    exit 1
fi

# Ensure dest exists
mkdir -p "$CRD_DEST"

# Copy CRDs
echo "Copying CRDs..."
for crd in "$CRD_SOURCE"/*.yaml; do
    base=$(basename "$crd")
    cp "$crd" "$CRD_DEST/$base"
    echo "  ✓ $base"
done

echo ""
echo "========================================"
echo "  Helm chart CRDs synced successfully"
echo "========================================"
echo ""
echo "Next steps:"
echo "  1. Review changes: git diff helm/gastown-operator/crds/"
echo "  2. Commit: git add helm/gastown-operator/crds/ && git commit -m 'chore: sync helm CRDs'"
echo ""
