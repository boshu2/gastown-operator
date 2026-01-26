# Gas Town Operator - Kubernetes Execution Layer

> Run polecats as pods. Scale your AI agent army beyond the laptop.

**Quick Links:** [kubectl-gt CLI](#kubectl-gt-cli) | [templates/](templates/) (YAML examples) | [FRICTION_POINTS.md](FRICTION_POINTS.md) (anti-patterns)

---

## Identity

| Attribute | Value |
|-----------|-------|
| **Name** | gastown-operator |
| **Version** | 0.4.2 |
| **Role** | Kubernetes execution for Gas Town |
| **Repository** | boshu2/gastown-operator |
| **Helm Chart** | `oci://ghcr.io/boshu2/charts/gastown-operator` |
| **Container** | `ghcr.io/boshu2/gastown-operator` |

---

## Purpose

The gastown-operator extends [Gas Town](https://github.com/steveyegge/gastown) to Kubernetes. Instead of running polecats locally via tmux, they run as pods in your cluster.

**What it provides:**
1. **Polecat pods** - Claude Code agents running as Kubernetes pods
2. **kubectl-gt CLI** - AI-native interface for managing Gas Town resources
3. **CRD-based management** - Rig, Polecat, Convoy, Witness, Refinery, BeadStore
4. **Git integration** - Clone repos, create branches, push commits

---

## Quick Start (CLI-First)

### 1. Install Operator

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.2 \
  --namespace gastown-system \
  --create-namespace
```

### 2. Install kubectl-gt Plugin

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.2/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt

# Linux
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.2/kubectl-gt-linux-amd64
chmod +x kubectl-gt-linux-amd64 && sudo mv kubectl-gt-linux-amd64 /usr/local/bin/kubectl-gt
```

### 3. Set Up Credentials

```bash
# Create git credentials secret
kubectl create secret generic git-creds -n gastown-system \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519

# Sync Claude credentials from local ~/.claude/
kubectl gt auth sync -n gastown-system
```

### 4. Create Rig and Dispatch Work

```bash
# Create a project rig
kubectl gt rig create my-project \
  --git-url git@github.com:myorg/myrepo.git \
  --prefix mp \
  -n gastown-system

# Dispatch work to a polecat
kubectl gt sling issue-123 my-project --name furiosa -n gastown-system

# Watch it work
kubectl gt polecat logs my-project/furiosa -f -n gastown-system
```

---

## kubectl-gt CLI

**The recommended way to interact with Gas Town.** AI-native interface designed for both humans and agents.

### Commands

| Command | Description |
|---------|-------------|
| `kubectl gt rig list` | List all rigs |
| `kubectl gt rig status <name>` | Show rig details |
| `kubectl gt rig create <name>` | Create a new rig |
| `kubectl gt polecat list [rig]` | List polecats |
| `kubectl gt polecat status <rig>/<name>` | Show polecat details |
| `kubectl gt polecat logs <rig>/<name>` | Stream polecat logs |
| `kubectl gt polecat nuke <rig>/<name>` | Terminate a polecat |
| `kubectl gt sling <bead-id> <rig>` | Dispatch work to a polecat |
| `kubectl gt convoy list` | List convoy batches |
| `kubectl gt convoy create <desc> <beads...>` | Create convoy |
| `kubectl gt auth sync` | Sync Claude creds to cluster |
| `kubectl gt auth status` | Check credential status |

### AI-Native Features

**JSON/YAML Output** - All commands support `-o json` and `-o yaml`:
```bash
kubectl gt polecat list -o json | jq '.[] | .metadata.name'
```

**Themed Naming** - Memorable names for polecats:
```bash
kubectl gt sling issue-123 my-project --theme mad-max
# Creates polecat named "furiosa", "nux", "toast", etc.
```

**Wait for Ready** - Block until pod is running:
```bash
kubectl gt sling issue-123 my-project --wait-ready --timeout 5m
```

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

## Verification Commands

```bash
# Check operator
kubectl get pods -n gastown-system
kubectl logs -f deployment/gastown-operator -n gastown-system

# Check CRDs
kubectl get crds | grep gastown

# Check resources via CLI
kubectl gt rig list -n gastown-system
kubectl gt polecat list -n gastown-system
kubectl gt convoy list -n gastown-system
```

---

## Troubleshooting

### Pod stuck in Pending

```bash
kubectl gt polecat status my-rig/my-polecat -n gastown-system
kubectl describe pod polecat-<name> -n gastown-system
```

Common causes: secret doesn't exist, insufficient resources.

### Git clone fails

```bash
kubectl gt polecat logs my-rig/my-polecat -n gastown-system
```

Verify SSH key: `kubectl get secret git-creds -n gastown-system -o jsonpath='{.data.ssh-privatekey}' | base64 -d | head -1`

### Claude auth fails

```bash
kubectl gt auth status -n gastown-system
```

Re-sync if stale: `kubectl gt auth sync --force -n gastown-system`

---

## YAML Templates (Alternative)

If you prefer declarative YAML, templates are in [templates/](templates/):

| Template | Use Case |
|----------|----------|
| [polecat-minimal.yaml](templates/polecat-minimal.yaml) | Quick polecat |
| [polecat-kubernetes.yaml](templates/polecat-kubernetes.yaml) | Full K8s execution |
| [convoy.yaml](templates/convoy.yaml) | Batch tracking |

**Validation:** `./scripts/validate-template.sh <file>`

---

## Related

| Project | Relationship |
|---------|--------------|
| [Gas Town](https://github.com/steveyegge/gastown) | Local execution (gt CLI) |
| [Beads](https://github.com/steveyegge/beads) | Issue tracking (bd CLI) |
| [Claude Code](https://github.com/anthropics/claude-code) | Default agent |

---

## Documentation

| Document | Purpose |
|----------|---------|
| [README.md](README.md) | Main documentation with CLI reference |
| [templates/](templates/) | YAML templates with `{{VARIABLE}}` markers |
| [FRICTION_POINTS.md](FRICTION_POINTS.md) | Anti-patterns and common mistakes |
| [docs/USER_GUIDE.md](docs/USER_GUIDE.md) | Complete setup walkthrough |
