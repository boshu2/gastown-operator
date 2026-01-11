# Plan: Fix Vibe Check Findings

**Date:** 2026-01-11
**Target:** gastown-operator
**Priority:** P1 (HIGH finding requires fix before production)

## Summary

Fix all issues identified in vibe semantic validation:
- 1 HIGH severity (P8: Silent error handling)
- 1 MEDIUM severity (P2: Test coverage gaps)

## Wave Structure

### Wave 1: Fix HIGH - Silent Error in ensureIdle (P8)

**Issue:** `internal/controller/polecat_controller.go:177-179`

Current code:
```go
if err := r.GTClient.PolecatReset(ctx, polecat.Spec.Rig, polecat.Name); err != nil {
    log.Error(err, "Failed to reset polecat")
    // Missing: status condition update and requeue
}
```

**Fix:** Update to match pattern from ensureTerminated (lines 232-243):
```go
if err := r.GTClient.PolecatReset(ctx, polecat.Spec.Rig, polecat.Name); err != nil {
    log.Error(err, "Failed to reset polecat")
    r.setCondition(polecat, ConditionPolecatReady, metav1.ConditionFalse, "ResetFailed",
        err.Error())

    if updateErr := r.Status().Update(ctx, polecat); updateErr != nil {
        timer.RecordResult(metrics.ResultError)
        return ctrl.Result{}, gterrors.Wrap(updateErr, "failed to update polecat status")
    }

    timer.RecordResult(metrics.ResultRequeue)
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

**Files:**
- `internal/controller/polecat_controller.go`

**Validation:**
- `make build` passes
- `make test` passes (existing tests)

---

### Wave 2: Create Mock GT Client Interface (P2 prerequisite)

**Issue:** Tests can't run without gt CLI. Need mock interface.

**Tasks:**

1. Create `pkg/gt/interface.go`:
```go
type ClientInterface interface {
    // Rig operations
    RigList(ctx context.Context) ([]RigInfo, error)
    RigStatus(ctx context.Context, name string) (*RigStatus, error)
    RigExists(ctx context.Context, name string) (bool, error)

    // Polecat operations
    Sling(ctx context.Context, beadID, rig string) error
    PolecatList(ctx context.Context, rig string) ([]PolecatInfo, error)
    PolecatStatus(ctx context.Context, rig, name string) (*PolecatStatus, error)
    PolecatNuke(ctx context.Context, rig, name string, force bool) error
    PolecatReset(ctx context.Context, rig, name string) error
    PolecatExists(ctx context.Context, rig, name string) (bool, error)

    // Convoy operations
    ConvoyCreate(ctx context.Context, description string, beadIDs []string) (string, error)
    ConvoyStatus(ctx context.Context, id string) (*ConvoyStatus, error)
    ConvoyList(ctx context.Context) ([]ConvoyInfo, error)

    // Hook operations
    Hook(ctx context.Context, beadID, assignee string) error
    HookStatus(ctx context.Context, assignee string) (*HookInfo, error)

    // Beads operations
    BeadStatus(ctx context.Context, beadID string) (*BeadStatus, error)

    // Mail operations
    MailSend(ctx context.Context, address, subject, message string) error
}
```

2. Create `pkg/gt/mock_client.go`:
```go
type MockClient struct {
    // Configurable responses
    RigListFunc      func(ctx context.Context) ([]RigInfo, error)
    PolecatStatusFunc func(ctx context.Context, rig, name string) (*PolecatStatus, error)
    // ... etc
}
```

3. Update controller structs to use interface:
```go
type RigReconciler struct {
    client.Client
    Scheme   *runtime.Scheme
    GTClient gt.ClientInterface  // Changed from *gt.Client
}
```

**Files:**
- `pkg/gt/interface.go` (new)
- `pkg/gt/mock_client.go` (new)
- `internal/controller/rig_controller.go`
- `internal/controller/polecat_controller.go`
- `internal/controller/convoy_controller.go`
- `internal/controller/beadssync_controller.go`
- `cmd/local/main.go`

**Validation:**
- `make build` passes
- Existing behavior unchanged

---

### Wave 3: Implement Controller Tests (P2)

**Issue:** All controller tests skip with TODO.

**Tasks:**

1. `internal/controller/rig_controller_test.go`:
   - Test reconcile with existing rig path
   - Test reconcile with missing rig path (degraded)
   - Test reconcile with gt CLI error
   - Test status sync from gt response

2. `internal/controller/polecat_controller_test.go`:
   - Test ensureWorking creates polecat via sling
   - Test ensureWorking syncs status
   - Test ensureIdle resets working polecat
   - Test ensureIdle handles reset failure (the fix from Wave 1)
   - Test ensureTerminated refuses with uncommitted work
   - Test ensureTerminated nukes clean polecat

3. `internal/controller/convoy_controller_test.go`:
   - Test creates convoy in beads system
   - Test syncs progress from gt CLI
   - Test sends notification on completion
   - Test doesn't requeue after completion

**Files:**
- `internal/controller/rig_controller_test.go`
- `internal/controller/polecat_controller_test.go`
- `internal/controller/convoy_controller_test.go`

**Validation:**
- `make test` passes
- Coverage improved (target: >70% for controllers)

---

## Execution Order

```
Wave 1 ─────────────────────────────────────────────────┐
(Fix P8 - ensureIdle error handling)                    │
                                                        │
Wave 2 ─────────────────────────────────────────────────┼──▶ Wave 3
(Create mock interface)                                 │    (Implement tests)
                                                        │
```

Wave 1 and Wave 2 can run in parallel.
Wave 3 depends on Wave 2 (needs mock client).

---

## Acceptance Criteria

- [ ] `polecat_controller.go` ensureIdle handles reset errors properly
- [ ] `pkg/gt/interface.go` defines ClientInterface
- [ ] `pkg/gt/mock_client.go` implements MockClient
- [ ] All controller tests implemented (not skipped)
- [ ] `make build` passes
- [ ] `make test` passes
- [ ] Test coverage > 70% for internal/controller/

---

## Commands

```bash
# Validate Wave 1
cd /Users/fullerbt/gt/gastown-operator
make build && make test

# Run specific controller tests
go test ./internal/controller/... -v

# Check coverage
go test ./internal/controller/... -coverprofile=coverage.out
go tool cover -func=coverage.out | grep -E "(rig|polecat|convoy)_controller"
```
