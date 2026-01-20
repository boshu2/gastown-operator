#!/bin/bash
# prescan.sh - Fast static quality validation for CI pipelines
#
# Vibe Prescan - Run before expensive builds to catch issues early
#
# Patterns checked:
#   P1:  Phantom modifications (git dirty state)
#   P4:  TODO/FIXME markers in code
#   P5:  Cyclomatic complexity > threshold
#   P8:  Unchecked error returns
#   P13: Undocumented error ignores
#   P14: Error wrapping with %v (should be %w)
#   P15: golangci-lint violations
#
# Exit codes:
#   0 - All checks passed
#   1 - Error running checks
#   2 - CRITICAL findings detected
#   3 - HIGH severity findings detected
#   4 - MEDIUM severity findings detected
#
# Usage:
#   ./scripts/prescan.sh             # Scan all Go files
#   ./scripts/prescan.sh recent      # Scan only recently changed files
#   ./scripts/prescan.sh <dir>       # Scan specific directory
#   ./scripts/prescan.sh --help      # Show help
#
# Environment:
#   PRESCAN_FAIL_ON=HIGH             # Threshold (CRITICAL, HIGH, MEDIUM)
#   PRESCAN_COMPLEXITY_THRESHOLD=15  # Max cyclomatic complexity

set -euo pipefail

# Configuration
FAIL_ON="${PRESCAN_FAIL_ON:-HIGH}"
COMPLEXITY_THRESHOLD="${PRESCAN_COMPLEXITY_THRESHOLD:-15}"
TARGET="${1:-all}"

# Counters
CRITICAL_COUNT=0
HIGH_COUNT=0
MEDIUM_COUNT=0

# Colors
RED='\033[0;31m'
YELLOW='\033[0;33m'
GREEN='\033[0;32m'
CYAN='\033[0;36m'
NC='\033[0m'

log_critical() { echo -e "${RED}[CRITICAL]${NC} $1"; ((CRITICAL_COUNT++)); }
log_high() { echo -e "${RED}[HIGH]${NC} $1"; ((HIGH_COUNT++)); }
log_medium() { echo -e "${YELLOW}[MEDIUM]${NC} $1"; ((MEDIUM_COUNT++)); }
log_info() { echo -e "${CYAN}[INFO]${NC} $1"; }
log_pass() { echo -e "${GREEN}[PASS]${NC} $1"; }

# Get files to scan
get_files() {
    case "$TARGET" in
        all)
            find . -name "*.go" -not -path "./vendor/*" -not -path "./.git/*"
            ;;
        recent)
            # Files changed in last commit or uncommitted
            git diff --name-only HEAD~1 2>/dev/null | grep '\.go$' || true
            git diff --name-only 2>/dev/null | grep '\.go$' || true
            ;;
        *)
            find "$TARGET" -name "*.go" -not -path "./vendor/*"
            ;;
    esac
}

# P1: Check for phantom modifications (git dirty state)
check_phantom_modifications() {
    log_info "P1: Checking for phantom modifications..."

    if ! git diff --quiet 2>/dev/null; then
        log_high "P1: Uncommitted changes detected"
        git diff --stat 2>/dev/null || true
    else
        log_pass "P1: No phantom modifications"
    fi
}

# P4: Check for TODO/FIXME markers
check_todo_markers() {
    log_info "P4: Checking for TODO/FIXME markers..."

    local count=0
    while IFS= read -r file; do
        local matches
        matches=$(grep -n -E "(TODO|FIXME|XXX|HACK):" "$file" 2>/dev/null || true)
        if [[ -n "$matches" ]]; then
            echo "$matches" | while read -r line; do
                log_medium "P4: $file:$line"
                ((count++)) || true
            done
        fi
    done < <(get_files)

    if [[ $count -eq 0 ]]; then
        log_pass "P4: No TODO/FIXME markers found"
    fi
}

# P5: Check cyclomatic complexity
check_complexity() {
    log_info "P5: Checking cyclomatic complexity (threshold: $COMPLEXITY_THRESHOLD)..."

    if ! command -v gocyclo &> /dev/null; then
        log_info "P5: gocyclo not installed, skipping"
        return
    fi

    local high_complexity
    high_complexity=$(gocyclo -over "$COMPLEXITY_THRESHOLD" . 2>/dev/null || true)

    if [[ -n "$high_complexity" ]]; then
        echo "$high_complexity" | while read -r line; do
            # Skip if nolint:gocyclo comment exists
            local func_file
            func_file=$(echo "$line" | awk '{print $NF}')
            if grep -q "nolint:gocyclo" "$func_file" 2>/dev/null; then
                log_info "P5: Skipping (nolint): $line"
            else
                log_high "P5: High complexity: $line"
            fi
        done
    else
        log_pass "P5: All functions below complexity threshold"
    fi
}

# P8: Check for unchecked error returns
check_unchecked_errors() {
    log_info "P8: Checking for unchecked error returns..."

    local count=0
    while IFS= read -r file; do
        # Look for patterns like: something() without = err
        # This is a simplified check - golangci-lint does this better
        local matches
        matches=$(grep -n -E '^\s*[a-zA-Z_][a-zA-Z0-9_]*\([^)]*\)\s*$' "$file" 2>/dev/null | grep -v "defer" || true)
        if [[ -n "$matches" ]]; then
            echo "$matches" | head -5 | while read -r line; do
                # Only flag if likely returns error (contains common error-returning patterns)
                if echo "$line" | grep -qE '(Close|Write|Read|Flush|Sync)\(' ; then
                    log_medium "P8: Potential unchecked error: $file:$line"
                    ((count++)) || true
                fi
            done
        fi
    done < <(get_files)

    if [[ $count -eq 0 ]]; then
        log_pass "P8: No obvious unchecked errors found"
    fi
}

