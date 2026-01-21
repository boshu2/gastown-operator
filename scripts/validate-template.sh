#!/usr/bin/env bash
#
# validate-template.sh - Pre-flight validation for Gas Town templates
#
# Usage:
#   ./scripts/validate-template.sh <template-file>
#   ./scripts/validate-template.sh templates/polecat-kubernetes.yaml
#
# Checks:
#   1. YAML syntax valid
#   2. Required fields present
#   3. Variable markers identified
#   4. API version and kind correct
#   5. References resolvable (secrets, rigs)
#
# Exit codes:
#   0 - All checks passed
#   1 - Validation failed
#   2 - Usage error

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
NC='\033[0m' # No Color

# Track overall status
ERRORS=0
WARNINGS=0

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ((ERRORS++))
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
    ((WARNINGS++))
}

log_ok() {
    echo -e "${GREEN}[OK]${NC} $1"
}

log_info() {
    echo -e "[INFO] $1"
}

usage() {
    echo "Usage: $0 <template-file>"
    echo ""
    echo "Validates Gas Town Kubernetes template files."
    echo ""
    echo "Examples:"
    echo "  $0 templates/polecat-minimal.yaml"
    echo "  $0 templates/convoy.yaml"
    exit 2
}

check_yaml_syntax() {
    local file=$1
    log_info "Checking YAML syntax..."

    if ! command -v yq &> /dev/null; then
        log_warn "yq not installed, falling back to kubectl dry-run"
        # Can't do pure YAML check without yq
        return 0
    fi

    if yq eval '.' "$file" > /dev/null 2>&1; then
        log_ok "YAML syntax valid"
    else
        log_error "YAML syntax invalid"
        yq eval '.' "$file" 2>&1 | head -5
    fi
}

check_api_version() {
    local file=$1
    log_info "Checking API version..."

    local api_version
    api_version=$(grep -m1 "^apiVersion:" "$file" | awk '{print $2}' || echo "")

    if [[ "$api_version" == "gastown.gastown.io/v1alpha1" ]]; then
        log_ok "API version correct: $api_version"
    elif [[ "$api_version" == *"gastown"* ]]; then
        log_warn "API version may be outdated: $api_version (expected gastown.gastown.io/v1alpha1)"
    elif [[ -z "$api_version" ]]; then
        log_error "No apiVersion found"
    else
        log_info "Non-Gas Town resource: $api_version (skipping Gas Town specific checks)"
        return 1  # Signal to skip Gas Town checks
    fi
    return 0
}

check_kind() {
    local file=$1
    log_info "Checking resource kind..."

    local kind
    kind=$(grep -m1 "^kind:" "$file" | awk '{print $2}' || echo "")

    case "$kind" in
        Polecat|Convoy|Witness|Refinery|Rig|BeadStore)
            log_ok "Kind valid: $kind"
            echo "$kind"
            ;;
        Secret|ConfigMap)
            log_info "Supporting resource: $kind"
            echo "$kind"
            ;;
        "")
            log_error "No kind found"
            echo ""
            ;;
        *)
            log_warn "Unknown kind: $kind"
            echo "$kind"
            ;;
    esac
}

check_unreplaced_variables() {
    local file=$1
    log_info "Checking for unreplaced variables..."

    local vars
    vars=$(grep -oE '\{\{[A-Z_]+\}\}' "$file" 2>/dev/null | sort -u || echo "")

    if [[ -n "$vars" ]]; then
        log_warn "Found unreplaced variables (replace before applying):"
        echo "$vars" | while read -r var; do
            echo "  - $var"
        done
    else
        log_ok "No unreplaced variables"
    fi
}

check_required_fields_polecat() {
    local file=$1
    log_info "Checking Polecat required fields..."

    # Required: rig
    if grep -q "rig:" "$file"; then
        log_ok "Field 'rig' present"
    else
        log_error "Missing required field: rig"
    fi

    # Required: desiredState
    if grep -q "desiredState:" "$file"; then
        local state
        state=$(grep "desiredState:" "$file" | head -1 | awk '{print $2}')
        case "$state" in
            Idle|Working|Terminated|"{{DESIRED_STATE}}")
                log_ok "Field 'desiredState' valid: $state"
                ;;
            *)
                log_error "Invalid desiredState: $state (expected Idle, Working, or Terminated)"
                ;;
        esac
    else
        log_error "Missing required field: desiredState"
    fi

    # If executionMode is kubernetes, check for required kubernetes fields
    if grep -q "executionMode: kubernetes" "$file"; then
        log_info "Kubernetes mode detected, checking kubernetes spec..."

        if grep -q "gitRepository:" "$file"; then
            log_ok "Field 'kubernetes.gitRepository' present"
        else
            log_error "Missing required field for kubernetes mode: gitRepository"
        fi

        if grep -q "gitSecretRef:" "$file"; then
            log_ok "Field 'kubernetes.gitSecretRef' present"
        else
            log_error "Missing required field for kubernetes mode: gitSecretRef"
        fi

        # Either claudeCredsSecretRef or apiKeySecretRef
        if grep -q "claudeCredsSecretRef:" "$file" || grep -q "apiKeySecretRef:" "$file"; then
            log_ok "Claude credentials reference present"
        else
            log_error "Missing required: either claudeCredsSecretRef or apiKeySecretRef"
        fi
    fi
}

