# Gastown Operator — Road to A+ and Beyond

**Date:** 2026-02-07
**Scope:** A+ remediation across 6 dimensions + K8s-native AI agent orchestration architecture

---

## Part 1: Road to A+ — Concrete Remediation Plan

### 1.1 Performance: B+ → A+ (3 changes)

**P1. Fix memory leak in BackoffCalculator** — HIGH PRIORITY

The `retries` map in `pkg/errors/backoff.go:46` grows unbounded. The `Cleanup()` method exists (line 107) but is never called from any controller.

**Fix:** Call `Cleanup()` from WitnessReconciler after each reconcile cycle. The Witness already lists all Polecats, making it the natural owner of this cleanup.

```go
// witness_controller.go — after calculateSummary (line ~126)

// Prune backoff entries for resources that no longer exist
if r.Backoff != nil {
    activeKeys := make(map[string]bool, len(polecatList.Items)+1)
    activeKeys[backoffKey] = true // Keep current witness entry
    r.Backoff.Cleanup(activeKeys)
}
```

**P2. Fix N+1 query in ConvoyReconciler**

`convoy_controller.go:102` calls `r.List(ctx, &polecatList)` without any filter — fetches ALL Polecats cluster-wide.

**Fix:** Add a field index in `SetupWithManager` and filter by rig.

```go
// convoy_controller.go SetupWithManager — add field index
if err := mgr.GetFieldIndexer().IndexField(
    context.Background(),
    &gastownv1alpha1.Polecat{}, "spec.rig",
    func(rawObj client.Object) []string {
        polecat, ok := rawObj.(*gastownv1alpha1.Polecat)
        if !ok { return nil }
        return []string{polecat.Spec.Rig}
    },
); err != nil {
    return err
}

// convoy_controller.go Reconcile line 102 — use filtered list
r.List(ctx, &polecatList, client.MatchingFields{"spec.rig": convoy.Spec.RigRef})
```

Note: The `spec.rig` index is already registered in `rig_controller.go:364`. If both controllers run in the same manager, the index only needs to be registered once. Verify this during implementation — if `SetupWithManager` is called after the Rig controller registers the index, this will work automatically. Otherwise, move index registration to a shared setup function.

**P3. Pre-allocate slices when size is known**

```go
// convoy_controller.go:125 — before the categorization loop
completed := make([]string, 0, len(convoy.Spec.TrackedBeads))
pending := make([]string, 0, len(convoy.Spec.TrackedBeads))

// convoy_controller.go:117 — map with capacity hint
beadStatus := make(map[string]gastownv1alpha1.PolecatPhase, len(polecatList.Items))
```

---

### 1.2 Error Handling: A → A+ (2 changes)

**E1. Wrap status update errors with context**

Multiple controllers return bare `r.Status().Update()` errors. Wrap them for debuggability.

Files to fix:
- `witness_controller.go:116` — `return ctrl.Result{...}, r.Status().Update(ctx, witness)`
- `witness_controller.go:180` — `return ctrl.Result{}, err`
- `refinery_controller.go:110` — `return ctrl.Result{...}, r.Status().Update(ctx, refinery)`
- `refinery_controller.go:139` — `return ctrl.Result{}, err`
- `refinery_controller.go:186` — `return ctrl.Result{}, err`
- `refinery_controller.go:359` — `return err`

**Pattern:**
```go
// Before
return ctrl.Result{RequeueAfter: healthCheckInterval}, r.Status().Update(ctx, witness)

// After
if err := r.Status().Update(ctx, witness); err != nil {
    return ctrl.Result{}, gterrors.Wrap(err, "failed to update witness status")
}
return ctrl.Result{RequeueAfter: healthCheckInterval}, nil
```

**E2. Wrap field indexer setup errors**

`rig_controller.go:368,379` return bare errors from `IndexField`.

```go
// Before
return err

// After
return gterrors.Wrap(err, "failed to index polecat spec.rig field")
```

---

### 1.3 API Design: A- → A+ (3 changes)

**A1. Fix godoc field comments to use PascalCase names**

All struct field comments in these files use lowercase JSON tag names instead of the exported Go identifier. Go convention: "RigRef references..." not "rigRef references...".

Files and fields to fix:

`witness_types.go`:
- Line 26: `rigRef` → `RigRef`
- Line 30: `healthCheckInterval` → `HealthCheckInterval`
- Line 35: `stuckThreshold` → `StuckThreshold`
- Line 40: `escalationTarget` → `EscalationTarget`
- Line 48: `phase` → `Phase`
- Line 53: `lastCheckTime` → `LastCheckTime`
- Line 57: `polecatsSummary` → `PolecatsSummary`
- Line 61: `conditions` → `Conditions`
- Line 70: `total` → `Total`
- Line 73: `running` → `Running`
- Line 76: `succeeded` → `Succeeded`
- Line 79: `failed` → `Failed`
- Line 82: `stuck` → `Stuck`

