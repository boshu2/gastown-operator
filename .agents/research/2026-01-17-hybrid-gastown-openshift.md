# Research: Hybrid Gas Town - Local Mayor, Remote Polecats

**Date:** 2026-01-17
**Topic:** Great Gas Town experience in OpenShift with Claude Code (OAuth2) first
**Status:** COMPLETE

---

## Executive Summary

**Goal:** Same `gt sling` workflow, but polecats run in your OpenShift cluster instead of your laptop.

**Architecture:**
```
┌─────────────────────┐         ┌──────────────────────────────┐
│    Your Laptop      │         │     OpenShift Cluster        │
│                     │         │                              │
│  ┌───────────────┐  │         │  ┌────────────────────────┐  │
│  │    Mayor      │  │  gt     │  │   Polecat Pods         │  │
│  │  (Claude)     │──┼──sling──┼─>│   - toast-001          │  │
│  │               │  │         │  │   - shadow-002         │  │
│  └───────────────┘  │         │  │   - copper-003         │  │
│                     │         │  └────────────────────────┘  │
│  ┌───────────────┐  │         │                              │
│  │  ~/.claude/   │  │  sync   │  ┌────────────────────────┐  │
│  │  (OAuth2)     │──┼─────────┼─>│  Secret: claude-creds  │  │
│  └───────────────┘  │         │  └────────────────────────┘  │
│                     │         │                              │
│  ┌───────────────┐  │         │  ┌────────────────────────┐  │
│  │    Beads      │<─┼───git───┼─>│  BeadStore (git sync)  │  │
│  │  (issues)     │  │  push   │  └────────────────────────┘  │
│  └───────────────┘  │         │                              │
└─────────────────────┘         └──────────────────────────────┘
```

**Key insight:** Your OAuth2-authenticated `~/.claude/` stays on your laptop. Polecats in the cluster mount a copy (Secret) that you sync when you re-auth.

---

## Why This Matters for Adoption

### Gas Town's Current Position

