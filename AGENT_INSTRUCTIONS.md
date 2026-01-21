# Gas Town Operator - Kubernetes Execution Layer

> Run polecats as pods. Scale your AI agent army beyond the laptop.

---

## Identity

| Attribute | Value |
|-----------|-------|
| **Name** | gastown-operator |
| **Role** | Kubernetes execution for Gas Town |
| **Repository** | boshu2/gastown-operator |
| **Helm Chart** | `oci://ghcr.io/boshu2/charts/gastown-operator` |
| **Container** | `ghcr.io/boshu2/gastown-operator` |

---

## Purpose

The gastown-operator extends [Gas Town](https://github.com/steveyegge/gastown) to Kubernetes. Instead of running polecats locally via tmux, they run as pods in your cluster.

**What it provides:**
1. **Polecat pods** - AI agents running as Kubernetes pods
2. **CRD-based management** - Rig, Polecat, Convoy, Witness, Refinery, BeadStore
3. **Git integration** - Clone repos, create branches, push commits
4. **Multiple agents** - claude-code (default), opencode, aider, or custom

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                              │
│   ┌──────────────────────────────────────────────────────┐  │
│   │              gastown-operator                         │  │
│   │                                                       │  │
│   │   Rig       Polecat     Convoy     Witness           │  │
│   │ Controller  Controller  Controller  Controller       │  │
│   └─────────────────────┬────────────────────────────────┘  │
│                         │                                    │
│         ┌───────────────┴───────────────┐                   │
│         ▼                               ▼                   │
│   ┌───────────┐                   ┌───────────┐            │
│   │  Polecat  │        ...        │  Polecat  │            │
│   │   (Pod)   │                   │   (Pod)   │            │
│   │ ┌───────┐ │                   │ ┌───────┐ │            │
│   │ │Claude │ │                   │ │Claude │ │            │
│   │ │ Code  │ │                   │ │ Code  │ │            │
│   │ └───────┘ │                   │ └───────┘ │            │
│   └───────────┘                   └───────────┘            │
└─────────────────────────────────────────────────────────────┘
```

---

## Installation

```bash
# Standard Kubernetes
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace

# OpenShift (restricted SCC)
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace \
  --set securityContext.allowPrivilegeEscalation=false \
  --set securityContext.runAsNonRoot=true \
  --set securityContext.runAsUser=null \
  --set securityContext.readOnlyRootFilesystem=true
```

---

## Secrets Setup

Polecats need two secrets in the `gastown-workers` namespace:

### Git SSH Key

```bash
kubectl create namespace gastown-workers

kubectl create secret generic git-credentials -n gastown-workers \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519
```

### Claude Credentials

**Option A: API Key (recommended for automation)**

```bash
kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=api-key=$ANTHROPIC_API_KEY
```

**Option B: OAuth (from `claude /login`)**

```bash
# macOS - extract from Keychain
CREDS=$(security find-generic-password -s "Claude Code-credentials" -w)
kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=.credentials.json="$CREDS"
```

---

## Custom Resources

| CRD | Description |
|-----|-------------|
| **Rig** | Project workspace (cluster-scoped) |
| **Polecat** | Autonomous worker agent pod |
| **Convoy** | Batch tracking for parallel execution |
| **Witness** | Worker lifecycle monitor |
| **Refinery** | Merge queue processor |
| **BeadStore** | Issue tracking backend |

---

## Usage Pattern

**You don't manually write Polecat CRs.** The normal workflow is:

1. Install operator (above)
2. Create secrets (above)
3. From Mayor session: `gt sling issue-123 my-rig --mode kubernetes`
4. Claude generates the Polecat CR
5. Operator creates the pod
6. Watch progress: `gt convoy list` or `kubectl logs -f polecat-issue-123`

---

## Polecat CR Example

If you DO need to create a Polecat manually (testing, debugging):

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-worker
  namespace: gastown-workers
spec:
  rig: my-project
  beadID: issue-123
  desiredState: Working
  executionMode: kubernetes
  taskDescription: |
    Implement the /health endpoint.
    Commit and push when done.
  kubernetes:
    gitRepository: "git@github.com:myorg/myrepo.git"
    gitBranch: main
    gitSecretRef:
      name: git-credentials
    apiKeySecretRef:
      name: claude-credentials
      key: api-key
```

---

## Verification Commands

```bash
# Check operator
kubectl get pods -n gastown-system
kubectl logs -f deployment/gastown-operator -n gastown-system

# Check CRDs
kubectl get crds | grep gastown

# Check polecats
kubectl get polecats -A
kubectl logs -f polecat-<name> -n gastown-workers
```

---

## Troubleshooting

### Pod stuck in Pending

```bash
kubectl describe pod polecat-<name> -n gastown-workers
```

Common causes: secret doesn't exist, insufficient resources.

### Git clone fails

```bash
kubectl logs polecat-<name> -c git-init -n gastown-workers
```

Verify SSH key format: `kubectl get secret git-credentials -n gastown-workers -o jsonpath='{.data.ssh-privatekey}' | base64 -d | head -1`

### Claude auth fails

```bash
kubectl logs polecat-<name> -c claude -n gastown-workers
```

Check API key is valid or OAuth tokens haven't expired (24hr lifetime).

---

## Related

| Project | Relationship |
|---------|--------------|
| [Gas Town](https://github.com/steveyegge/gastown) | Local execution (gt CLI) |
| [Beads](https://github.com/steveyegge/beads) | Issue tracking (bd CLI) |
| [Claude Code](https://github.com/anthropics/claude-code) | Default agent |

---

## Documentation

- [USER_GUIDE.md](docs/USER_GUIDE.md) - Complete setup walkthrough
- [CRD_REFERENCE.md](docs/CRD_REFERENCE.md) - Full spec/status docs
- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - Common issues