`refinery_types.go`:
- Line 27: `rigRef` → `RigRef`
- Line 31: `targetBranch` → `TargetBranch`
- Line 36: `testCommand` → `TestCommand`
- Line 41: `parallelism` → `Parallelism`
- Line 48: `gitSecretRef` → `GitSecretRef`
- Line 55: `name` → `Name`
- Line 62: `phase` → `Phase`
- Line 67: `queueLength` → `QueueLength`
- Line 72: `currentMerge` → `CurrentMerge`
- Line 75: `lastMergeTime` → `LastMergeTime`
- Line 79: `mergesSummary` → `MergesSummary`
- Line 83: `conditions` → `Conditions`
- Line 92: `total` → `Total`
- Line 95: `succeeded` → `Succeeded`
- Line 98: `failed` → `Failed`
- Line 101: `pending` → `Pending`

`beadstore_types.go`:
- Line 26: `rigRef` → `RigRef`
- Line 30: `prefix` → `Prefix`
- Line 35: `gitSecretRef` → `GitSecretRef`
- Line 39: `syncInterval` → `SyncInterval`
- Line 47: `phase` → `Phase`
- Line 51: `lastSyncTime` → `LastSyncTime`
- Line 55: `issueCount` → `IssueCount`
- Line 59: `conditions` → `Conditions`

**A2. Extract anonymous interface to named type**

`witness_controller.go:60-62` has an inline anonymous interface.

```go
// Before (witness_controller.go:56-67)
type WitnessReconciler struct {
    client.Client
    Scheme   *runtime.Scheme
    Recorder record.EventRecorder
    GTClient interface {
        MailSend(ctx context.Context, address, subject, message string) error
    }
    Backoff *gterrors.BackoffCalculator
}

// After — add named interface above the struct
// GTMailer sends alert messages for escalation.
type GTMailer interface {
    MailSend(ctx context.Context, address, subject, message string) error
}

type WitnessReconciler struct {
    client.Client
    Scheme   *runtime.Scheme
    Recorder record.EventRecorder
    GTClient GTMailer
    Backoff  *gterrors.BackoffCalculator
}
```

Update `cmd/main.go` and test files that construct `WitnessReconciler` — they already pass compatible types, so no functional change needed.

**A3. Use safe type assertions in field indexers**

`rig_controller.go:365,373` — use comma-ok pattern.

```go
// Before
polecat := rawObj.(*gastownv1alpha1.Polecat)

// After
polecat, ok := rawObj.(*gastownv1alpha1.Polecat)
if !ok { return nil }
```

---

### 1.4 Testing: B+ → A+ (5 changes)

**T1. Add Refinery controller tests** — HIGHEST PRIORITY (zero coverage today)

Create `internal/controller/refinery_controller_unit_test.go`:

```
Test cases:
1. Idle refinery with no merge-ready polecats → Phase=Idle
2. Polecat with Available=True → added to merge queue
3. Polecat with old Ready/PodSucceeded → added to merge queue (backward compat)
4. Merge succeeds → MergesSummary.Succeeded incremented, event emitted
5. Merge fails (rebase conflict) → MergesSummary.Failed incremented, conflict metric recorded
6. Git credentials extracted correctly from Secret
7. Missing git credentials → error returned
8. Rig not found → error with context
9. Polecat has no branch in status → error
10. Queue length metric updated correctly
```

Use the same fake client pattern as other unit tests. Inject a mock `GitClientFactory`.

**T2. Add explicit timeouts to Eventually() calls**

`polecat_controller_test.go:165` and similar — add timeout parameter.

```go
// Before
Eventually(func() bool { ... }).Should(BeTrue())

// After
Eventually(func() bool { ... }).WithTimeout(10 * time.Second).WithPolling(100 * time.Millisecond).Should(BeTrue())
```

**T3. Replace time.Sleep with polling**

`metrics_reconcile_test.go:46` — `time.Sleep(10*time.Millisecond)` is fragile.

```go
// Before
time.Sleep(10 * time.Millisecond)

// After
Eventually(func() float64 {
    return testutil.ToFloat64(metrics.ReconcileTotal.WithLabelValues("test", "success"))
}).Should(BeNumerically(">", 0))
```

**T4. Add E2E failure scenarios**

`test/e2e/e2e_test.go` — add negative test cases:

```
1. Create Polecat with missing Kubernetes spec → Phase=Stuck
2. Create Polecat with invalid git URL → Pod fails
3. Delete Rig → verify Witness and Refinery cleaned up (finalizer)
4. Create Convoy with non-existent beads → stays InProgress
5. Witness detects stuck Polecat → event emitted
```

**T5. Mock time.Now() for threshold testing**

`witness_controller_unit_test.go:200` uses hardcoded `time.Now().Add(-20*time.Minute)`.

Introduce a `Clock` interface:

```go
// pkg/clock/clock.go
type Clock interface {
    Now() time.Time
}

type RealClock struct{}
func (RealClock) Now() time.Time { return time.Now() }
```

Inject into `WitnessReconciler`. In tests, use a fixed clock.

---

### 1.5 Remaining Dimensions (already A/A+)

**Concurrency (A+):** No changes needed. Zero goroutines, proper mutex patterns, tuned MaxConcurrentReconciles.

**Project Structure (A+):** No changes needed. Clean layout, zero circular deps, proper go.mod.

---

### 1.6 Implementation Order

| Phase | Items | Estimated Risk |
|-------|-------|---------------|
| 1 | P1 (backoff cleanup), A2 (GTMailer interface) | Low — isolated changes |
| 2 | P2 (convoy index), E1 (wrap errors), A1 (godoc comments) | Low — mechanical |
| 3 | T1 (refinery tests), T2 (timeouts), T3 (sleep→poll) | None — additive |
| 4 | A3 (safe assertions), E2 (wrap indexer errors), P3 (pre-alloc) | None — polish |
| 5 | T4 (E2E failures), T5 (clock injection) | Low — test-only |

After each phase: `make validate-all && make test`

---

## Part 2: K8s-Native AI Agent Orchestration

### 2.1 The Insight

Gastown already is an AI agent orchestration platform. It just doesn't know it yet.

Look at what's already built through the lens of agent orchestration:

| Gastown CRD | Agent Orchestration Concept | Status |
|-------------|---------------------------|--------|
| **Rig** | Team / Project workspace | Done |
| **Polecat** | Worker Agent (with pod lifecycle) | Done |
| **Witness** | Health Monitor / Circuit Breaker | Done |
| **Refinery** | Integration Pipeline (merge queue) | Done |
| **BeadStore** | Task Registry (issue tracking) | Done |
| **Convoy** | Batch Orchestrator | Done |

Every framework — LangGraph, CrewAI, AutoGen, Mastra — implements these same concepts as Python objects in a single process. Gastown implements them as K8s resources with reconciliation loops. This is the difference between a toy and an operating system.

**Why K8s-native matters for agents:**

1. **Fault tolerance is free.** A Python agent crashes? Process dies, work lost. A Polecat pod crashes? K8s restarts it, Witness detects it, Refinery preserves the branch.

2. **Scaling is free.** Need 50 agents? Create 50 Polecats. K8s cluster autoscaler provisions nodes. No code changes.

3. **Security is free.** RBAC limits who can create agents. Network policies restrict what agents can reach. Namespace isolation separates teams. Secrets are injected, never exposed.

4. **Observability is free.** Every reconciliation emits Prometheus metrics. Every state transition fires a K8s event. `kubectl describe` gives you the full agent lifecycle.

5. **Multi-tenancy is free.** Different teams get different namespaces. Different Rigs get different resource budgets. Platform teams control the operator, app teams create Rigs.

6. **GitOps is free.** Agent teams are declared in YAML. ArgoCD/Flux can deploy and manage them. Version control for your agent topology.

---

### 2.2 What's Missing — The Gap Analysis

Map the current architecture against a complete agent orchestration framework:

| Capability | Current State | Gap |
|------------|--------------|-----|
| Agent execution | Polecat (pod lifecycle) | Done |
| Health monitoring | Witness (stuck detection, escalation) | Done |
| Code integration | Refinery (rebase-merge queue) | Done |
| Task tracking | BeadStore (git-native issues) | Done |
| Batch operations | Convoy (progress tracking) | Done |
| **Task decomposition** | Manual (user creates beads) | **Need Foreman** |
| **Agent communication** | None (agents work in isolation) | **Need Vault** |
| **Human-in-the-loop** | Witness escalation only | **Need Gate** |
| **Quality evaluation** | Refinery test command only | **Need Evaluator** |
| **Cost management** | None | **Need Budget** |
| **Tool/capability registry** | Hardcoded Claude CLI | **Need ToolKit** |
| **Agent specialization** | All Polecats are identical | **Need Roles via ToolKit** |
| **Dynamic scheduling** | Static MaxPolecats | **Need Scheduler** |

---

### 2.3 New CRDs — The v1alpha2 Architecture

Six new CRDs, each following the same proven patterns (Spec/Status, Conditions, Finalizers, Reconciliation Loops):

