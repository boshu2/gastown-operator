#!/bin/bash
# version-sync.sh - Sync VERSION file to Chart.yaml and other locations
#
# Single source of truth: VERSION file in repo root
# This script syncs that version to:
#   - helm/gastown-operator/Chart.yaml (version + appVersion)
#
# Usage:
#   ./scripts/version-sync.sh          # Sync current VERSION
#   ./scripts/version-sync.sh 0.2.0    # Override with specific version
#   ./scripts/version-sync.sh --check  # Check if in sync (exit 1 if not)
#
# Called by:
#   - semantic-release (after version bump)
#   - Manual release preparation

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

VERSION_FILE="$REPO_ROOT/VERSION"
CHART_FILE="$REPO_ROOT/helm/gastown-operator/Chart.yaml"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

get_version() {
    if [[ -f "$VERSION_FILE" ]]; then
        cat "$VERSION_FILE" | tr -d '[:space:]'
    else
        log_error "VERSION file not found at $VERSION_FILE"
        exit 1
    fi
}

get_chart_version() {
    grep "^version:" "$CHART_FILE" | awk '{print $2}' | tr -d '[:space:]'
}

get_chart_app_version() {
    grep "^appVersion:" "$CHART_FILE" | awk '{print $2}' | tr -d '"[:space:]'
}

check_sync() {
    local version
    version=$(get_version)
    local chart_version
    chart_version=$(get_chart_version)
    local app_version
    app_version=$(get_chart_app_version)

    local in_sync=true

    echo "Checking version sync..."
    echo "  VERSION file:     $version"
    echo "  Chart version:    $chart_version"
    echo "  Chart appVersion: $app_version"

    if [[ "$version" != "$chart_version" ]]; then
        log_error "Chart version mismatch: expected $version, got $chart_version"
        in_sync=false
    fi

    if [[ "$version" != "$app_version" ]]; then
        log_error "Chart appVersion mismatch: expected $version, got $app_version"
        in_sync=false
    fi

    if [[ "$in_sync" == "true" ]]; then
        log_info "All versions in sync: $version"
        exit 0
    else
        log_error "Versions out of sync. Run: ./scripts/version-sync.sh"
        exit 1
    fi
}

sync_version() {
    local version="$1"

    log_info "Syncing version: $version"

    # Update VERSION file if version was provided as argument
    if [[ "$2" == "update-file" ]]; then
        echo "$version" > "$VERSION_FILE"
        log_info "Updated VERSION file"
    fi

    # Update Chart.yaml version
    if [[ -f "$CHART_FILE" ]]; then
        # macOS sed requires different syntax
        if [[ "$(uname)" == "Darwin" ]]; then
            sed -i '' "s/^version:.*/version: $version/" "$CHART_FILE"
            sed -i '' "s/^appVersion:.*/appVersion: \"$version\"/" "$CHART_FILE"
        else
            sed -i "s/^version:.*/version: $version/" "$CHART_FILE"
            sed -i "s/^appVersion:.*/appVersion: \"$version\"/" "$CHART_FILE"
        fi
        log_info "Updated Chart.yaml: version=$version, appVersion=$version"
    else
        log_warn "Chart.yaml not found at $CHART_FILE"
    fi

    log_info "Version sync complete!"
}

# Main
case "${1:-}" in
    --check)
        check_sync
        ;;
    --help|-h)
        echo "Usage: $0 [VERSION|--check|--help]"
        echo ""
        echo "Commands:"
        echo "  (none)     Sync VERSION file to Chart.yaml"
        echo "  VERSION    Set version to VERSION and sync"
        echo "  --check    Check if versions are in sync"
        echo "  --help     Show this help"
        exit 0
        ;;
    "")
        # No argument - sync from VERSION file
        version=$(get_version)
        sync_version "$version" "no-update"
        ;;
    *)
        # Version provided - update VERSION file and sync
        version="$1"
        # Validate version format (semver)
        if ! echo "$version" | grep -qE '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.]+)?$'; then
            log_error "Invalid version format: $version"
            log_error "Expected: X.Y.Z (e.g., 0.1.1, 1.0.0-rc.1)"
            exit 1
        fi
        sync_version "$version" "update-file"
        ;;
esac
