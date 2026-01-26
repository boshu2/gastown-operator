# CRD Reference

Complete reference for Gas Town Operator Custom Resource Definitions.

**API Group:** `gastown.gastown.io`
**Version:** `v1alpha1`

---

## Rig

**Scope:** Cluster

A Rig represents a project workspace in Gas Town. Rigs are cluster-scoped because they represent physical filesystem paths on the node.

### Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `gitURL` | string | Yes | - | Git repository URL |
| `beadsPrefix` | string | Yes | - | Prefix for beads issues (e.g., "gt", "at"). Pattern: `^[a-z]{2,10}$` |
| `localPath` | string | Yes | - | Filesystem path to rig (e.g., `/home/user/workspaces/myproject`) |
| `settings.namepoolTheme` | string | No | - | Theme for polecat names (e.g., "mad-max") |
| `settings.maxPolecats` | int | No | `8` | Maximum concurrent polecats (1-100) |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Initializing`, `Ready`, `Degraded` |
| `polecatCount` | int | Number of polecats in this rig |
| `activeConvoys` | int | Number of in-progress convoys |
| `lastSyncTime` | timestamp | Last sync with gt CLI |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Ready Condition (v0.4.2+)

The Rig Ready condition is **aggregated** from child resources:

| Child Resource | Contributes to Ready |
|----------------|---------------------|
| Witness | Must be Active |
| Refinery | Must be Idle or Processing |
| Polecats | At least one must be Working or Done |

**Ready=True** requires:
1. Witness exists and is Active
2. Refinery exists and is not in Error phase
3. At least one polecat has completed successfully OR is actively working

**Ready=False** reasons:
- `WitnessNotReady`: Witness is missing or Degraded
- `RefineryError`: Refinery is in Error phase
- `NoPolecatActivity`: No polecats have completed work

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: myproject
spec:
  gitURL: "git@github.com:myorg/myproject.git"
  beadsPrefix: "mp"
  localPath: "/home/user/workspaces/myproject"
  settings:
    namepoolTheme: "mad-max"
    maxPolecats: 8
```

---

## Polecat

**Scope:** Namespaced

A Polecat is an autonomous worker agent that executes beads issues. Polecats run as Kubernetes Pods.

### Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rig` | string | Yes | - | Name of the parent Rig |
| `desiredState` | string | Yes | `Idle` | Target state: `Idle`, `Working`, `Terminated` |
| `beadID` | string | No | - | Bead ID to work on (triggers work when set) |
| `taskDescription` | string | No | - | Explicit task description for Claude (use when beads not synced) |
| `executionMode` | string | No | `kubernetes` | Where to run (kubernetes only) |
| `agent` | string | No | `claude-code` | Agent type (claude-code only) |
| `agentConfig` | object | No | - | Configuration for the coding agent |
| `kubernetes` | object | No* | - | Kubernetes execution config (*required if `executionMode=kubernetes`) |
| `resources` | ResourceRequirements | No | - | CPU/memory for the polecat pod |
| `ttlSecondsAfterFinished` | int32 | No | - | How long a completed polecat persists |
| `maxIdleSeconds` | int32 | No | - | Terminates polecat if idle for this duration |

### KubernetesSpec (for `executionMode: kubernetes`)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `gitRepository` | string | Yes | - | Git repo URL (SSH or HTTPS) |
| `gitBranch` | string | No | `main` | Branch to checkout |
| `workBranch` | string | No | `feature/<beadID>` | Branch name to create for work |
| `gitSecretRef.name` | string | Yes | - | Secret containing SSH key for git |
| `claudeCredsSecretRef.name` | string | No* | - | Secret containing ~/.claude/ contents (*required unless `apiKeySecretRef` provided) |
| `apiKeySecretRef` | SecretKeyRef | No* | - | Secret containing API key (*alternative to `claudeCredsSecretRef`) |
| `image` | string | No | - | Override agent container image |
| `resources` | ResourceRequirements | No | - | CPU/memory for agent container |
| `activeDeadlineSeconds` | int64 | No | `3600` | Max runtime before Pod termination |