#### 2.3.1 Foreman — Task Decomposition Controller

**Concept:** A Foreman takes a high-level goal and decomposes it into executable Beads. It's an LLM-powered planner that runs as a controller, not a pod.

```yaml
apiVersion: gastown.gastown.io/v1alpha2
kind: Foreman
metadata:
  name: myproject-foreman
  namespace: gastown-system
spec:
  # Which Rig does this Foreman plan for?
  rigRef: myproject

  # The high-level goal to decompose
  goal: |
    Implement user authentication with OAuth2 support.
    Must support Google, GitHub, and email/password providers.
    Include tests and documentation.

  # Planning strategy
  strategy: hierarchical  # hierarchical | flat | iterative

  # Maximum depth for hierarchical decomposition
  maxDepth: 3

  # Model to use for planning (separate from execution model)
  plannerConfig:
    provider: anthropic
    model: claude-sonnet-4-5-20250929
    apiKeySecretRef:
      name: planner-api-key
      key: ANTHROPIC_API_KEY

  # Dependency handling
  dependencies:
    mode: dag  # dag | sequential | parallel

  # Auto-dispatch to Polecats when beads are created
  autoDispatch: true

  # Template for created Polecats
  polecatTemplate:
    spec:
      executionMode: kubernetes
      agent: claude-code
      kubernetes:
        gitSecretRef:
          name: git-creds
        claudeCredsSecretRef:
          name: claude-creds

status:
  phase: Planning  # Planning | Dispatching | Monitoring | Complete | Failed
  totalBeads: 8
  dispatchedBeads: 5
  completedBeads: 3

  # The decomposition plan (DAG)
  plan:
    - beadID: "gt-101"
      title: "Set up OAuth2 provider abstraction"
      dependencies: []
      status: Complete
    - beadID: "gt-102"
      title: "Implement Google OAuth2 provider"
      dependencies: ["gt-101"]
      status: Working
    - beadID: "gt-103"
      title: "Implement GitHub OAuth2 provider"
      dependencies: ["gt-101"]
      status: Working
    # ...

  conditions:
    - type: Ready
      status: "True"
      reason: PlanGenerated
```

**Controller behavior:**
1. On create: Use planner LLM to decompose `goal` into Beads
2. Create Beads in BeadStore with dependency metadata
3. Respect dependency DAG — only dispatch Polecats when predecessors are done
4. Watch Polecat status → dispatch next round when dependencies resolve
5. Handle failures: re-plan if a Polecat fails (adjust remaining work)
6. Auto-create Convoy to track the batch

**Why this is better than LangGraph's planner:**
- The plan is a K8s resource. You can `kubectl get foreman` to see it.
- Re-planning on failure uses the same reconciliation loop, not exception handling.
- The DAG is observable via conditions and events.
- Multiple Foremen can run in the same cluster, each for different Rigs.

---

#### 2.3.2 Vault — Shared Agent Memory

**Concept:** A Vault provides persistent, shared state between Polecats. It's the "blackboard" pattern implemented as a K8s resource with optimistic concurrency.

```yaml
apiVersion: gastown.gastown.io/v1alpha2
kind: Vault
metadata:
  name: myproject-vault
  namespace: gastown-system
spec:
  rigRef: myproject

  # Storage backend
  backend: configmap  # configmap | pvc | external

  # Retention policy
  retention:
    maxEntries: 1000
    ttl: 72h

  # Access control
  readOnly: false
  allowedPolecats: []  # Empty = all Polecats in this Rig

status:
  phase: Ready
  entryCount: 42
  lastWrite: "2026-02-07T10:30:00Z"

  # The shared knowledge entries
  entries:
    - key: "architecture/auth-flow"
      summary: "OAuth2 flow uses PKCE with authorization code grant"
      author: "polecat-obsidian"
      timestamp: "2026-02-07T10:15:00Z"
    - key: "decisions/database"
      summary: "Using PostgreSQL with pgx driver, connection pooling via pgxpool"
      author: "polecat-quartz"
      timestamp: "2026-02-07T10:20:00Z"
    - key: "blockers/api-rate-limit"
      summary: "GitHub API rate limited, need to implement exponential backoff"
      author: "polecat-jasper"
      timestamp: "2026-02-07T10:30:00Z"
```

**Integration with Polecats:**

The Polecat pod builder injects Vault access as environment variables and a sidecar:

```go
// In the agent startup script, the Vault sidecar syncs entries
// to a local JSON file that the agent can read/write
{
    Name: "vault-sidecar",
    Image: "ghcr.io/boshu2/vault-sidecar:latest",
    Env: []corev1.EnvVar{
        {Name: "VAULT_NAME", Value: vaultName},
        {Name: "VAULT_NAMESPACE", Value: namespace},
    },
    VolumeMounts: []corev1.VolumeMount{
        {Name: "vault-data", MountPath: "/vault"},
    },
}
```

