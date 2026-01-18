# ADR-001: CRDs as Views Pattern

**Status**: Accepted
**Date**: 2026-01-01

## Context

The Gas Town operator needs to expose the state of local gt CLI operations to Kubernetes. This state includes rigs, polecats, convoys, and other Gas Town concepts.

Two primary approaches exist:

1. **CRDs as Source of Truth**: Store all state in Kubernetes, sync to local filesystem
2. **CRDs as Views**: Local `gt` CLI is source of truth, CRDs expose state for orchestration

## Decision

We adopt the **CRDs as Views** pattern:

- The `gt` CLI and local filesystem remain the authoritative source of truth
- Kubernetes CRDs provide a view into this state
- Controllers query `gt` CLI via exec on each reconciliation
- No state is duplicated or cached in the operator

## Architecture

```
┌─────────────────────────────────────────┐
│   Kubernetes (Cluster)                  │
│   ┌─────────────────────────────────┐   │
│   │  Rig, Polecat, Convoy CRDs      │   │
│   │  (View Layer - No State)        │   │
│   └──────────────┬──────────────────┘   │
└──────────────────┼──────────────────────┘
                   │ query on each reconcile
                   ▼
           ┌────────────────┐
           │   gt CLI       │ ← Source of Truth
           │ (gastown)      │
           └────────┬───────┘
                    │
                    ▼
           ┌────────────────┐
           │  Filesystem    │
           │  ~/.gt/, tmux, │
           │  beads, git    │
           └────────────────┘
```

## Consequences

### Positive

- **No state divergence**: Impossible for K8s and local state to drift
- **Graceful degradation**: If operator is down, local tools still work
- **Simpler recovery**: No complex reconciliation to restore state
- **Familiar tooling**: Users can still use `gt` CLI directly
- **Testability**: Easy to test gt CLI independently

### Negative

- **Latency**: Each reconciliation requires exec to gt CLI
- **Dependency**: Operator requires gt CLI to be installed and configured
- **Limited scale**: gt CLI designed for single-machine use

### Mitigations

- Circuit breaker prevents cascading failures on gt CLI issues
- Configurable timeouts prevent hung reconciliations
- Metrics track gt CLI latency and errors

## Alternatives Considered

### CRDs as Source of Truth

**Rejected because**:
- Would require syncing all state to Kubernetes
- Complex conflict resolution for concurrent modifications
- Breaks existing gt CLI workflows
- Users would need to learn new tools

### Hybrid (Partial Caching)

**Rejected because**:
- Introduces cache invalidation complexity
- Potential for stale data issues
- Unclear which source to trust during conflicts
