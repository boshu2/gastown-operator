# Research: From View Layer to Native K8s Gas Town

**Date:** 2026-01-17
**Topic:** Evolution path from gastown_operator (view layer) to native K8s orchestration
**Status:** COMPLETE

---

## Executive Summary

The gastown_operator currently operates as a **view layer** - CRDs provide Kubernetes-native access to Gas Town, but the `gt` CLI remains the source of truth. This research identifies what's needed to evolve to a **native K8s solution** where Kubernetes IS the source of truth.

**Current State:** Operator shells out to `gt` CLI, syncs status back to CRDs
**Target State:** Operator creates K8s Jobs/Pods directly, manages state in etcd

---

## Current Architecture Analysis

### What gastown_operator Does Today

```
┌─────────────────────────────────────────────────────┐
│              Kubernetes                              │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐             │
│  │   Rig   │  │ Polecat │  │ Convoy  │  ← CRDs    │
│  │   CRD   │  │   CRD   │  │   CRD   │    (Views) │
│  └────┬────┘  └────┬────┘  └────┬────┘             │
│       └───────────┼────────────┘                    │
│                   │ shell exec                      │
│                   ▼                                 │
│            ┌─────────────┐                          │
│            │   gt CLI    │  ← Source of Truth      │
│            └──────┬──────┘                          │
└───────────────────┼─────────────────────────────────┘
                    ▼
             ┌─────────────┐
             │  Filesystem │  (~/gt/, .beads/, tmux)
             └─────────────┘
```

### CRDs Currently Implemented

| CRD | Purpose | Controller Action |
|-----|---------|-------------------|
| **Rig** | Project workspace | Syncs status from `gt rig status` |
| **Polecat** | Worker agent | Calls `gt sling`, `gt polecat reset/nuke` |
| **Convoy** | Batch tracking | Calls `gt convoy create/status` |
| **BeadStore** | Git sync config | Syncs with bd CLI |
| **Witness** | Health monitor | Polls polecat status |
| **Refinery** | Merge processor | Manages merge queue |

### Key Dependencies on gt CLI

From `pkg/gt/client.go`:

```go
// Every operation shells out to gt
func (c *Client) run(ctx context.Context, args ...string) ([]byte, error) {
    cmd := exec.CommandContext(ctx, c.GTPath, args...)
    cmd.Env = append(os.Environ(), "GT_TOWN_ROOT="+c.TownRoot)
    return cmd.Output()
}
```

**Operations requiring gt CLI:**
- `gt sling <beadID> <rig>` - Dispatch work to polecat
- `gt polecat status/list/reset/nuke` - Polecat lifecycle
- `gt convoy create/status/list` - Convoy management
- `gt rig status/list` - Rig status
- `gt hook <beadID> <assignee>` - Work assignment

---

## Gap Analysis: View Layer vs Native K8s

### What "Native K8s" Means

| Aspect | Current (View Layer) | Native K8s Target |
|--------|---------------------|-------------------|
| **Source of Truth** | gt CLI + filesystem | etcd (CRD status) |
| **Agent Execution** | tmux sessions on host | K8s Jobs/Pods |
| **Git Operations** | Host git CLI | Init containers + git-sync |
| **Session State** | tmux session files | Pod lifecycle |
| **Work Dispatch** | `gt sling` → tmux | Controller → Job creation |
| **Persistence** | Host filesystem | PVCs / EmptyDir |

### Components to Replace

| Current | Native K8s Replacement | Complexity |
|---------|------------------------|------------|
| `gt sling` | Job creation with init container | Medium |
| tmux sessions | Pod containers | Low |
| `~/gt/polecats/` worktrees | EmptyDir or PVC | Low |
| `gt polecat status` | Pod/Job status | Low |
| `.beads/issues.jsonl` | ConfigMap or CRD per issue | Medium |
| beads git sync | CronJob or sidecar | Medium |
| Claude Code CLI | Container image | **External dependency** |

---

## Target Architecture

### Native K8s Gas Town