# P13: Check for undocumented error ignores
check_undocumented_ignores() {
    log_info "P13: Checking for undocumented error ignores..."

    local count=0
    while IFS= read -r file; do
        # Look for _ = something that looks like an error
        local matches
        matches=$(grep -n -E '^\s*_\s*=' "$file" 2>/dev/null || true)
        if [[ -n "$matches" ]]; then
            echo "$matches" | while read -r line; do
                local line_num
                line_num=$(echo "$line" | cut -d: -f1)
                # Check if previous line has nolint comment
                local prev_line
                prev_line=$(sed -n "$((line_num-1))p" "$file" 2>/dev/null || true)
                if ! echo "$prev_line" | grep -q "nolint:errcheck"; then
                    log_medium "P13: Undocumented error ignore: $file:$line"
                    ((count++)) || true
                fi
            done
        fi
    done < <(get_files)

    if [[ $count -eq 0 ]]; then
        log_pass "P13: All error ignores documented"
    fi
}

# P14: Check for error wrapping with %v instead of %w
check_error_wrapping() {
    log_info "P14: Checking for error wrapping with %v..."

    local count=0
    while IFS= read -r file; do
        # Look for fmt.Errorf with %v for errors (should be %w)
        local matches
        matches=$(grep -n 'fmt\.Errorf.*%v.*err' "$file" 2>/dev/null || true)
        if [[ -n "$matches" ]]; then
            echo "$matches" | while read -r line; do
                log_high "P14: Use %%w for error wrapping: $file:$line"
                ((count++)) || true
            done
        fi
    done < <(get_files)

    if [[ $count -eq 0 ]]; then
        log_pass "P14: All error wrapping uses %w"
    fi
}

# P15: Run golangci-lint
check_golangci_lint() {
    log_info "P15: Running golangci-lint..."

    if ! command -v golangci-lint &> /dev/null; then
        log_info "P15: golangci-lint not installed, skipping"
        return
    fi

    local lint_output
    lint_output=$(golangci-lint run --timeout=5m 2>&1 || true)

    if [[ -n "$lint_output" ]] && ! echo "$lint_output" | grep -q "congratulations"; then
        echo "$lint_output" | head -20 | while read -r line; do
            if echo "$line" | grep -qE "(error|Error)"; then
                log_high "P15: $line"
            else
                log_medium "P15: $line"
            fi
        done
    else
        log_pass "P15: golangci-lint passed"
    fi
}

# Main
main() {
    echo "========================================"
    echo "  Vibe Prescan - Quality Gate"
    echo "========================================"
    echo "Target: $TARGET"
    echo "Fail on: $FAIL_ON"
    echo "Complexity threshold: $COMPLEXITY_THRESHOLD"
    echo ""

    check_phantom_modifications
    echo ""
    check_todo_markers
    echo ""
    check_complexity
    echo ""
    check_unchecked_errors
    echo ""
    check_undocumented_ignores
    echo ""
    check_error_wrapping
    echo ""
    check_golangci_lint

    echo ""
    echo "========================================"
    echo "  Summary"
    echo "========================================"
    echo "CRITICAL: $CRITICAL_COUNT"
    echo "HIGH:     $HIGH_COUNT"
    echo "MEDIUM:   $MEDIUM_COUNT"
    echo ""

    # Determine exit code based on threshold
    if [[ $CRITICAL_COUNT -gt 0 ]]; then
        log_critical "Pipeline should fail - CRITICAL findings detected"
        echo "critical-count=$CRITICAL_COUNT" >> "${GITHUB_OUTPUT:-/dev/null}" 2>/dev/null || true
        echo "high-count=$HIGH_COUNT" >> "${GITHUB_OUTPUT:-/dev/null}" 2>/dev/null || true
        echo "medium-count=$MEDIUM_COUNT" >> "${GITHUB_OUTPUT:-/dev/null}" 2>/dev/null || true
        exit 2
    fi

    if [[ "$FAIL_ON" == "CRITICAL" ]]; then
        log_pass "Passed - No CRITICAL findings"
        exit 0
    fi

    if [[ $HIGH_COUNT -gt 0 ]]; then
        if [[ "$FAIL_ON" == "HIGH" ]] || [[ "$FAIL_ON" == "MEDIUM" ]]; then
            log_high "Pipeline should fail - HIGH findings detected"
            exit 3
        fi
    fi

    if [[ $MEDIUM_COUNT -gt 0 ]] && [[ "$FAIL_ON" == "MEDIUM" ]]; then
        log_medium "Pipeline should fail - MEDIUM findings detected"
        exit 4
    fi

    log_pass "All checks passed!"
    exit 0
}

# Help
if [[ "${1:-}" == "--help" ]] || [[ "${1:-}" == "-h" ]]; then
    cat << 'EOF'
Vibe Prescan - Fast static quality validation

Usage:
  ./scripts/prescan.sh             # Scan all Go files
  ./scripts/prescan.sh recent      # Scan only recently changed files
  ./scripts/prescan.sh <dir>       # Scan specific directory

Environment:
  PRESCAN_FAIL_ON=HIGH             # Threshold (CRITICAL, HIGH, MEDIUM)
  PRESCAN_COMPLEXITY_THRESHOLD=15  # Max cyclomatic complexity

Exit codes:
  0 - All checks passed
  1 - Error running checks
  2 - CRITICAL findings detected
  3 - HIGH severity findings detected
  4 - MEDIUM severity findings detected
EOF
    exit 0
fi

main