The agent reads `/vault/entries.json` and writes new entries by appending to `/vault/outbox/`. The sidecar syncs to the Vault CR via the K8s API.

**Agent prompt injection:**

```
You have access to a shared knowledge vault at /vault/entries.json.
Before starting work, check if other agents have recorded relevant decisions.
After completing work, record key decisions and architectural patterns.
Write entries to /vault/outbox/<key>.json with format: {"summary": "...", "details": "..."}
```

**Why this is better than in-memory shared state:**
- Survives pod restarts.
- Entries are durable K8s resources with audit trails.
- RBAC controls who can read/write.
- Can be backed by a PVC for large knowledge bases.

---

#### 2.3.3 Gate — Human-in-the-Loop Approval

**Concept:** A Gate blocks Polecat progression at defined checkpoints until a human (or automated system) approves. It's the equivalent of a manual approval step in a CI/CD pipeline, but for agent work.

```yaml
apiVersion: gastown.gastown.io/v1alpha2
kind: Gate
metadata:
  name: myproject-review-gate
  namespace: gastown-system
spec:
  rigRef: myproject

  # When does this gate activate?
  trigger:
    # Gate before merge (after Polecat completes, before Refinery processes)
    phase: pre-merge
    # Could also be: pre-dispatch, mid-work, post-merge

  # What requires approval?
  scope:
    # All polecats, or specific patterns
    beadPattern: "*"  # Glob pattern on bead IDs
    # Only gate on changes exceeding thresholds
    thresholds:
      linesChanged: 500
      filesChanged: 10
      newDependencies: true

  # Approval mechanism
  approval:
    method: github-pr-review  # github-pr-review | slack | manual | auto

    # For github-pr-review:
    github:
      requiredApprovals: 1
      requiredReviewers: ["@platform-team"]

    # For slack:
    slack:
      channel: "#agent-review"
      webhookSecretRef:
        name: slack-webhook
        key: url

    # Auto-approve after timeout (safety valve)
    autoApproveAfter: 24h

  # What happens on rejection?
  onReject:
    action: revise  # revise | terminate | escalate
    # For revise: create new Polecat with feedback
    reviseFeedback: true

status:
  phase: Active
  pendingApprovals: 2
  approved: 5
  rejected: 1

  pending:
    - polecatRef: polecat-obsidian
      beadID: "gt-103"
      prURL: "https://github.com/org/repo/pull/42"
      requestedAt: "2026-02-07T10:30:00Z"
      changesummary: "+245 -12 across 4 files"
```

**Controller behavior:**
1. Watch Polecats for phase transitions matching the trigger
2. When triggered: create a GateRequest (embedded status), pause Refinery processing
3. Check approval mechanism periodically:
   - GitHub: poll PR review status via `gh api`
   - Slack: check for reaction/thread reply
   - Manual: wait for `kubectl gt approve <gate> <polecat>`
4. On approval: update GateRequest status, unblock Refinery
5. On rejection with `revise`: create new Polecat with rejection feedback injected into the task description
6. On timeout: auto-approve (with annotation noting it was auto-approved)

---

#### 2.3.4 Evaluator — Automated Quality Gate

**Concept:** An Evaluator runs automated quality checks on Polecat output before the Refinery merges it. It's a quality gate that can run any combination of static analysis, test suites, and even LLM-based code review.

```yaml
apiVersion: gastown.gastown.io/v1alpha2
kind: Evaluator
metadata:
  name: myproject-evaluator
  namespace: gastown-system
spec:
  rigRef: myproject

  # Quality checks to run
  checks:
    - name: lint
      type: command
      command: "make validate"
      required: true

    - name: tests
      type: command
      command: "make test"
      required: true

    - name: coverage
      type: threshold
      command: "go tool cover -func=cover.out | tail -1 | awk '{print $3}'"
      threshold: "80%"
      required: false  # Advisory, not blocking

    - name: code-review
      type: llm-review
      reviewerConfig:
        provider: anthropic
        model: claude-sonnet-4-5-20250929
        apiKeySecretRef:
          name: reviewer-api-key
          key: ANTHROPIC_API_KEY
      # Criteria for the LLM reviewer
      criteria:
        - "Security: No hardcoded secrets, proper input validation"
        - "Architecture: Follows existing patterns in the codebase"
        - "Testing: New code has corresponding tests"
        - "Naming: Variables and functions follow Go conventions"
      required: true
      # Minimum score (1-10) to pass
      minScore: 7

  # What to do with results
  reporting:
    addPRComment: true
    emitEvents: true
    updateConditions: true

status:
  phase: Active
  evaluations:
    - polecatRef: polecat-obsidian
      beadID: "gt-103"
      results:
        - check: lint
          passed: true
          duration: "3s"
        - check: tests
          passed: true
          duration: "12s"
        - check: coverage
          passed: false
          value: "72%"
          threshold: "80%"
        - check: code-review
          passed: true
          score: 8
          feedback: "Clean implementation. Consider adding error wrapping in handler."
      overallPassed: true  # All required checks passed
```