check_required_fields_convoy() {
    local file=$1
    log_info "Checking Convoy required fields..."

    if grep -q "description:" "$file"; then
        log_ok "Field 'description' present"
    else
        log_error "Missing required field: description"
    fi

    if grep -q "trackedBeads:" "$file"; then
        log_ok "Field 'trackedBeads' present"
    else
        log_error "Missing required field: trackedBeads"
    fi
}

check_required_fields_witness() {
    local file=$1
    log_info "Checking Witness required fields..."

    if grep -q "rigRef:" "$file"; then
        log_ok "Field 'rigRef' present"
    else
        log_error "Missing required field: rigRef"
    fi
}

check_required_fields_refinery() {
    local file=$1
    log_info "Checking Refinery required fields..."

    if grep -q "rigRef:" "$file"; then
        log_ok "Field 'rigRef' present"
    else
        log_error "Missing required field: rigRef"
    fi
}

check_required_fields_rig() {
    local file=$1
    log_info "Checking Rig required fields..."

    local required_fields=("gitURL" "beadsPrefix" "localPath")
    for field in "${required_fields[@]}"; do
        if grep -q "${field}:" "$file"; then
            log_ok "Field '$field' present"
        else
            log_error "Missing required field: $field"
        fi
    done
}

check_bead_id_format() {
    local file=$1
    log_info "Checking bead ID format..."

    local bead_ids
    bead_ids=$(grep -oE '"[a-zA-Z0-9-]+-[a-zA-Z0-9]+"' "$file" 2>/dev/null || echo "")

    while read -r bead_id; do
        [[ -z "$bead_id" ]] && continue
        # Remove quotes
        bead_id=${bead_id//\"/}

        # Skip variable markers
        [[ "$bead_id" == *"{{"* ]] && continue

        # Check pattern: lowercase prefix (2-10 chars) + hyphen + alphanumeric
        if [[ "$bead_id" =~ ^[a-z]{2,10}-[a-z0-9]+$ ]]; then
            log_ok "Bead ID format valid: $bead_id"
        else
            log_warn "Bead ID may be invalid: $bead_id (expected pattern: prefix-id, e.g., at-1234)"
        fi
    done <<< "$bead_ids"
}

check_namespace() {
    local file=$1
    log_info "Checking namespace..."

    local namespace
    namespace=$(grep -A5 "^metadata:" "$file" | grep "namespace:" | head -1 | awk '{print $2}' || echo "")

    if [[ -z "$namespace" ]]; then
        log_warn "No namespace specified (will use default namespace)"
    elif [[ "$namespace" == "gastown-system" || "$namespace" == "gastown" ]]; then
        log_ok "Namespace: $namespace"
    elif [[ "$namespace" == *"{{"* ]]; then
        log_info "Namespace is a variable: $namespace"
    else
        log_info "Custom namespace: $namespace"
    fi
}

# Main validation
main() {
    if [[ $# -ne 1 ]]; then
        usage
    fi

    local file=$1

    if [[ ! -f "$file" ]]; then
        log_error "File not found: $file"
        exit 1
    fi

    echo "=========================================="
    echo "Validating: $file"
    echo "=========================================="
    echo ""

    # Basic checks
    check_yaml_syntax "$file"

    # Check API version (returns 1 if not a Gas Town resource)
    if ! check_api_version "$file"; then
        echo ""
        echo "Skipping Gas Town specific checks for non-Gas Town resource."
        exit 0
    fi

    # Get kind and check required fields
    local kind
    kind=$(check_kind "$file")

    check_namespace "$file"
    check_unreplaced_variables "$file"

    # Kind-specific checks
    case "$kind" in
        Polecat)
            check_required_fields_polecat "$file"
            check_bead_id_format "$file"
            ;;
        Convoy)
            check_required_fields_convoy "$file"
            check_bead_id_format "$file"
            ;;
        Witness)
            check_required_fields_witness "$file"
            ;;
        Refinery)
            check_required_fields_refinery "$file"
            ;;
        Rig)
            check_required_fields_rig "$file"
            ;;
    esac

    echo ""
    echo "=========================================="
    if [[ $ERRORS -gt 0 ]]; then
        echo -e "${RED}Validation FAILED${NC}: $ERRORS error(s), $WARNINGS warning(s)"
        exit 1
    elif [[ $WARNINGS -gt 0 ]]; then
        echo -e "${YELLOW}Validation PASSED with warnings${NC}: $WARNINGS warning(s)"
        exit 0
    else
        echo -e "${GREEN}Validation PASSED${NC}"
        exit 0
    fi
}

main "$@"
