#!/bin/bash
# Validate that GOALS.yaml check commands can actually fail.
# A check that always passes (e.g., due to platform-specific grep flags)
# provides false confidence. This script tests representative check
# patterns against known-bad input to verify they return non-zero.
#
# Added after evolve cycle 8 discovered grep -oP silently failing on macOS.

set -e

PASS=0
FAIL=0

# Helper: expect a command to FAIL (non-zero exit)
expect_fail() {
  local desc="$1"
  shift
  if eval "$@" >/dev/null 2>&1; then
    echo "FAIL: $desc — expected non-zero exit but got 0"
    FAIL=$((FAIL + 1))
  else
    PASS=$((PASS + 1))
  fi
}

# Helper: expect a command to PASS (zero exit)
expect_pass() {
  local desc="$1"
  shift
  if eval "$@" >/dev/null 2>&1; then
    PASS=$((PASS + 1))
  else
    echo "FAIL: $desc — expected zero exit but got non-zero"
    FAIL=$((FAIL + 1))
  fi
}

# --- Coverage pipeline: must reject below-threshold values ---
expect_fail "coverage rejects 5%" \
  "echo 'coverage: 5.0' | grep -oE 'coverage: [0-9]+\.[0-9]+' | grep -oE '[0-9]+\.[0-9]+' | awk '{exit (\$1 < 60)}'"

expect_pass "coverage accepts 85%" \
  "echo 'coverage: 85.0' | grep -oE 'coverage: [0-9]+\.[0-9]+' | grep -oE '[0-9]+\.[0-9]+' | awk '{exit (\$1 < 60)}'"

# --- grep -q: must fail when pattern is absent ---
expect_fail "grep -q fails on missing pattern" \
  "echo 'no match here' | grep -q 'EventRecorder'"

expect_pass "grep -q passes on present pattern" \
  "echo 'has EventRecorder here' | grep -q 'EventRecorder'"

# --- friction-points count: must fail when count is below threshold ---
expect_fail "doc count rejects 0" \
  "test 0 -ge 9"

expect_pass "doc count accepts 10" \
  "test 10 -ge 9"

# --- Report ---
echo ""
echo "Goal check validation: $PASS passed, $FAIL failed"
test "$FAIL" -eq 0
