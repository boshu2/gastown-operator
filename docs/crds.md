# CRD Reference

Complete reference for Gas Town Operator Custom Resource Definitions.

## Rig

**Scope:** Cluster
**API Version:** `gastown.gastown.io/v1alpha1`
**Kind:** `Rig`

A Rig represents a project workspace in Gas Town. Rigs are cluster-scoped because they represent physical filesystem paths on the node.

### Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `gitURL` | string | Yes | Git repository URL |
| `beadsPrefix` | string | Yes | Prefix for beads issues (e.g., "frac") |
| `localPath` | string | Yes | Filesystem path to rig (e.g., `/Users/you/gt/fractal`) |
| `settings.namepoolTheme` | string | No | Theme for polecat names (default: "mad-max") |
| `settings.maxPolecats` | int | No | Maximum concurrent polecats (default: 10) |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase: `Initializing`, `Ready`, `Degraded` |
| `polecatCount` | int | Number of active polecats |
| `activeConvoys` | int | Number of in-progress convoys |
| `lastSyncTime` | time | Last successful sync with gt CLI |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: fractal
spec:
  gitURL: "git@github.com:myorg/fractal.git"
  beadsPrefix: "frac"
  localPath: "/Users/me/gt/fractal"
  settings:
    namepoolTheme: "mad-max"
    maxPolecats: 5
```

---

## Polecat

**Scope:** Namespaced
**API Version:** `gastown.gastown.io/v1alpha1`
**Kind:** `Polecat`

A Polecat is an autonomous worker that executes beads issues. Polecats run in tmux sessions on the host.

### Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `rig` | string | Yes | Name of the parent Rig |
| `desiredState` | string | Yes | Target state: `Idle`, `Working`, `Terminated` |
| `beadID` | string | No | Bead ID to work on (required when `desiredState=Working`) |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase: `Pending`, `Working`, `Idle`, `Terminated`, `Failed` |
| `assignedBead` | string | Currently assigned bead ID |
| `branch` | string | Git branch for this polecat's work |
| `tmuxSession` | string | Tmux session name (e.g., `gt-fractal-furiosa`) |
| `sessionActive` | bool | Whether tmux session is running |
| `cleanupStatus` | string | Status during termination: `Pending`, `InProgress`, `Complete` |
| `conditions` | []Condition | Standard Kubernetes conditions |

### State Transitions

```
         ┌─────────────┐
         │   Pending   │
         └──────┬──────┘
                │ (gt sling succeeds)
                ▼
         ┌─────────────┐
    ┌───▶│   Working   │◀───┐
    │    └──────┬──────┘    │
    │           │           │
    │ (new bead)│ (bead     │ (new bead)
    │           │ completes)│
    │           ▼           │
    │    ┌─────────────┐    │
    └────│    Idle     │────┘
         └──────┬──────┘
                │ (desiredState=Terminated)
                ▼
         ┌─────────────┐
         │ Terminated  │
         └─────────────┘
```

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: furiosa
  namespace: default
spec:
  rig: fractal
  desiredState: Working
  beadID: "frac-abc-123"
```

---

## Convoy

**Scope:** Namespaced
**API Version:** `gastown.gastown.io/v1alpha1`
**Kind:** `Convoy`

A Convoy tracks a batch of beads for coordinated execution. Useful for wave-based implementation patterns.

### Spec

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `description` | string | Yes | Human-readable description |
| `trackedBeads` | []string | Yes | List of bead IDs to track |
| `notifyOnComplete` | bool | No | Send notification when all beads complete |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | Current phase: `Pending`, `InProgress`, `Completed`, `Failed` |
| `progress` | string | Progress indicator (e.g., "3/5 complete") |
| `completedBeads` | []string | Beads that have been closed |
| `pendingBeads` | []string | Beads still in progress |
| `beadsConvoyID` | string | ID from beads system (`gt convoy create`) |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Convoy
metadata:
  name: wave-1
  namespace: default
spec:
  description: "Wave 1: Core infrastructure"
  trackedBeads:
    - "frac-abc-123"
    - "frac-def-456"
    - "frac-ghi-789"
  notifyOnComplete: true
```

---

## Common Condition Types

All CRDs use standard Kubernetes conditions:

| Type | Description |
|------|-------------|
| `Ready` | Resource is fully reconciled and operational |
| `Synced` | Last sync with gt CLI succeeded |
| `Degraded` | Resource is operational but with issues |
| `Progressing` | Resource is being updated |

### Condition Example

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-01-11T10:00:00Z"
      reason: ReconcileSuccess
      message: "Rig is ready"
    - type: Synced
      status: "True"
      lastTransitionTime: "2026-01-11T10:00:00Z"
      reason: SyncComplete
      message: "Last sync: 2026-01-11T10:00:00Z"
```
