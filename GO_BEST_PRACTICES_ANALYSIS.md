# Go Best Practices Analysis — Gastown Operator

**Date:** 2026-02-07
**Scope:** Full codebase (~9,400 lines of Go across 67 files)
**Model:** Claude Opus 4.6 — 6-agent parallel analysis

---

## Executive Summary

| Area | Grade | Critical Issues |
|------|-------|-----------------|
| Error Handling | A | 0 |
| Concurrency & Safety | A+ | 0 |
| API Design & Types | A- | 2 medium |
| Testing Patterns | B+ | 3 medium |
| Project Structure | A+ | 0 |
| Performance & Resources | B+ | 1 high, 2 medium |

**Overall: A — Production-grade operator with targeted improvements possible.**

---

## 1. Error Handling

### Strengths

- **Consistent `%w` wrapping** across all controllers and git operations
- **Custom `GasTownError` type** (`pkg/errors/errors.go`) with stack traces, error categorization (Transient/Permanent/Validation/NotFound/Conflict), retryable flag, and context map
- **Proper `errors.As()` usage** — no raw type assertions on errors found
- **Reconcile loop pattern** correctly uses `ctrl.Result{RequeueAfter: ...}, nil` for transient errors and `ctrl.Result{}, err` for fatal errors
- **No double logging** — errors logged once, then either returned or requeued
- **Deferred cleanup** uses `_ = Close()` with `//nolint:errcheck` comments for intent clarity

### Issues

| # | Severity | Location | Description |
|---|----------|----------|-------------|
| E1 | Low | `witness_controller.go:116,180` | `Status().Update()` errors returned without context wrapping |
| E2 | Low | `refinery_controller.go:110,139,186` | Same — bare `return ctrl.Result{}, err` after status update |
| E3 | Low | `pkg/client/*.go` (multiple) | CRUD methods pass through dynamic client errors without wrapping |
| E4 | Low | `rig_controller.go:368,379` | `IndexField` errors returned without context |

### Recommended Fix (E1/E2)

```go
// Before
return ctrl.Result{}, r.Status().Update(ctx, witness)

// After
if err := r.Status().Update(ctx, witness); err != nil {
    return ctrl.Result{}, gterrors.Wrap(err, "failed to update witness status")
}
return ctrl.Result{RequeueAfter: healthCheckInterval}, nil
```

---

## 2. Concurrency & Goroutine Safety

### Strengths

- **Zero explicit goroutines** — all concurrency delegated to controller-runtime
- **Proper context propagation** throughout the entire call chain
- **`sync.RWMutex`** used correctly in `pkg/health/health.go` and `pkg/errors/backoff.go` with consistent `defer Unlock()` pattern
- **MaxConcurrentReconciles tuned per controller:**

| Controller | Max Concurrent | Rationale |
|------------|---------------|-----------|
| Rig | 3 | Cluster-scoped, limit concurrency |
| Polecat | 5 | Pod creation bottleneck |
| Convoy | 3 | Batch processing |
| Witness | 2 | Lightweight monitors |
| Refinery | 2 | Merges benefit from serialization |
| BeadStore | 1 | Singleton config |

- **Rate limiter** combines exponential backoff (5ms–5min) with token bucket (10/s, burst 100)
- **`GenerationChangedPredicate`** on all 6 controllers prevents spurious reconciliations
- **Tickers properly stopped** with `defer ticker.Stop()` in CLI commands

### Issues

None critical. One theoretical concern:

| # | Severity | Location | Description |
|---|----------|----------|-------------|
| C1 | Info | `backoff.go:107` | `Cleanup(activeKeys)` receives external map — safe if caller doesn't mutate concurrently (current usage is safe) |

---

## 3. API Design & Type Safety

### Strengths

- **Small, focused interfaces**: `RigClient`, `PolecatClient`, `ConvoyClient` (3-6 methods each)
- **Interfaces defined by consumers** in `pkg/client/` and `internal/git/interface.go`
- **Consistent pointer receivers** across all implementations
- **Proper `New*` constructors** and builder pattern (`git.NewClient().WithSSHKey()`)
- **Kubernetes API conventions** followed: Spec/Status separation, `metav1.Condition`, finalizers, owner references
- **Naming conventions** correct: `APIKeySecretRef`, `SSHKnownHostsConfigMapRef`, `BeadID`
- **CRD validation** via CEL markers — no webhook overhead