### AgentConfig (for custom agent configuration)

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `provider` | string | No | `litellm` | LLM provider: `litellm`, `anthropic`, `openai`, `ollama` |
| `model` | string | No | - | Model name/ID (e.g., "claude-sonnet-4", "devstral-123b") |
| `modelProvider.endpoint` | string | No | - | API base URL (e.g., https://ai-gateway.example.com/v1) |
| `modelProvider.apiKeySecretRef` | SecretKeyRef | No | - | Secret containing the API key |
| `image` | string | No | - | Override container image for the agent |
| `command` | []string | No | - | Override entrypoint command |
| `args` | []string | No | - | Additional arguments to the agent command |
| `configMapRef.name` | string | No | - | ConfigMap containing agent configuration |
| `env` | []EnvVar | No | - | Additional environment variables |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Idle`, `Working`, `Done`, `Stuck`, `Terminated` |
| `assignedBead` | string | Currently assigned bead ID |
| `branch` | string | Git branch for this polecat's work |
| `podName` | string | Pod name |
| `podActive` | bool | Whether Pod is running |
| `lastActivity` | timestamp | When polecat last showed activity |
| `cleanupStatus` | string | `clean`, `has_uncommitted`, `has_unpushed`, `unknown` |
| `agent` | string | Agent type currently running |
| `agentImage` | string | Container image being used |
| `agentModel` | string | LLM model being used |
| `conditions` | []Condition | Standard Kubernetes conditions |

### State Transitions

```
         ┌─────────────┐
         │    Idle     │◀────────────────┐
         └──────┬──────┘                 │
                │ (beadID set)           │ (work completes)
                ▼                        │
         ┌─────────────┐          ┌──────┴──────┐
         │   Working   │─────────▶│    Done     │
         └──────┬──────┘          └─────────────┘
                │ (no progress)
                ▼
         ┌─────────────┐
         │    Stuck    │
         └──────┬──────┘
                │ (desiredState=Terminated)
                ▼
         ┌─────────────┐
         │ Terminated  │
         └─────────────┘
```

### Examples

**Kubernetes execution with Claude Code:**

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: claude-worker
  namespace: gastown-system
spec:
  rig: myproject
  desiredState: Working
  beadID: "gt-abc-123"
  executionMode: kubernetes
  agent: claude-code
  kubernetes:
    gitRepository: "git@github.com:myorg/myproject.git"
    gitBranch: main
    workBranch: feature/gt-abc-123
    gitSecretRef:
      name: git-creds
    claudeCredsSecretRef:
      name: claude-creds
    activeDeadlineSeconds: 3600
    resources:
      requests:
        cpu: "500m"
        memory: "1Gi"
      limits:
        cpu: "2"
        memory: "4Gi"
```

**Kubernetes execution with API key (headless):**

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: api-key-worker
  namespace: gastown-system
spec:
  rig: myproject
  desiredState: Working
  beadID: "gt-abc-123"
  executionMode: kubernetes
  agent: claude-code
  kubernetes:
    gitRepository: "git@github.com:myorg/myproject.git"
    gitBranch: main
    gitSecretRef:
      name: git-creds
    apiKeySecretRef:
      name: anthropic-api-key
      key: api-key
```

---

## Convoy

**Scope:** Namespaced

A Convoy tracks a batch of beads for coordinated execution. Used for wave-based implementation patterns.

### Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `description` | string | Yes | - | Human-readable description |
| `trackedBeads` | []string | Yes | - | List of bead IDs to track (min 1) |
| `notifyOnComplete` | string | No | - | Mail address for completion notification |
| `parallelism` | int32 | No | `0` | Max concurrent polecats (0=unlimited) |
| `rigRef` | string | No | - | Rig where polecats will be created |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Pending`, `InProgress`, `Complete`, `Failed` |
| `progress` | string | Progress indicator (e.g., "3/5") |
| `completedBeads` | []string | Beads that have been closed |
| `pendingBeads` | []string | Beads still in progress |
| `beadsConvoyID` | string | ID from beads system |
| `startedAt` | timestamp | When convoy started |
| `completedAt` | timestamp | When convoy completed |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Convoy
metadata:
  name: wave-1
  namespace: gastown-system
spec:
  description: "Wave 1: Core infrastructure"
  trackedBeads:
    - "gt-abc-123"
    - "gt-def-456"
    - "gt-ghi-789"
  parallelism: 3
  rigRef: myproject
  notifyOnComplete: "mayor"
```

---

## Witness

**Scope:** Namespaced
**Olympian API:** Sentinel

A Witness monitors the health of Polecats in a Rig and escalates issues.

### Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rigRef` | string | Yes | - | Rig to monitor |
| `healthCheckInterval` | duration | No | `30s` | How often to check polecat health |
| `stuckThreshold` | duration | No | `15m` | How long idle before considered stuck |
| `escalationTarget` | string | No | `mayor` | Where to send alerts (mayor, slack, email) |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Pending`, `Active`, `Degraded` |
| `lastCheckTime` | timestamp | Last health check timestamp |
| `polecatsSummary.total` | int32 | Total polecats in rig |
| `polecatsSummary.running` | int32 | Actively running polecats |
| `polecatsSummary.succeeded` | int32 | Successfully completed |
| `polecatsSummary.failed` | int32 | Failed polecats |
| `polecatsSummary.stuck` | int32 | Polecats with no progress |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Circuit Breaker (v0.4.2+)

The Witness uses **exponential backoff** for escalation to prevent alert storms:

| Consecutive Failures | Backoff Delay | Behavior |
|---------------------|---------------|----------|
| 1 | 0 (immediate) | First escalation sent immediately |
| 2 | 1 minute | Wait before second escalation |
| 3 | 2 minutes | Doubled backoff |
| 4 | 4 minutes | Doubled again |
| 5+ | 8 minutes (max) | Capped at 8 minute intervals |

**Reset behavior:**
- Successful health check resets the backoff counter
- Polecat returning to Working phase resets for that polecat
- Manual `kubectl gt polecat nuke` followed by re-sling resets

**Status fields for circuit breaker:**

| Field | Type | Description |
|-------|------|-------------|
| `escalationBackoff` | duration | Current backoff delay |
| `lastEscalationTime` | timestamp | When last escalation was sent |
| `consecutiveFailures` | int32 | Failures since last reset |

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Witness
metadata:
  name: myproject-witness
  namespace: gastown-system
spec:
  rigRef: myproject
  healthCheckInterval: 30s
  stuckThreshold: 15m
  escalationTarget: mayor
```

---

## Refinery

**Scope:** Namespaced
**Olympian API:** Crucible

A Refinery processes merge queues for a Rig, sequentially rebasing and merging polecat branches after validation.

### Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rigRef` | string | Yes | - | Rig to process merges for |
| `targetBranch` | string | No | `main` | Branch to merge into |
| `testCommand` | string | No | - | Command to run after rebase for validation |
| `parallelism` | int32 | No | `1` | Concurrent merge processing (sequential by default) |
| `gitSecretRef.name` | string | No | - | Secret containing git credentials |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Idle`, `Processing`, `Error` |
| `queueLength` | int32 | Branches waiting to merge |
| `currentMerge` | string | Branch currently being processed |
| `lastMergeTime` | timestamp | Last successful merge |
| `mergesSummary.total` | int32 | Total merges attempted |
| `mergesSummary.succeeded` | int32 | Successful merges |
| `mergesSummary.failed` | int32 | Failed merges |
| `mergesSummary.pending` | int32 | Branches in queue |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Refinery
metadata:
  name: myproject-refinery
  namespace: gastown-system
spec:
  rigRef: myproject
  targetBranch: main
  testCommand: "make test"
  parallelism: 1
  gitSecretRef:
    name: git-creds
```

---

## BeadStore

**Scope:** Namespaced

A BeadStore manages the configuration for a beads issue tracking database.

### Spec

| Field | Type | Required | Default | Description |
|-------|------|----------|---------|-------------|
| `rigRef` | string | Yes | - | Rig this BeadStore is associated with |
| `prefix` | string | Yes | - | Issue ID prefix (e.g., "gt-"). Pattern: `^[a-z]+-$` |
| `gitSecretRef.name` | string | No | - | Secret containing git credentials for syncing |
| `syncInterval` | duration | No | `5m` | How often to sync with git |

### Status

| Field | Type | Description |
|-------|------|-------------|
| `phase` | string | `Pending`, `Synced`, `Error` |
| `lastSyncTime` | timestamp | Last successful sync |
| `issueCount` | int32 | Number of issues in this beadstore |
| `conditions` | []Condition | Standard Kubernetes conditions |

### Example

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: BeadStore
metadata:
  name: myproject-beads
  namespace: gastown-system
spec:
  rigRef: myproject
  prefix: "gt-"
  syncInterval: 5m
  gitSecretRef:
    name: git-creds
```

---

## Common Patterns

### Condition Types

All CRDs use standard Kubernetes conditions:

| Type | Description |
|------|-------------|
| `Ready` | Resource is fully reconciled and operational |
| `Synced` | Last sync with gt CLI succeeded |
| `Degraded` | Resource is operational but with issues |
| `Progressing` | Resource is being updated |

### SecretReference

Several CRDs reference Secrets for credentials:

```yaml
gitSecretRef:
  name: git-creds  # type: kubernetes.io/ssh-auth with ssh-privatekey
claudeCredsSecretRef:
  name: claude-creds  # type: Opaque with credentials.json, settings.json
```

### Condition Example

```yaml
status:
  conditions:
    - type: Ready
      status: "True"
      lastTransitionTime: "2026-01-17T10:00:00Z"
      reason: ReconcileSuccess
      message: "Resource is ready"
```

---

## Olympian API Mapping

Gas Town uses internal naming. External APIs use Olympian terms:

| Gas Town (Internal) | Olympian (External API) |
|---------------------|-------------------------|
| Rig | Forge |
| Polecat | Automaton |
| Convoy | Phalanx |
| Witness | Sentinel |
| Refinery | Crucible |