```
┌────────────────────────────────────────────────────────────────┐
│                    SINGLE NAMESPACE (e.g., "gastown")          │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │              GASTOWN OPERATOR                             │ │
│  │                                                           │ │
│  │  BeadStore ──────> CronJob (bd sync every 30s)           │ │
│  │      │                                                    │ │
│  │      └──> ConfigMaps (issue data) or PVC (.beads/)       │ │
│  │                                                           │ │
│  │  Polecat CRD ────> K8s Job                               │ │
│  │      │                                                    │ │
│  │      ├──> Init Container: git clone + worktree           │ │
│  │      ├──> Main Container: claude-code / opencode         │ │
│  │      └──> EmptyDir: workspace (or PVC for large repos)   │ │
│  │                                                           │ │
│  │  Convoy CRD ─────> Label aggregation on Polecats         │ │
│  │                                                           │ │
│  └──────────────────────────────────────────────────────────┘ │
│                                                                │
│  ┌──────────────────────────────────────────────────────────┐ │
│  │              SHARED RESOURCES                             │ │
│  │                                                           │ │
│  │  Secret: api-keys ────> ANTHROPIC_API_KEY, LITELLM_KEY   │ │
│  │  Secret: git-creds ───> SSH key for git operations       │ │
│  │  ConfigMap: config ───> Agent settings, rig config       │ │
│  │                                                           │ │
│  └──────────────────────────────────────────────────────────┘ │
└────────────────────────────────────────────────────────────────┘
```

---

## Implementation Requirements

### 1. Polecat Controller Rewrite

**Current:** Calls `gt sling`, polls `gt polecat status`

**Native:** Creates K8s Job directly

```go
func (r *PolecatReconciler) ensureWorking(ctx context.Context, polecat *v1alpha1.Polecat) (ctrl.Result, error) {
    // Instead of: r.GTClient.Sling(ctx, polecat.Spec.BeadID, polecat.Spec.Rig)
    // Create Job directly:

    job := r.constructJob(polecat)
    if err := r.Create(ctx, job); err != nil {
        return ctrl.Result{}, err
    }

    polecat.Status.Phase = PolecatPhaseWorking
    polecat.Status.JobName = job.Name
    return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
}

func (r *PolecatReconciler) constructJob(polecat *v1alpha1.Polecat) *batchv1.Job {
    return &batchv1.Job{
        Spec: batchv1.JobSpec{
            Template: corev1.PodTemplateSpec{
                Spec: corev1.PodSpec{
                    RestartPolicy: corev1.RestartPolicyNever,
                    InitContainers: []corev1.Container{
                        r.gitInitContainer(polecat),
                    },
                    Containers: []corev1.Container{
                        r.agentContainer(polecat),
                    },
                    Volumes: r.workspaceVolumes(polecat),
                },
            },
        },
    }
}
```

### 2. Git Operations via Init Container

**Current:** `gt sling` creates git worktree on host

**Native:** Init container handles git setup

```go
func (r *PolecatReconciler) gitInitContainer(polecat *v1alpha1.Polecat) corev1.Container {
    return corev1.Container{
        Name:  "git-init",
        Image: "alpine/git:latest",
        Command: []string{"/bin/sh", "-c"},
        Args: []string{fmt.Sprintf(`
            mkdir -p ~/.ssh
            cp /git-creds/ssh-privatekey ~/.ssh/id_rsa
            chmod 600 ~/.ssh/id_rsa
            ssh-keyscan github.com >> ~/.ssh/known_hosts

            git clone --depth=1 -b %s %s /workspace/repo
            cd /workspace/repo
            git checkout -b %s
        `, polecat.Spec.GitBranch, polecat.Spec.GitRepository, polecat.Spec.WorkBranch)},
        VolumeMounts: []corev1.VolumeMount{
            {Name: "workspace", MountPath: "/workspace"},
            {Name: "git-creds", MountPath: "/git-creds", ReadOnly: true},
        },
    }
}
```

### 3. Agent Container

**Current:** `gt sling` spawns Claude Code in tmux

**Native:** Main container runs agent CLI

```go
func (r *PolecatReconciler) agentContainer(polecat *v1alpha1.Polecat) corev1.Container {
    image := "ghcr.io/opencode-ai/opencode:latest"
    command := "opencode"

    if polecat.Spec.Agent == "claude-code" {
        // Claude Code would need official container image
        image = "anthropic/claude-code:latest" // hypothetical
        command = "claude"
    }

    return corev1.Container{
        Name:       "agent",
        Image:      image,
        WorkingDir: "/workspace/repo",
        Command:    []string{command},
        Args:       []string{"--dangerously-skip-permissions"},
        Env: []corev1.EnvVar{
            r.apiKeyEnvVar(polecat),
            {Name: "GT_ISSUE", Value: polecat.Spec.BeadID},
            {Name: "GT_POLECAT", Value: polecat.Name},
        },
        VolumeMounts: []corev1.VolumeMount{
            {Name: "workspace", MountPath: "/workspace"},
        },
        Resources: polecat.Spec.Resources,
    }
}
```

### 4. BeadStore Controller

**Current:** Relies on `bd` CLI on host

**Options for Native K8s:**