### Issues

| # | Severity | Location | Description |
|---|----------|----------|-------------|
| A1 | Medium | `witness_types.go`, `refinery_types.go`, `beadstore_types.go` | Field doc comments use lowercase (`rigRef references...`) instead of exported name (`RigRef references...`) — violates godoc conventions |
| A2 | Medium | `witness_controller.go:60-62` | Inline anonymous interface in struct — should extract to named `GTMailer` interface |
| A3 | Low | `rig_controller.go:365,373` | Unsafe type assertions `rawObj.(*Polecat)` in field indexers — should use comma-ok pattern |
| A4 | Low | `polecat_types.go:263`, `rig_types.go:56`, `convoy_types.go:49` | Phase enum types lack `String()` methods (works due to underlying string type, but explicit is better) |

### Recommended Fix (A2)

```go
// Before (inline anonymous interface)
GTClient interface {
    MailSend(ctx context.Context, address, subject, message string) error
}

// After (named interface at package level)
type GTMailer interface {
    MailSend(ctx context.Context, address, subject, message string) error
}

type WitnessReconciler struct {
    // ...
    GTClient GTMailer
}
```

---

## 4. Testing Patterns

### Strengths

- **Clear test type separation**: `*_unit_test.go` (fake client), `*_test.go` (envtest), `test/e2e/` (Kind)
- **Excellent table-driven tests** with custom validation functions per case
- **Proper test helpers** with `t.Helper()` calls: `newTestRig()`, `newTestPolecat()`, `newTestWitness()`
- **`fakeGTClient`** mock tracks calls and allows behavior override
- **`WithStatusSubresource()`** correctly configured in fake client builders
- **envtest** properly set up with BeforeSuite/AfterSuite and context cancellation
- **BDD structure** (Ginkgo) for integration tests with Describe/Context/It hierarchy

### Issues

| # | Severity | Location | Description |
|---|----------|----------|-------------|
| T1 | Medium | `polecat_controller_test.go:165` | `Eventually()` without explicit timeout — could hang in CI |
| T2 | Medium | `metrics_reconcile_test.go:46` | `time.Sleep(10ms)` instead of polling — fragile on slow CI |
| T3 | Medium | `test/e2e/e2e_test.go` | E2E tests only cover happy path — no failure scenarios |
| T4 | Low | `witness_controller_unit_test.go:200` | Hardcoded `time.Now().Add(-20*time.Minute)` — brittle threshold testing |
| T5 | Low | Multiple | No `testdata/` directory for complex YAML fixtures |

### Test Coverage Matrix

| Controller | Unit (fake) | Integration (envtest) | E2E (Kind) |
|-----------|-------------|----------------------|------------|
| Rig | Yes (556 lines) | Yes (372 lines) | Yes |
| Polecat | Yes (788 lines) | Yes (655 lines) | Yes |
| Witness | Yes (854 lines) | No | No |
| Refinery | No | No | No |
| Convoy | No | Yes (327 lines) | No |
| BeadStore | No | Yes (327 lines) | No |

**Gap:** Refinery controller has zero test coverage.

---

## 5. Project Structure & Dependencies

### Strengths

- **Textbook Go project layout**: `cmd/`, `api/`, `internal/`, `pkg/`, `config/`, `test/`, `helm/`
- **Zero circular dependencies** — clean unidirectional dependency graph
- **Zero `replace` directives** in go.mod — all deps from public modules
- **All dependencies pinned** to exact versions; K8s packages aligned at v0.35.0
- **Build tags** use modern `//go:build` syntax
- **Makefile** well-organized with `##@` section headers and centralized tool versioning
- **Code generation** properly integrated: `make manifests`, `make generate`, CI validation
- **VERSION file** as single source of truth
- **Security-aware**: HTTP/2 disabled to prevent stream cancellation CVE

### Issues

None found. This area is exemplary.

---

## 6. Performance & Resource Management

### Strengths

- **Rate limiting** via exponential backoff + token bucket (production-grade)
- **Field indexes** for Rig→Polecat and Rig→Convoy lookups
- **Resource cleanup** with `defer` throughout (temp files, SSH keys, git dirs)
- **Context timeouts** via `WithGTClientTimeout()` helper
- **No expensive operations in log statements**

