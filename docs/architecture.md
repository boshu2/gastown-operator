# Architecture

How the Gas Town Operator works and why it's designed this way.

## Core Principle: CRDs as Views

**The `gt` CLI is the source of truth. CRDs are views into that truth.**

```
┌─────────────────────────────────────────────────────────┐
│                    Kubernetes                            │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐                  │
│  │   Rig   │  │ Polecat │  │ Convoy  │   ← CRDs        │
│  │   CRD   │  │   CRD   │  │   CRD   │     (Views)     │
│  └────┬────┘  └────┬────┘  └────┬────┘                  │
│       │            │            │                        │
│       └────────────┼────────────┘                        │
│                    │                                     │
│            ┌───────┴───────┐                             │
│            │   Operator    │                             │
│            │  Controllers  │                             │
│            └───────┬───────┘                             │
└────────────────────┼────────────────────────────────────┘
                     │
                     │ shell exec
                     ▼
              ┌─────────────┐
              │   gt CLI    │  ← Source of Truth
              └──────┬──────┘
                     │
                     ▼
              ┌─────────────┐
              │ Kubernetes  │  (Pods, Secrets, Git)
              │  + Beads    │
              └─────────────┘
```

### Why This Pattern?

1. **gt CLI is mature** - It handles all the complexity of git branches, beads sync
2. **Operator adds orchestration** - K8s-native scheduling, conditions, events
3. **No state duplication** - We query gt CLI, don't maintain parallel state
4. **Graceful degradation** - If operator is down, gt CLI still works

## Controllers

### Rig Controller

**Purpose:** Sync Rig CRD status with actual rig state on filesystem.

**Reconcile Loop:**
1. Verify `spec.localPath` exists on filesystem
2. Call `gt rig status <name>` to get current state
3. Update CRD status with polecat count, convoy count
4. Set conditions (Ready, Synced, Degraded)

**Does NOT:**
- Create directories
- Initialize git repos
- Manage gt configuration

### Polecat Controller

**Purpose:** Manage polecat lifecycle through state machine.

**States:**
- `Pending` → Initial state, waiting for gt sling
- `Working` → Actively working on a bead
- `Idle` → Work complete, available for new work
- `Terminated` → Cleanup complete, resource can be deleted

**Reconcile Loop:**
1. Read `spec.desiredState`
2. Compare with `status.phase`
3. Execute transition:
   - `Idle → Working`: Call `gt sling <beadID> <rig>`
   - `Working → Idle`: Poll for bead completion
   - `* → Terminated`: Call `gt polecat nuke` (respects uncommitted work)

**Safety:**
- Never terminates polecat with uncommitted work
- Reports cleanup status in conditions

### Convoy Controller

**Purpose:** Create and track convoy progress in beads system.

**Reconcile Loop:**
1. If no `beadsConvoyID`, call `gt convoy create`
2. Poll `gt convoy status` for progress
3. Update `completedBeads` and `pendingBeads` lists
4. When all complete, set phase to `Completed`
5. If `notifyOnComplete`, send notification

## Sync Patterns

### Pull-Based Sync (Default)

Controllers poll gt CLI on each reconcile:

```go
func (r *RigReconciler) Reconcile(ctx context.Context, req ctrl.Request) {
    // Every reconcile queries gt CLI for current state
    status, err := r.GTClient.RigStatus(ctx, req.Name)
    // Update CRD status from gt response
}
```

### External Change Detection

The `BeadsSyncController` handles changes made outside Kubernetes:

```go
// Polls for changes every 30 seconds
func (r *BeadsSyncReconciler) detectExternalChanges() {
    // Compare CRD state with gt CLI state
    // If different, trigger reconcile for affected resources
}
```

This handles:
- Polecats created via `gt sling` (not through CRD)
- Beads closed via `bd close` (convoy progress)
- Rig changes via `gt rig` commands

## Configuration

### Operator Configuration

| Env Variable | Default | Description |
|--------------|---------|-------------|
| `GT_TOWN_ROOT` | `~/gt` | Path to Gas Town root |
| `GT_PATH` | `gt` | Path to gt binary |

### Helm Values

```yaml
gtConfig:
  townRoot: "/home/user/workspaces"
  gtBinary: "/usr/local/bin/gt"

volumes:
  enabled: true
  hostPath: "/home/user/workspaces"
```

## Metrics

Operator exposes Prometheus metrics:

### Reconciliation Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `gastown_reconcile_total` | Counter | Total reconciliations by controller |
| `gastown_reconcile_errors_total` | Counter | Failed reconciliations |
| `gastown_reconcile_duration_seconds` | Histogram | Reconcile latency |

### Refinery Metrics (v0.4.2+)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gastown_refinery_merge_total` | Counter | rig, result | Total merge attempts (success/conflict/failure) |
| `gastown_refinery_merge_duration_seconds` | Histogram | rig | Time to complete merge operation |
| `gastown_refinery_conflicts_total` | Counter | rig | Merge conflicts detected |
| `gastown_refinery_queue_length` | Gauge | rig | Current merge queue depth |

### Phase Gauges (v0.4.2+)

| Metric | Type | Labels | Description |
|--------|------|--------|-------------|
| `gastown_rig_phase` | Gauge | rig, phase | Current rig phase (1=active) |
| `gastown_polecat_phase` | Gauge | rig, name, phase | Current polecat phase (1=active) |
| `gastown_convoy_phase` | Gauge | rig, name, phase | Current convoy phase (1=active) |

### CLI Metrics

| Metric | Type | Description |
|--------|------|-------------|
| `gastown_gt_cli_calls_total` | Counter | gt CLI invocations |
| `gastown_gt_cli_errors_total` | Counter | gt CLI failures |

## Failure Modes

### gt CLI Not Available

- Controllers return transient errors
- K8s will retry with backoff
- CRD status shows `Degraded` condition

### Filesystem Path Missing

- Rig controller sets `Degraded` condition
- Clear error message in condition
- Does not block other rigs

### Pod Failure

- Polecat controller detects pod termination
- Sets `Failed` phase with reason
- Witness controller tracks stuck polecats with exponential backoff

## Security Considerations

### Kubernetes Execution

The operator runs polecats as Kubernetes pods:
- Claude Code agent runs in each pod
- Git credentials via Kubernetes Secrets
- No host filesystem access required

### RBAC

Minimal cluster permissions:
- Full access to `gastown.gastown.io` CRDs
- Leases for leader election
- Events for status reporting

No access to:
- Secrets
- ConfigMaps
- Pods (doesn't create workloads)
