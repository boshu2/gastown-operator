# ADR-003: Finalizer Cleanup Strategy

**Status**: Accepted
**Date**: 2026-01-01

## Context

When a Polecat CRD is deleted, external resources may exist:

- Local tmux sessions
- Git branches with uncommitted work
- Kubernetes Pods (in kubernetes mode)

Without proper cleanup, these resources would be orphaned.

## Decision

Use **Kubernetes finalizers** to ensure cleanup before CRD deletion.

```go
const polecatFinalizer = "gastown.io/polecat-cleanup"
```

When a Polecat is marked for deletion:

1. Check for finalizer
2. If present, run cleanup logic
3. Only remove finalizer after cleanup succeeds
4. Kubernetes then completes deletion

## Implementation

### Cleanup Steps

**Local Mode**:
1. Check for uncommitted work (`git status`)
2. If uncommitted work exists:
   - Option A: Block deletion (safety first)
   - Option B: Commit work, then cleanup
3. Terminate tmux session (`gt polecat nuke --force`)
4. Remove finalizer

**Kubernetes Mode**:
1. Delete associated Pod (if exists)
2. Wait for Pod termination
3. Remove finalizer

### Safety Checks

Never terminate a polecat with uncommitted work unless `--force` is specified:

```go
if hasUncommittedWork && !force {
    return ctrl.Result{RequeueAfter: time.Minute},
           errors.New("cannot delete polecat with uncommitted work")
}
```

## Consequences

### Positive

- **No orphaned resources**: All external resources cleaned up
- **Data safety**: Uncommitted work not lost silently
- **Graceful shutdown**: Polecats can complete in-flight work

### Negative

- **Deletion can hang**: If cleanup fails, CRD can't be deleted
- **Complexity**: Need to handle all failure modes

### Mitigations

- Timeout on cleanup operations
- Manual override via `kubectl patch` to remove finalizer
- Clear error messages when deletion is blocked

## Failure Handling

| Scenario | Behavior |
|----------|----------|
| Tmux session already gone | Continue with finalizer removal |
| Pod already deleted | Continue with finalizer removal |
| Git push fails | Retry, then proceed (work in local branch) |
| Uncommitted work | Block deletion, require force |
| Network timeout | Retry with exponential backoff |

## Manual Recovery

If a Polecat is stuck in deletion:

```bash
# Check what's blocking
kubectl describe polecat <name>

# Force remove finalizer (data loss risk)
kubectl patch polecat <name> --type=merge \
  -p '{"metadata":{"finalizers":[]}}'
```

## Alternatives Considered

### No Finalizers

**Rejected because**:
- Resources would be orphaned
- No way to ensure cleanup
- Poor user experience

### Owner References Only

**Rejected because**:
- Doesn't cover local resources (tmux)
- No opportunity for safety checks
- Garbage collection is fire-and-forget
