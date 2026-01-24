#!/bin/bash
# Validate Helm chart CRDs match generated CRDs
#
# This script validates that:
#   1. Helm chart CRDs are in sync with config/crd/bases/
#   2. Helm RBAC includes required pod permissions
#
# Usage:
#   ./scripts/validate-helm-sync.sh
#
# Exit codes:
#   0 - All validations passed
#   1 - Validation failed

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

CRD_SOURCE="$PROJECT_ROOT/config/crd/bases"
CRD_DEST="$PROJECT_ROOT/helm/gastown-operator/crds"
CLUSTERROLE="$PROJECT_ROOT/helm/gastown-operator/templates/clusterrole.yaml"

ERRORS=0

echo "========================================"
echo "  Validating Helm Chart Sync"
echo "========================================"
echo ""

# =============================================================================
# Check 1: CRD sync
# =============================================================================
echo "[1/2] Checking CRD sync..."

if [ ! -d "$CRD_SOURCE" ]; then
    echo "  ERROR: CRD source directory does not exist: $CRD_SOURCE"
    echo "  Run 'make manifests' first."
    exit 1
fi

if [ ! -d "$CRD_DEST" ]; then
    echo "  ERROR: Helm CRD directory does not exist: $CRD_DEST"
    exit 1
fi

CRD_DRIFT=false
for crd in "$CRD_SOURCE"/*.yaml; do
    base=$(basename "$crd")
    helm_crd="$CRD_DEST/$base"

    if [ ! -f "$helm_crd" ]; then
        echo "  ✗ MISSING: $base"
        CRD_DRIFT=true
        continue
    fi

    if ! diff -q "$crd" "$helm_crd" > /dev/null 2>&1; then
        echo "  ✗ DRIFT: $base"
        # Show size difference for context
        src_size=$(wc -c < "$crd" | tr -d ' ')
        dst_size=$(wc -c < "$helm_crd" | tr -d ' ')
        echo "    Source: $src_size bytes, Helm: $dst_size bytes"
        CRD_DRIFT=true
    else
        echo "  ✓ $base"
    fi
done

if [ "$CRD_DRIFT" = true ]; then
    echo ""
    echo "  ERROR: CRD drift detected!"
    echo "  Run 'make sync-helm' to fix."
    ERRORS=$((ERRORS + 1))
else
    echo "  PASS: All CRDs in sync"
fi

echo ""

# =============================================================================
# Check 2: RBAC pod permissions
# =============================================================================
echo "[2/2] Checking RBAC pod permissions..."

if [ ! -f "$CLUSTERROLE" ]; then
    echo "  ERROR: ClusterRole template not found: $CLUSTERROLE"
    ERRORS=$((ERRORS + 1))
else
    # Check for pod resource in core API group
    if grep -A 5 'apiGroups:' "$CLUSTERROLE" | grep -q 'pods'; then
        echo "  ✓ Pod permissions present"
    else
        echo "  ✗ Pod permissions missing!"
        echo "    Helm RBAC must include pods (create, get, list, watch, delete) for Kubernetes execution mode."
        ERRORS=$((ERRORS + 1))
    fi
fi

echo ""

# =============================================================================
# Summary
# =============================================================================
echo "========================================"
if [ $ERRORS -eq 0 ]; then
    echo "  All validations PASSED"
    echo "========================================"
    exit 0
else
    echo "  Validations FAILED ($ERRORS errors)"
    echo "========================================"
    echo ""
    echo "To fix:"
    echo "  make sync-helm  # Sync CRDs"
    echo "  # Then fix any RBAC issues manually"
    exit 1
fi