**Controller behavior:**
1. Watch for Polecats reaching `Available=True` (work complete)
2. Clone the branch, run each check as a K8s Job (or in-process for speed)
3. For `llm-review`: call the configured LLM with the diff and criteria
4. Aggregate results, set conditions on the Polecat
5. Only allow Refinery to process if Evaluator passes
6. Post results as PR comments (via `gh pr comment`)

**Interaction with Refinery:**
The Refinery's `findMergeReadyPolecats` gains a new check:

```go
// Only merge polecats that passed evaluation
evalCond := meta.FindStatusCondition(polecat.Status.Conditions, "Evaluated")
if evalCond == nil || evalCond.Status != metav1.ConditionTrue {
    continue // Skip — not yet evaluated or failed evaluation
}
```

---

#### 2.3.5 Budget — Cost and Token Management

**Concept:** A Budget sets spending limits on LLM usage per Rig, Convoy, or individual Polecat. It prevents runaway costs from stuck or looping agents.

```yaml
apiVersion: gastown.gastown.io/v1alpha2
kind: Budget
metadata:
  name: myproject-budget
  namespace: gastown-system
spec:
  rigRef: myproject

  limits:
    # Per-Polecat limits
    perPolecat:
      maxTokens: 500000        # Input + output tokens
      maxDuration: 2h          # Wall clock time
      maxAPICallsCostUSD: 10.00  # Dollar limit

    # Per-Convoy limits
    perConvoy:
      maxTokens: 5000000
      maxAPICallsCostUSD: 100.00

    # Global Rig limits (monthly)
    monthly:
      maxTokens: 50000000
      maxAPICallsCostUSD: 1000.00

  # What happens when budget is exceeded
  onExceeded:
    action: pause  # pause | terminate | alert-only
    alertTarget: "mayor"

status:
  phase: Active

  currentMonth:
    tokensUsed: 12500000
    estimatedCostUSD: 250.00
    polecatsThrottled: 0

  perPolecat:
    - name: polecat-obsidian
      tokensUsed: 125000
      estimatedCostUSD: 2.50
      withinBudget: true
```

**Integration with telemetry sidecar:**

The existing telemetry sidecar (`pkg/pod/builder.go:486`) already collects metrics. Extend it to track token usage:

```sh
# In telemetry sidecar — monitor Claude CLI output for token counts
# Claude CLI logs usage to stderr: "Input tokens: 1234, Output tokens: 567"
# Parse and expose as Prometheus metrics:
# polecat_tokens_input_total{polecat="...", rig="...", bead="..."} 1234
# polecat_tokens_output_total{...} 567
```

**Controller behavior:**
1. Watch Prometheus metrics for token usage per Polecat/Rig
2. When approaching limit: set Warning condition
3. When exceeded: execute `onExceeded` action (patch Polecat `desiredState: Idle`)
4. Aggregate monthly usage in Budget status
5. Reset counters on month boundary

---

#### 2.3.6 ToolKit — Agent Capability Registry

**Concept:** A ToolKit declares what tools and capabilities are available to Polecats. Different tasks need different tools — a database migration agent needs SQL tools, a frontend agent needs browser tools, a security agent needs scanning tools.

```yaml
apiVersion: gastown.gastown.io/v1alpha2
kind: ToolKit
metadata:
  name: backend-toolkit
  namespace: gastown-system
spec:
  # MCP servers to inject into the agent pod
  mcpServers:
    - name: postgres
      image: "ghcr.io/modelcontextprotocol/server-postgres:latest"
      transport: stdio
      env:
        - name: DATABASE_URL
          valueFrom:
            secretKeyRef:
              name: db-credentials
              key: url

    - name: github
      image: "ghcr.io/modelcontextprotocol/server-github:latest"
      transport: stdio
      env:
        - name: GITHUB_TOKEN
          valueFrom:
            secretKeyRef:
              name: github-token
              key: token

  # CLI tools to install in the agent image
  tools:
    - name: sqlc
      installCommand: "go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest"
    - name: migrate
      installCommand: "go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"

  # CLAUDE.md additions injected into the repo
  agentInstructions: |
    You have access to a PostgreSQL database via the postgres MCP server.
    Use sqlc for type-safe SQL. Use migrate for schema changes.
    Always create reversible migrations.

  # Network policies to apply to Polecat pods using this toolkit
  networkAccess:
    egress:
      - to: ["postgres.database.svc.cluster.local"]
        ports: [5432]
      - to: ["api.github.com"]
        ports: [443]
```