**Option A: Sidecar with bd binary** (Recommended)
```yaml
# BeadStore creates a Deployment with bd daemon
spec:
  template:
    containers:
      - name: beads-sync
        image: beads:latest  # Contains bd binary
        command: ["bd", "daemon", "--sync-interval=30s"]
        volumeMounts:
          - name: beads-data
            mountPath: /workspace/.beads
```

**Option B: ConfigMap per issue**
```go
// Sync issues.jsonl to individual ConfigMaps
func (r *BeadStoreReconciler) syncIssues(ctx context.Context, store *v1alpha1.BeadStore) error {
    issues := r.parseIssuesJsonl(store)
    for _, issue := range issues {
        cm := &corev1.ConfigMap{
            ObjectMeta: metav1.ObjectMeta{
                Name: "bead-" + issue.ID,
                Labels: map[string]string{
                    "bead-status": issue.Status,
                    "bead-type": issue.Type,
                },
            },
            Data: map[string]string{
                "issue.json": issue.ToJSON(),
            },
        }
        r.CreateOrUpdate(ctx, cm)
    }
}
```

**Option C: Single ConfigMap with full JSONL**
```yaml
# Simpler, but less queryable
apiVersion: v1
kind: ConfigMap
metadata:
  name: beads-data
data:
  issues.jsonl: |
    {"id":"gt-0001","title":"Fix auth","status":"open"}
    {"id":"gt-0002","title":"Add tests","status":"closed"}
```

### 5. Convoy Controller

**Current:** Calls `gt convoy create/status`

**Native:** Pure label aggregation

```go
func (r *ConvoyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    var convoy v1alpha1.Convoy
    r.Get(ctx, req.NamespacedName, &convoy)

    // List polecats with matching labels
    var polecats v1alpha1.PolecatList
    r.List(ctx, &polecats, client.MatchingLabels{"convoy": convoy.Name})

    // Aggregate status
    convoy.Status.Total = len(polecats.Items)
    convoy.Status.Succeeded = 0
    convoy.Status.Failed = 0
    convoy.Status.Running = 0

    for _, p := range polecats.Items {
        switch p.Status.Phase {
        case PolecatPhaseDone:
            convoy.Status.Succeeded++
        case PolecatPhaseStuck:
            convoy.Status.Failed++
        case PolecatPhaseWorking:
            convoy.Status.Running++
        }
    }

    r.Status().Update(ctx, &convoy)
    return ctrl.Result{RequeueAfter: 30 * time.Second}, nil
}
```

---

## External Dependencies

### Claude Code Container Image

**Blocker:** Anthropic does not publish an official Claude Code container image.

**Options:**

1. **Build our own** - Package Claude Code CLI in container
   ```dockerfile
   FROM node:20-alpine
   RUN npm install -g @anthropic-ai/claude-code
   ENTRYPOINT ["claude"]
   ```

2. **Use OpenCode** - Already has container image, supports LiteLLM
   ```yaml
   image: ghcr.io/opencode-ai/opencode:latest
   ```

3. **Hybrid approach** - Local Claude Code via host mount (current), OpenCode for K8s-native

### LLM Provider Integration

**Kagent pattern** (from cyclopes) shows how to handle multiple providers:

```go
type AgentConfig struct {
    Provider        LLMProvider  // litellm, anthropic, openai, ollama
    Model           string
    ModelProvider   *ModelProviderConfig
}

type ModelProviderConfig struct {
    Endpoint        string
    APIKeySecretRef *SecretKeyRef
}
```

---

## Migration Strategy

### Phase 0: Preparation (No Breaking Changes)

1. Add `spec.executionMode` field to Polecat CRD
   - `local` (default) - Current behavior via gt CLI
   - `kubernetes` - New K8s-native execution

2. Add new fields for K8s-native execution:
   ```yaml
   spec:
     executionMode: kubernetes
     kubernetes:
       gitRepository: "git@github.com:org/repo.git"
       gitSecretRef:
         name: git-creds
       agent: opencode
       agentConfig:
         provider: litellm
         model: "devstral-123b"
   ```

### Phase 1: Parallel Implementation

1. Implement Job-based Polecat execution
2. Both modes work simultaneously
3. Default remains `local`

### Phase 2: BeadStore K8s-Native

1. Implement beads-sync sidecar pattern
2. ConfigMap-based issue storage
3. Deprecate `bd` CLI dependency

### Phase 3: Cutover

1. Default to `kubernetes` execution mode
2. Deprecate `local` mode
3. Remove gt CLI dependency

### Phase 4: Cleanup

1. Remove gt CLI wrapper code
2. Remove hostPath volume requirements
3. Simplify operator to pure controller-runtime

---

## Comparison with Kagent

The cyclopes (kagent) fork provides a reference for K8s-native agent orchestration:

| Aspect | gastown_operator (current) | kagent (cyclopes) |
|--------|---------------------------|-------------------|
| Execution | tmux sessions | K8s Deployments |
| State | Host filesystem | Database (GORM) |
| LLM Config | API keys in env | ModelConfig CRD |
| Tools | gt CLI | MCP Servers |
| A2A | Not implemented | TRPC A2A protocol |
| Agent Types | claude-code, opencode | Declarative, BYO |

**Learnings from kagent:**
1. Use Secrets for API keys with hash tracking for pod restarts
2. Support multiple LLM providers via ModelConfig pattern
3. Separate agent framework (ADK) from K8s orchestration
4. A2A protocol enables agent-to-agent communication

---

## Recommended Approach

### "Titans Before Olympus"

Based on the olympus research (`2026-01-15-minimal-gastown-operator.md`):

**Don't try to build everything. Build what works first.**

1. **3 CRDs:** BeadStore, Polecat, Convoy
2. **2 Controllers:** BeadStore (git sync), Polecat (Job creation)
3. **~1500 LOC** total
4. **Single namespace** scope

### Implementation Priority

| Priority | Component | Rationale |
|----------|-----------|-----------|
| P0 | Job-based Polecat | Core execution model |
| P0 | Git init container | Required for Job execution |
| P1 | BeadStore with bd sidecar | Reuses existing bd, minimal rewrite |
| P1 | OpenCode support | Has container image, LiteLLM support |
| P2 | ConfigMap-based beads | Pure K8s, no bd dependency |
| P2 | Claude Code container | Requires packaging effort |
| P3 | A2A protocol | Agent-to-agent communication |
| P3 | Multi-namespace | Federation across rigs |

---

## Success Criteria

### Minimum Viable Native K8s

```bash
# Create polecat via CRD
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: test-worker
spec:
  executionMode: kubernetes
  kubernetes:
    gitRepository: "git@github.com:example/repo.git"
    gitBranch: "main"
    gitSecretRef:
      name: git-creds
    agent: opencode
    agentConfig:
      provider: litellm
      model: "devstral-123b"
      modelProvider:
        endpoint: "https://ai-gateway.example.com/v1"
        apiKeySecretRef:
          name: litellm-key
          key: API_KEY
  beadID: "gt-0042"
EOF

# Watch Job creation
kubectl get jobs -w

# Verify pod runs
kubectl logs job/polecat-test-worker -c agent

# Check polecat status
kubectl get polecat test-worker -o yaml
```

### Definition of Done

- [ ] Polecat CRD creates K8s Job (not tmux session)
- [ ] Init container clones git repo successfully
- [ ] Agent container runs opencode and exits cleanly
- [ ] Polecat status reflects Job completion
- [ ] No dependency on gt CLI for K8s-native execution
- [ ] BeadStore syncs issues from git (via sidecar or CronJob)
- [ ] Convoy aggregates Polecat status via labels

---

## Files to Modify

### New Files

| File | Purpose |
|------|---------|
| `internal/controller/polecat_job.go` | Job construction logic |
| `internal/controller/beadstore_sync.go` | Beads sync sidecar logic |
| `pkg/git/init_container.go` | Git init container templates |
| `pkg/agent/container.go` | Agent container templates |

### Modified Files

| File | Changes |
|------|---------|
| `api/v1alpha1/polecat_types.go` | Add `executionMode`, `kubernetes` spec |
| `internal/controller/polecat_controller.go` | Add K8s-native execution path |
| `internal/controller/beadstore_controller.go` | Add sidecar deployment option |
| `pkg/gt/client.go` | Make optional (only for local mode) |

---

## Conclusion

Evolving from view layer to native K8s requires:

1. **Replacing gt CLI shelling** with direct K8s resource creation
2. **Moving execution** from tmux sessions to K8s Jobs
3. **Handling git operations** via init containers
4. **Choosing agent runtime** (OpenCode recommended for K8s-native)
5. **Syncing beads** via sidecar or ConfigMaps

The migration can be incremental via `executionMode` field, allowing both patterns to coexist during transition.

**Estimated effort:** 2 weeks for MVP (Job-based execution + OpenCode support)

---

## Related Documents

- `docs/architecture.md` - Current operator architecture
- `~/gt/olympus/crew/goku/.agents/research/2026-01-15-minimal-gastown-operator.md` - Minimal operator proposal
- `~/gt/cyclopes/` - kagent fork with K8s-native patterns
- `~/gt/daedalus/` - Gas Town CLI source

---

## Next Steps

1. **Decide:** OpenCode-first or Claude Code container packaging?
2. **Prototype:** Job-based Polecat in separate branch
3. **Validate:** Test with real beads issue
4. **/formulate:** Create implementation plan from this research