### Issues

| # | Severity | Location | Description |
|---|----------|----------|-------------|
| P1 | **High** | `pkg/errors/backoff.go:45-46` | `retries map[string]int` grows unbounded — `Cleanup()` method exists but is **never called** from any reconciler. Long-running operators will leak memory. |
| P2 | Medium | `convoy_controller.go:102` | N+1 query: `r.List(ctx, &polecatList)` fetches ALL Polecats cluster-wide instead of using field index by rig |
| P3 | Medium | `convoy_controller.go:125-132`, `refinery_controller.go:232` | Slices appended without pre-allocation when size is known from input |
| P4 | Low | `convoy_controller.go:117` | Map created without capacity hint: `make(map[string]PolecatPhase)` vs `make(map[string]PolecatPhase, len(list))` |
| P5 | Low | `refinery_controller.go:151,163,175` | String concatenation with `+` instead of `fmt.Sprintf()` in condition messages |

### Recommended Fix (P1 — High Priority)

```go
// Option A: Call Cleanup() in WitnessReconciler.Reconcile after processing
activeKeys := make(map[string]bool, len(witnessList.Items))
for _, w := range witnessList.Items {
    activeKeys[w.Name] = true
}
r.Backoff.Cleanup(activeKeys)

// Option B: Add TTL-based eviction to BackoffCalculator
type retryEntry struct {
    count     int
    lastSeen  time.Time
}
```

### Recommended Fix (P2)

```go
// Before — fetches ALL Polecats
var polecatList gastownv1alpha1.PolecatList
r.List(ctx, &polecatList)

// After — use field index (requires index registration in SetupWithManager)
r.List(ctx, &polecatList, client.MatchingFields{"spec.rig": convoy.Spec.RigRef})
```

---

## Prioritized Action Items

### Must Fix

| # | Area | Description | Files |
|---|------|-------------|-------|
| P1 | Performance | Call `BackoffCalculator.Cleanup()` or add TTL eviction | `backoff.go`, `witness_controller.go` |

### Should Fix

| # | Area | Description | Files |
|---|------|-------------|-------|
| P2 | Performance | Add field index for Convoy→Polecat queries | `convoy_controller.go` |
| A1 | API Design | Fix godoc field comments to use PascalCase names | `witness_types.go`, `refinery_types.go`, `beadstore_types.go` |
| A2 | API Design | Extract anonymous interface to named `GTMailer` | `witness_controller.go` |
| T1 | Testing | Add explicit timeout to `Eventually()` calls | `polecat_controller_test.go` |
| T3 | Testing | Add E2E error/failure scenarios | `test/e2e/e2e_test.go` |

### Nice to Have

| # | Area | Description | Files |
|---|------|-------------|-------|
| E1-E2 | Errors | Wrap `Status().Update()` errors with context | Controllers |
| A3 | Types | Use comma-ok type assertions in field indexers | `rig_controller.go` |
| P3 | Performance | Pre-allocate slices when size is known | Controllers |
| T2 | Testing | Replace `time.Sleep` with polling | `metrics_reconcile_test.go` |
| T4 | Testing | Mock `time.Now()` for threshold tests | `witness_controller_unit_test.go` |

---

## Patterns Worth Preserving

These patterns are exemplary and should be maintained in new code:

1. **Custom `GasTownError` with stack traces and categorization** — enables automatic condition reason mapping via `ToConditionReason()`
2. **BackoffCalculator circuit breaker** — prevents escalation storms in Witness
3. **Requeue-based error handling** — `ctrl.Result{RequeueAfter: ...}, nil` for transient failures
4. **Interface-based DI** — `GitClientFactory` enables clean testing without circular imports
5. **`GenerationChangedPredicate`** on all controllers — eliminates spurious reconciliations
6. **Dual rate limiter** — exponential backoff + token bucket prevents thundering herd
7. **Field indexes** for cross-resource lookups — efficient Rig→Polecat queries
8. **`internal/` encapsulation** — controllers and git operations properly hidden
9. **Table-driven tests with validation functions** — flexible, readable test cases
10. **`t.Helper()` in test factories** — clean test output with correct line numbers