**Integration with Polecat:**

Reference a ToolKit in the Polecat spec:

```yaml
spec:
  toolKitRef: backend-toolkit  # New field in PolecatSpec
```

The Polecat pod builder reads the ToolKit and:
1. Adds MCP server containers as sidecars
2. Generates `claude_desktop_config.json` with MCP server connections
3. Installs CLI tools in an init container
4. Applies NetworkPolicy for restricted egress
5. Injects `agentInstructions` as a ConfigMap mounted at `/workspace/repo/.claude/toolkit.md`

---

### 2.4 The Complete Architecture

```
                         ┌─────────────────────────────────────┐
                         │  kubectl gt / GitOps / API          │
                         └───────────────┬─────────────────────┘
                                         │
                                         ▼
                    ┌─────────────────────────────────────────────┐
                    │                  Rig (v1)                    │
                    │  cluster-scoped workspace definition         │
                    │  auto-provisions: Witness, Refinery,        │
                    │    Foreman, Evaluator, Budget, Vault        │
                    └──────────────────┬──────────────────────────┘
                                       │ creates
            ┌──────────────────────────┼──────────────────────────┐
            │                          │                          │
            ▼                          ▼                          ▼
    ┌───────────────┐      ┌───────────────────┐      ┌──────────────────┐
    │ Foreman (v2)  │      │ BeadStore (v1)    │      │ Budget (v2)      │
    │ decomposes    │─────▶│ stores tasks      │      │ tracks costs     │
    │ goals → beads │      │ (git-native)      │      │ throttles agents │
    └───────┬───────┘      └───────────────────┘      └────────┬─────────┘
            │ dispatches                                        │ enforces
            ▼                                                   │
    ┌───────────────────────────────────────────────────────────┐│
    │                    Convoy (v1)                             ││
    │   tracks batch progress across beads                      ││
    └──────────────────────┬────────────────────────────────────┘│
                           │ creates                             │
         ┌─────────────────┼─────────────────────┐               │
         │                 │                     │               │
         ▼                 ▼                     ▼               │
    ┌──────────┐    ┌──────────┐          ┌──────────┐          │
    │ Polecat  │    │ Polecat  │   ...    │ Polecat  │◄─────────┘
    │ worker A │    │ worker B │          │ worker N │
    │ ┌──────┐ │    │ ┌──────┐ │          │ ┌──────┐ │
    │ │ Pod  │ │    │ │ Pod  │ │          │ │ Pod  │ │
    │ │git   │ │    │ │git   │ │          │ │git   │ │
    │ │agent │ │    │ │agent │ │          │ │agent │ │
    │ │telem │ │    │ │telem │ │          │ │telem │ │
    │ │vault │ │    │ │vault │ │          │ │vault │ │
    │ │mcp*  │ │    │ │mcp*  │ │          │ │mcp*  │ │
    │ └──────┘ │    │ └──────┘ │          │ └──────┘ │
    └────┬─────┘    └────┬─────┘          └────┬─────┘
         │               │                     │
         └───────────┬───┘─────────────────────┘
                     │ reads/writes
              ┌──────▼──────┐
              │ Vault (v2)  │
              │ shared      │
              │ knowledge   │
              └─────────────┘

    ┌──────────┐    ┌──────────────┐    ┌──────────┐
    │ Witness  │    │ Evaluator    │    │  Gate    │
    │ monitors │    │ quality      │    │ human    │
    │ health   │    │ checks       │    │ review   │
    └────┬─────┘    └──────┬───────┘    └────┬─────┘
         │                 │                  │
         └────────┬────────┘──────────────────┘
                  │ gates
           ┌──────▼──────┐
           │ Refinery    │
           │ merge queue │
           │ (rebase +   │
           │  push)      │
           └─────────────┘

    ┌──────────┐
    │ ToolKit  │ referenced by Polecats
    │ MCP svrs │ injects sidecars + config
    │ CLI tools│
    │ net policy│
    └──────────┘
```

---

### 2.5 The Rig as Auto-Provisioner

The key pattern: **creating a Rig gives you a complete AI engineering team.**

Today, `rig_controller.go:182` (`ensureChildren`) creates Witness + Refinery. Extend this:

```go
func (r *RigReconciler) ensureChildren(ctx context.Context, rig *Rig) error {
    // v1 children (existing)
    r.ensureWitness(ctx, rig)     // Health monitoring
    r.ensureRefinery(ctx, rig)    // Merge queue

    // v2 children (new)
    r.ensureForeman(ctx, rig)     // Task planner (if rig.spec.foreman is set)
    r.ensureEvaluator(ctx, rig)   // Quality gate (if rig.spec.evaluator is set)
    r.ensureBudget(ctx, rig)      // Cost limits (if rig.spec.budget is set)
    r.ensureVault(ctx, rig)       // Shared memory (always)
}
```

The user experience becomes:

```bash
# Create a Rig — everything else is automatic
kubectl gt rig create myproject \
  --git-url git@github.com:org/repo.git \
  --prefix my

# Give it a goal — Foreman plans, Polecats execute, Evaluator reviews, Refinery merges
kubectl gt plan myproject "Implement OAuth2 authentication"

# Watch the team work
kubectl gt convoy status cv-auth-oauth2
```

---

### 2.6 Agent Communication Patterns

Three patterns for inter-agent communication, each suited to different scenarios:

**Pattern 1: Vault (Blackboard)**

Async, loosely-coupled. Agents read and write shared knowledge. Good for architectural decisions, discovered constraints, and reusable context.

```
Agent A writes: "Database schema uses UUID primary keys"
Agent B reads: knows to use UUIDs without asking
```

**Pattern 2: Bead Dependencies (DAG)**

Structural coupling via the Foreman's dependency graph. Agent B blocks until Agent A's bead is complete. Good for sequential workflows.

```
gt-101: "Create user model" → DONE
gt-102: "Create auth endpoints" (depends on gt-101) → NOW DISPATCHED
```

**Pattern 3: Gate Feedback (Review Loop)**

Human or LLM reviewer provides feedback that creates a new revision Polecat. Good for quality iteration.

```
Polecat A submits PR → Evaluator scores 5/10 → Gate rejects
→ New Polecat A' created with feedback injected → submits revised PR
→ Evaluator scores 9/10 → Gate approves → Refinery merges
```

---

### 2.7 Multi-Model Agent Topology

The ToolKit and AgentConfig already support multiple LLM providers. The orchestration layer can assign different models to different roles:

| Role | Model | Rationale |
|------|-------|-----------|
| Foreman (planner) | claude-opus-4-6 | Complex reasoning for task decomposition |
| Polecat (worker) | claude-sonnet-4-5 | Best cost/quality for implementation |
| Evaluator (reviewer) | claude-sonnet-4-5 | Code review with grading criteria |
| Gate (auto-review) | claude-haiku-4-5 | Fast, cheap for simple approval checks |

This is configured per-CRD, not globally — each Rig can have its own model topology.

---

### 2.8 Implementation Phases

| Phase | Milestone | CRDs | Est. Effort |
|-------|-----------|------|-------------|
| **0** | A+ remediation (Part 1) | — | 1-2 days |
| **1** | Vault + ToolKit | 2 new CRDs, pod builder changes | 1 week |
| **2** | Foreman + Budget | 2 new CRDs, BeadStore integration | 1-2 weeks |
| **3** | Evaluator + Gate | 2 new CRDs, Refinery integration | 1-2 weeks |
| **4** | CLI + Docs | `kubectl gt plan`, `kubectl gt approve` | 1 week |

**Phase 0** has zero dependencies and should ship immediately.

**Phase 1** is foundational — Vault enables agent communication, ToolKit enables specialization. Both are prerequisites for Phase 2.

**Phase 2** is the core value — autonomous task decomposition and cost management.

**Phase 3** closes the quality loop — agents can iterate on feedback without human intervention for trivial issues, while humans review high-impact changes.

---

### 2.9 Why This Wins

Every AI agent framework today is a library. Libraries run in a single process, on a single machine, with a single set of credentials. They don't survive crashes. They don't scale. They don't have access control.

Gastown is an operating system for AI agents. It treats agents as workloads, just like K8s treats containers as workloads. The same primitives that made containers ubiquitous — declarative state, reconciliation loops, health checks, resource limits, RBAC — make agents manageable at scale.

The six existing CRDs already prove the pattern works. The six new CRDs complete the picture. The result is a platform where you can say:

```bash
kubectl gt rig create myapp --git-url git@github.com:org/myapp.git --prefix ma
kubectl gt plan myapp "Build a REST API for user management with auth"
```

And walk away. The Foreman plans. The Polecats build. The Vault shares knowledge. The Evaluator reviews. The Gate escalates what matters. The Refinery merges what's ready. The Witness watches everything. The Budget keeps costs sane.

That's K8s-native AI agent orchestration.