From [Steve Yegge's announcement](https://steve-yegge.medium.com/welcome-to-gas-town-4f25ee16dd04):

> "You need to be at least level 6... Gas Town is for massive parallelization across a large codebase."

**Current limitations:**
- Runs entirely on laptop (tmux sessions)
- Competes for local resources (CPU, RAM, context)
- $100/hour burn rate with 20-30 agents
- Tmux sessions die when laptop sleeps

### The Enterprise Gap

| What Users Have | What Enterprises Need |
|-----------------|----------------------|
| Anthropic subscription | Same subscription, but shared |
| Laptop execution | Cluster execution (scale) |
| Individual agent | Team visibility |
| Local tmux | Kubernetes observability |
| Filesystem state | Centralized state |

**Your niche:** Bridge this gap. Same workflow, enterprise infrastructure.

### Claude Code First (OAuth2) Strategy

Most Gas Town users have Anthropic subscriptions (OAuth2), not API keys:
- Login once: `claude login`
- Credentials in `~/.claude/`
- Works with all Claude models (Sonnet, Opus, Haiku)
- **Metered by Anthropic** (no separate billing)

OpenCode/LiteLLM is for enterprises with self-hosted models - that's phase 2.

---

## Technical Foundation

### Claude Code CAN Run Headless

From Gas Town source (`internal/config/agents.go`):

```go
AgentClaude: {
    Command: "claude",
    Args:    []string{"--dangerously-skip-permissions"},
    SessionIDEnv: "CLAUDE_SESSION_ID",
    ResumeFlag:   "--resume",
}
```

**Requirements for container execution:**
1. `CLAUDE_CONFIG_DIR` → Mount authenticated credentials
2. `--dangerously-skip-permissions` → Skip interactive dialogs
3. `CLAUDE_SESSION_ID` → Resume sessions across restarts

**What this means:** Claude Code can run in a Pod if you mount the auth.

### gt CLI Is Ready for Extension

The codebase isolates tmux dependency to one layer:

```go
// Current: tightly coupled to tmux
type SessionManager struct {
    tmux *tmux.Tmux  // ← Only this varies
    rig  *rig.Rig
}

// Could become: pluggable backend
type SessionManager struct {
    backend SessionBackend  // Interface
    rig     *rig.Rig
}

type SessionBackend interface {
    NewSession(opts SessionStartOptions) (*SessionInfo, error)
    SendKeys(sessionName, keys string) error
    IsAgentRunning(sessionName string) bool
}
```

**Estimated change:** 1,500-2,000 new lines, 300-500 modified lines.

### Beads Works Everywhere

Beads is git-backed issue tracking. It works over SSH:

```bash
# On laptop (Mayor)
bd create "Fix the auth bug"
git push

# In cluster (Polecat Pod)
git pull  # Gets the issue
bd update gt-0042 --status in_progress
git push  # Reports progress
```

**This is the key:** State lives in git, not in process memory.

---

## Proposed Architecture

### Hybrid Mode: Local Mayor + Remote Polecats

```yaml
# Rig configuration with remote backend
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: my-project
spec:
  gitURL: "git@github.com:myorg/my-project.git"
  beadsPrefix: "proj"

  # NEW: Execution backend
  backend:
    type: kubernetes  # "local" | "kubernetes"
    kubernetes:
      namespace: gastown
      image: "gastown-polecat:latest"

  # Claude credentials (synced from laptop)
  claudeCredentials:
    secretRef:
      name: claude-creds
```

### Workflow

```bash
# 1. User authenticates locally (once)
claude login

# 2. Sync credentials to cluster (once, or when re-auth needed)
gt remote sync-auth --cluster my-cluster

# 3. Use gt exactly as before
gt sling proj-0042 my-project

# 4. Under the hood:
#    - If rig.backend.type == "kubernetes":
#        - Create Pod with git init + claude container
#        - Mount claude-creds secret
#        - Pod runs `claude --dangerously-skip-permissions`
#    - Else:
#        - Current behavior (tmux session)
```

### Authentication Sync

```bash
# New command: gt remote sync-auth
#
# 1. Reads ~/.claude/ from laptop
# 2. Creates/updates Secret in cluster
# 3. Stores hash for change detection

$ gt remote sync-auth --cluster my-cluster --namespace gastown

Syncing Claude credentials to my-cluster...
  - Reading ~/.claude/
  - Creating Secret gastown/claude-creds
  - Storing sync timestamp
Done. Polecats in gastown namespace can now use your Claude account.

# Re-run if you re-authenticate:
$ claude login
$ gt remote sync-auth --cluster my-cluster
```

---

## The Operator's Role

### What gastown_operator Does

```
┌─────────────────────────────────────────────────────────────┐
│                    OpenShift Cluster                         │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              GASTOWN OPERATOR                           │ │
│  │                                                         │ │
│  │  Rig CRD ──────────> Validates config, creates NS      │ │
│  │                                                         │ │
│  │  Polecat CRD ──────> Creates Pod with:                 │ │
│  │      │               - Git init container              │ │
│  │      │               - Claude container                │ │
│  │      │               - claude-creds Secret mount       │ │
│  │      │               - EmptyDir workspace              │ │
│  │      └──────────> Updates status from Pod phase        │ │
│  │                                                         │ │
│  │  Convoy CRD ───────> Aggregates Polecat status         │ │
│  │                                                         │ │
│  │  BeadStore CRD ────> CronJob for git sync              │ │
│  │                                                         │ │
│  └────────────────────────────────────────────────────────┘ │
│                                                              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │              RESOURCES CREATED                          │ │
│  │                                                         │ │
│  │  Secret: claude-creds ───> Mounted in all polecat pods │ │
│  │  Secret: git-creds ──────> SSH key for git operations  │ │
│  │  Pods: polecat-* ────────> Worker containers           │ │
│  │  PVCs (optional) ────────> For large repos             │ │
│  │                                                         │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
```

### gt CLI Changes (Minimal)

```go
// New in internal/cmd/sling.go

func (c *SlingCmd) Execute() error {
    rig := c.resolveRig(target)

    // NEW: Check backend type
    if rig.Backend.Type == "kubernetes" {
        return c.slingToKubernetes(rig, beadID)
    }

    // Existing: local tmux
    return c.slingToLocal(rig, beadID)
}

func (c *SlingCmd) slingToKubernetes(rig *Rig, beadID string) error {
    // Create Polecat CRD, let operator handle the rest
    polecat := &gastownv1alpha1.Polecat{
        Spec: gastownv1alpha1.PolecatSpec{
            Rig:          rig.Name,
            BeadID:       beadID,
            DesiredState: "Working",
        },
    }
    return c.kubeClient.Create(ctx, polecat)
}
```

### How It Looks to Users

**Before (local only):**
```bash
gt sling proj-0042 my-project
# → Creates tmux session on laptop
# → Claude runs locally
```

**After (hybrid):**
```bash
gt sling proj-0042 my-project
# → Creates Polecat CRD
# → Operator creates Pod in cluster
# → Claude runs in cluster
# → Same beads tracking, same git state
```

**Same command. Different execution.**

---

## Polecat Pod Specification

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: polecat-toast-001
  namespace: gastown
  labels:
    gastown.io/rig: my-project
    gastown.io/polecat: toast-001
    gastown.io/bead: proj-0042
spec:
  restartPolicy: Never

  initContainers:
    - name: git-init
      image: alpine/git:latest
      command: ["/bin/sh", "-c"]
      args:
        - |
          mkdir -p ~/.ssh
          cp /git-creds/ssh-privatekey ~/.ssh/id_rsa
          chmod 600 ~/.ssh/id_rsa
          ssh-keyscan github.com >> ~/.ssh/known_hosts
          git clone --depth=1 -b main $GIT_URL /workspace/repo
          cd /workspace/repo
          git checkout -b feature/proj-0042
      env:
        - name: GIT_URL
          value: "git@github.com:myorg/my-project.git"
      volumeMounts:
        - name: workspace
          mountPath: /workspace
        - name: git-creds
          mountPath: /git-creds
          readOnly: true

  containers:
    - name: claude
      image: node:20-slim
      command: ["/bin/sh", "-c"]
      args:
        - |
          npm install -g @anthropic-ai/claude-code
          export CLAUDE_CONFIG_DIR=/claude-creds
          exec claude --dangerously-skip-permissions
      workingDir: /workspace/repo
      env:
        - name: GT_ISSUE
          value: "proj-0042"
        - name: GT_POLECAT
          value: "toast-001"
        - name: GT_RIG
          value: "my-project"
      volumeMounts:
        - name: workspace
          mountPath: /workspace
        - name: claude-creds
          mountPath: /claude-creds
          readOnly: true
      resources:
        requests:
          cpu: "500m"
          memory: "1Gi"
        limits:
          cpu: "2"
          memory: "4Gi"

  volumes:
    - name: workspace
      emptyDir: {}
    - name: git-creds
      secret:
        secretName: git-creds
    - name: claude-creds
      secret:
        secretName: claude-creds
```

---

## Your Niche: "The Operator Guy"

### What Steve Built

- **Gas Town CLI** - Local orchestration, tmux-based
- **Beads** - Git-backed issue tracking
- **The workflow** - Mayor, Polecats, Convoys, Sling

### What You Build

- **gastown_operator** - K8s-native execution layer
- **Hybrid mode** - Local Mayor + Remote Polecats
- **Enterprise features** - Observability, RBAC, resource limits
- **Auth sync** - `gt remote sync-auth` command

### The Value Proposition

| Audience | Value |
|----------|-------|
| **Individual devs** | "My laptop doesn't burn up running 10 agents" |
| **Teams** | "Shared cluster, shared visibility, one subscription" |
| **Enterprises** | "Same workflow, our infrastructure, our compliance" |

### Differentiation from Competitors

**vs claude-flow (ruvnet):**
- They: MCP protocol, swarm intelligence, new paradigm
- You: Same Gas Town workflow, just runs remotely

**vs local-only Gas Town:**
- Steve: "Works on your laptop"
- You: "Works on your cluster"

### Strategic Position

```
                    Gas Town Ecosystem
                           │
        ┌──────────────────┼──────────────────┐
        │                  │                  │
   ┌────▼────┐       ┌─────▼─────┐      ┌─────▼─────┐
   │  Beads  │       │  gt CLI   │      │ Operator  │
   │ (Steve) │       │  (Steve)  │      │  (Boden)  │
   └─────────┘       └───────────┘      └───────────┘
        │                  │                  │
        └──────────────────┼──────────────────┘
                           │
                  Works together
```

---

## Implementation Roadmap

### Phase 1: Foundation (2 weeks)

**Goal:** `gt sling` creates Polecat Pod that runs Claude Code

| Week | Deliverable |
|------|-------------|
| 1 | Polecat controller creates Pod (not Job, for session persistence) |
| 1 | Git init container clones repo |
| 2 | Claude container runs with mounted creds |
| 2 | `gt remote sync-auth` command |

**Success criteria:**
```bash
# User syncs auth once
gt remote sync-auth --cluster my-cluster

# User slings to kubernetes backend
gt sling proj-0042 my-project --backend kubernetes

# Pod runs, Claude does work, commits pushed
kubectl logs polecat-toast-001 -c claude
# Shows: Claude working on proj-0042
```

### Phase 2: Integration (2 weeks)

**Goal:** Seamless hybrid mode with backend in rig config

| Week | Deliverable |
|------|-------------|
| 3 | Rig config supports `backend: kubernetes` |
| 3 | `gt sling` automatically routes based on config |
| 4 | Convoy aggregation works across local + remote |
| 4 | BeadStore sync CronJob |

**Success criteria:**
```bash
# Configure rig for kubernetes
gt rig set-backend my-project kubernetes --namespace gastown

# Sling works exactly as before
gt sling proj-0042 my-project
# → Routes to cluster automatically

# Convoy shows progress
gt convoy status wave-1
# Shows local + remote polecats together
```

### Phase 3: Polish (2 weeks)

**Goal:** Production-ready with observability

| Week | Deliverable |
|------|-------------|
| 5 | Prometheus metrics for polecat lifecycle |
| 5 | Pod logs accessible via `gt polecat logs` |
| 6 | Resource limits configurable per rig |
| 6 | Helm chart for operator deployment |

---

## Technical Decisions

### Pod vs Job

**Decision: Pod (not Job)**

Rationale:
- Jobs have restart semantics we don't want
- Pods can be monitored for activity
- Session resumption needs stable identity
- Polecat lifecycle managed by operator, not Job controller

### EmptyDir vs PVC

**Decision: EmptyDir by default, PVC option**

Rationale:
- Most repos fit in EmptyDir
- No provisioning wait
- PVC for large monorepos (configurable)

### Single Namespace vs Multi

**Decision: Single namespace per rig**

Rationale:
- Simpler RBAC
- One BeadStore per namespace
- Multi-namespace = multiple operator instances (future)

### Image Strategy

**Decision: Base image + npm install at runtime**

Rationale:
- Claude CLI updates frequently
- Don't want to rebuild image on every release
- Startup cost acceptable (one-time per pod)

Future: Pre-built image with pinned Claude version for enterprise.

---

## Open Questions

### 1. Session Persistence Across Pod Restarts

**Problem:** If pod dies, how do we resume the Claude session?

**Options:**
- A) Don't resume - let polecat re-read issue and continue
- B) Save session ID in annotation, pass `--resume` on restart
- C) PVC for `~/.claude-code/` session state

**Recommendation:** Option A for v1. Beads has full context, Claude can continue.

### 2. Authentication Refresh

**Problem:** OAuth2 tokens expire. What happens when cluster creds expire?

**Options:**
- A) Manual: User re-runs `gt remote sync-auth`
- B) Semi-auto: Operator alerts when pods fail auth
- C) Auto: Daemon on laptop syncs periodically

**Recommendation:** Option B for v1. Alert + easy fix command.

### 3. Cost Attribution

**Problem:** Multiple users sharing one Claude subscription?

**Options:**
- A) Single user subscription (current Gas Town model)
- B) Anthropic organization billing (enterprise)
- C) LiteLLM proxy with per-user tracking

**Recommendation:** Option A for Claude Code first. Option C for OpenCode phase.

---

## Success Metrics

### Adoption Indicators

- [ ] 10 users try hybrid mode (beta)
- [ ] 3 production deployments on OpenShift
- [ ] Gas Town docs link to operator

### Technical Indicators

- [ ] Pod startup < 60 seconds
- [ ] 95% polecat completion rate
- [ ] Zero auth failures after sync

### Community Indicators

- [ ] Steve mentions operator in blog/docs
- [ ] GitHub stars on gastown_operator
- [ ] Contributions from Gas Town community

---

## Conclusion

**The path to adoption:**

1. **Claude Code first** - Users have subscriptions, use OAuth2
2. **Same workflow** - `gt sling` just works
3. **Remote execution** - Laptop orchestrates, cluster executes
4. **Your niche** - "The Operator Guy" bridges local and enterprise

**Estimated effort:** 6 weeks for production-ready hybrid mode

**Next step:** `/formulate` to create implementation plan

---

## References

- [Gas Town GitHub](https://github.com/steveyegge/gastown)
- [Steve Yegge's announcement](https://steve-yegge.medium.com/welcome-to-gas-town-4f25ee16dd04)
- [Gas Town workflow analysis](https://paddo.dev/blog/gastown-two-kinds-of-multi-agent/)
- Previous research: `2026-01-17-native-k8s-evolution.md`
