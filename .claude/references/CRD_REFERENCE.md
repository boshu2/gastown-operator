# CRD Reference

Complete API reference for Gas Town Operator custom resources.

---

## Polecat

Autonomous worker agent that executes tasks.

### Spec

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: string              # Unique identifier (lowercase, hyphens)
  namespace: string         # Usually gastown-system
spec:
  # Required
  rig: string               # Parent rig name (must exist)
  desiredState: string      # Idle | Working | Terminated

  # Optional - Task
  beadID: string            # Bead issue to work on (e.g., "at-1234")
  taskDescription: string   # Natural language instructions (if no beadID)

  # Optional - Execution
  executionMode: string     # local (default) | kubernetes
  agent: string             # claude-code (default) | opencode | aider | custom

  # Optional - Agent Config
  agentConfig:
    provider: string        # anthropic | litellm | openai | ollama
    model: string           # Full model ID (e.g., claude-sonnet-4-20250514)
    maxTokens: int          # Max output tokens
    temperature: float      # 0.0-1.0

  # Required for executionMode: kubernetes
  kubernetes:
    gitRepository: string   # SSH URL (git@github.com:org/repo.git)
    gitBranch: string       # Base branch (default: main)
    workBranch: string      # Branch for changes (default: polecat/<name>)
    gitSecretRef:
      name: string          # Secret with ssh-privatekey
    claudeCredsSecretRef:   # Option A: OAuth credentials
      name: string
    apiKeySecretRef:        # Option B: API key
      name: string
      key: string           # Key within secret (default: api-key)
    activeDeadlineSeconds: int  # Timeout (REQUIRED - prevents runaway)
    resources:
      requests:
        cpu: string         # e.g., "500m"
        memory: string      # e.g., "1Gi"
      limits:
        cpu: string         # e.g., "2"
        memory: string      # e.g., "4Gi"
```

### Status

```yaml
status:
  phase: string             # Pending | Idle | Working | Succeeded | Failed | Terminated
  currentState: string      # Actual state (may lag desiredState)
  beadID: string            # Currently assigned bead
  message: string           # Human-readable status
  podName: string           # Pod name (kubernetes mode)
  tmuxSession: string       # Tmux session (local mode)
  startTime: timestamp
  completionTime: timestamp
  conditions:
    - type: string          # Ready | Working | Failed
      status: string        # True | False | Unknown
      reason: string
      message: string
      lastTransitionTime: timestamp
```

### State Transitions

```
Idle ─────► Working ─────► Succeeded
  │            │              │
  │            ▼              │
  │         Failed ◄──────────┘
  │            │
  ▼            ▼
Terminated ◄───┘
```

---

## Convoy

Batch tracking for parallel polecat execution.

### Spec

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Convoy
metadata:
  name: string
  namespace: string
spec:
  description: string       # Human-readable name
  trackedBeads:             # List of bead IDs to track
    - string
  parallelism: int          # Max concurrent polecats (default: unlimited)
  rigRef: string            # Target rig for spawned polecats
  autoSpawn: bool           # Auto-create polecats for beads (default: false)
```

### Status

```yaml
status:
  phase: string             # Pending | Active | Completed | Failed
  total: int                # Total beads tracked
  completed: int            # Beads completed
  failed: int               # Beads failed
  inProgress: int           # Beads in progress
  pending: int              # Beads not started
  polecats:                 # Spawned polecat references
    - name: string
      beadID: string
      phase: string
  startTime: timestamp
  completionTime: timestamp
```

---

## Witness

Health monitoring for polecats in a rig.

### Spec

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Witness
metadata:
  name: string
  namespace: string
spec:
  rigRef: string            # Rig to monitor
  checkInterval: duration   # How often to check (default: 30s)
  stuckThreshold: duration  # When to consider stuck (default: 10m)
  actions:
    onStuck: string         # notify | restart | terminate (default: notify)
    onFailed: string        # notify | restart | terminate (default: notify)
  notifications:
    slack:
      webhookURL: string
      channel: string
    email:
      to: string
```

### Status

```yaml
status:
  phase: string             # Active | Degraded | Healthy
  lastCheck: timestamp
  polecatsMonitored: int
  polecatsHealthy: int
  polecatsStuck: int
  polecatsFailed: int
  alerts:
    - polecat: string
      type: string          # Stuck | Failed | Recovered
      since: timestamp
      message: string
```

---

## Refinery

Merge queue processor for polecat branches.

### Spec

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Refinery
metadata:
  name: string
  namespace: string
spec:
  rigRef: string            # Rig to process merges for
  targetBranch: string      # Branch to merge into (default: main)
  strategy: string          # merge | rebase | squash (default: merge)
  autoMerge: bool           # Auto-merge passing PRs (default: false)
  requiredChecks:           # CI checks that must pass
    - string
  conflictResolution: string # manual | theirs | ours (default: manual)
```

### Status

```yaml
status:
  phase: string             # Idle | Processing | Blocked
  queue:                    # Branches waiting to merge
    - branch: string
      polecat: string
      beadID: string
      status: string        # Pending | Testing | Ready | Conflict
  lastMerge:
    branch: string
    timestamp: timestamp
    commit: string
  conflicts:
    - branch: string
      files:
        - string
```

---

## Rig

Project workspace (cluster-scoped).

### Spec

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: string              # Rig name (e.g., athena, daedalus)
spec:
  gitURL: string            # Repository URL
  beadsPrefix: string       # Prefix for beads (e.g., "at" for athena)
  localPath: string         # Path on host for local execution
  defaultBranch: string     # Default branch (default: main)
  namePools:                # Names for spawned polecats
    - string                # e.g., "mad-max", "minerals"
```

### Status

```yaml
status:
  phase: string             # Ready | NotReady
  polecatCount: int         # Active polecats
  lastSync: timestamp       # Last beads sync
  conditions:
    - type: string
      status: string
      reason: string
      message: string
```

---

## BeadStore

Issue tracking backend configuration.

### Spec

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: BeadStore
metadata:
  name: string
  namespace: string
spec:
  rigRef: string            # Associated rig
  syncInterval: duration    # How often to sync (default: 5m)
  source:
    type: string            # jsonl | sqlite | daemon
    path: string            # Path to beads data
  filters:
    status:                 # Only sync these statuses
      - string
    labels:                 # Only sync with these labels
      - string
```

### Status

```yaml
status:
  phase: string             # Syncing | Synced | Error
  lastSync: timestamp
  issueCount: int
  errors:
    - message: string
      timestamp: timestamp
```
