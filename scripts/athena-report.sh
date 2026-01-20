#!/bin/bash
# athena-report.sh - Format and send pipeline data to Athena for knowledge persistence
#
# Stores pipeline results as episodes in Athena memory for future reference:
#   - Release metadata (version, date, commit)
#   - Vibe scan results (findings count, patterns detected)
#   - Build artifacts (images, SBOMs)
#   - Test results
#
# Usage:
#   ./scripts/athena-report.sh release 0.2.0          # Store release episode
#   ./scripts/athena-report.sh vibe results.json     # Store vibe results
#   ./scripts/athena-report.sh pipeline <status>     # Store pipeline status
#
# Environment:
#   ATHENA_API_KEY    - API key for Athena (required)
#   ATHENA_API_URL    - Athena API endpoint (default: https://athena.olympus.io/api)
#   ATHENA_COLLECTION - Collection/tenant name (default: default)

set -euo pipefail

# Configuration
ATHENA_API_URL="${ATHENA_API_URL:-https://athena.olympus.io/api}"
ATHENA_COLLECTION="${ATHENA_COLLECTION:-default}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m'

log_info() { echo -e "${GREEN}[INFO]${NC} $1"; }
log_warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
log_error() { echo -e "${RED}[ERROR]${NC} $1"; }

# Check prerequisites
check_prereqs() {
    if [[ -z "${ATHENA_API_KEY:-}" ]]; then
        log_warn "ATHENA_API_KEY not set - skipping Athena persistence"
        log_info "To enable: export ATHENA_API_KEY=<your-key>"
        exit 0
    fi

    if ! command -v curl &> /dev/null; then
        log_error "curl is required but not installed"
        exit 1
    fi

    if ! command -v jq &> /dev/null; then
        log_error "jq is required but not installed"
        exit 1
    fi
}

# Store a memory in Athena
store_memory() {
    local memory_type="$1"
    local content="$2"
    local tags="$3"
    local source="${4:-tekton-pipeline}"
    local confidence="${5:-0.9}"

    local payload
    payload=$(jq -n \
        --arg type "$memory_type" \
        --arg content "$content" \
        --arg source "$source" \
        --argjson confidence "$confidence" \
        --argjson tags "$tags" \
        '{
            memory_type: $type,
            content: $content,
            source: $source,
            confidence: $confidence,
            tags: $tags,
            collection: "'"$ATHENA_COLLECTION"'"
        }')

    log_info "Storing memory in Athena..."
    log_info "Type: $memory_type"
    log_info "Tags: $tags"

    local response
    response=$(curl -s -w "\n%{http_code}" \
        -X POST "$ATHENA_API_URL/memory" \
        -H "Authorization: Bearer $ATHENA_API_KEY" \
        -H "Content-Type: application/json" \
        -d "$payload")

    local http_code
    http_code=$(echo "$response" | tail -n1)
    local body
    body=$(echo "$response" | sed '$d')

    if [[ "$http_code" -ge 200 ]] && [[ "$http_code" -lt 300 ]]; then
        log_info "Memory stored successfully"
        echo "$body" | jq . 2>/dev/null || echo "$body"
    else
        log_error "Failed to store memory (HTTP $http_code)"
        echo "$body"
        return 1
    fi
}

# Store release episode
store_release() {
    local version="$1"
    local git_sha="${2:-$(git rev-parse HEAD 2>/dev/null || echo 'unknown')}"
    local git_branch="${3:-$(git branch --show-current 2>/dev/null || echo 'unknown')}"

    local content="Release v$version of gastown-operator completed.
Git SHA: $git_sha
Branch: $git_branch
Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
Repository: github.com/boshu2/gastown-operator

Artifacts:
- Container image: ghcr.io/boshu2/gastown-operator:v$version
- Helm chart: oci://ghcr.io/boshu2/gastown-operator:$version
- SBOM: ghcr.io/boshu2/gastown-operator:v$version.sbom"

    local tags='["release", "gastown-operator", "v'"$version"'", "kubernetes", "operator"]'

    store_memory "episode" "$content" "$tags" "tekton-release-pipeline" "0.95"
}

# Store vibe prescan results
store_vibe_results() {
    local results_file="${1:-}"
    local critical="${2:-0}"
    local high="${3:-0}"
    local medium="${4:-0}"

    local content="Vibe prescan completed for gastown-operator.
Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
Findings:
- CRITICAL: $critical
- HIGH: $high
- MEDIUM: $medium

Patterns checked:
- P1: Phantom modifications
- P4: TODO/FIXME markers
- P5: Cyclomatic complexity
- P8: Unchecked errors
- P13: Undocumented ignores
- P14: Error wrapping
- P15: golangci-lint"

    if [[ -n "$results_file" ]] && [[ -f "$results_file" ]]; then
        content="$content

Details:
$(cat "$results_file" | head -50)"
    fi

    local tags='["vibe", "quality", "gastown-operator", "prescan", "static-analysis"]'

    store_memory "fact" "$content" "$tags" "vibe-prescan" "0.85"
}

# Store pipeline status
store_pipeline_status() {
    local status="$1"
    local pipeline_name="${2:-gastown-operator-ci}"
    local duration="${3:-unknown}"

    local content="Pipeline $pipeline_name $status.
Date: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
Duration: $duration
Status: $status

Stages completed:
- Clone
- Security scans (govulncheck, gosec, gitleaks, trivy)
- Vibe prescan
- Unit tests
- Build
- Image scan
- SBOM generation
- Image signing
- GHCR mirror"

    local tags='["pipeline", "ci", "gastown-operator", "tekton", "'"$status"'"]'

    store_memory "episode" "$content" "$tags" "tekton-pipeline" "0.9"
}

# Main
main() {
    check_prereqs

    local command="${1:-help}"

    case "$command" in
        release)
            local version="${2:?Version required}"
            store_release "$version" "${3:-}" "${4:-}"
            ;;
        vibe)
            store_vibe_results "${2:-}" "${3:-0}" "${4:-0}" "${5:-0}"
            ;;
        pipeline)
            local status="${2:?Status required (success/failure)}"
            store_pipeline_status "$status" "${3:-}" "${4:-}"
            ;;
        help|--help|-h)
            cat << 'EOF'
Athena Report - Pipeline Knowledge Persistence

Usage:
  ./scripts/athena-report.sh release <version> [sha] [branch]
  ./scripts/athena-report.sh vibe [results.json] [critical] [high] [medium]
  ./scripts/athena-report.sh pipeline <status> [name] [duration]

Environment:
  ATHENA_API_KEY    - API key for Athena (required)
  ATHENA_API_URL    - API endpoint (default: https://athena.olympus.io/api)
  ATHENA_COLLECTION - Collection name (default: default)

Examples:
  ./scripts/athena-report.sh release 0.2.0
  ./scripts/athena-report.sh vibe results.json 0 2 5
  ./scripts/athena-report.sh pipeline success gastown-operator-ci 15m
EOF
            ;;
        *)
            log_error "Unknown command: $command"
            log_info "Run with --help for usage"
            exit 1
            ;;
    esac
}

main "$@"
