# Gas Town Operator User Guide

Run AI coding agents as Kubernetes pods. Scale your agent army beyond the laptop.

**Quick Links:** [kubectl-gt CLI](#kubectl-gt-cli-recommended) | [YAML Templates](#yaml-alternative) | [Troubleshooting](#troubleshooting)

---

## Supported Agent

The operator runs **Claude Code** - Anthropic's official coding agent CLI.

| Agent | Description |
|-------|-------------|
| `claude-code` | Anthropic's Claude Code CLI |

## How It Works

The operator uses a **laptop replica** pattern:

```
Your Laptop                          Kubernetes Cluster
┌─────────────────┐                  ┌─────────────────────────────┐
│ claude /login   │                  │     gastown-operator        │
│      ↓          │                  │            ↓                │
│ ~/.claude/      │  ──export───→    │   Secret: claude-home       │
│ (OAuth tokens)  │                  │            ↓                │
└─────────────────┘                  │   Polecat CR → Pod          │
                                     │   ┌─────────────────────┐   │
                                     │   │ git-init (clone)    │   │
                                     │   │ claude (agent)      │   │
                                     │   │  └─ OAuth auth      │   │
                                     │   │  └─ Execute work    │   │
                                     │   └─────────────────────┘   │
                                     └─────────────────────────────┘
```

Authentication options:
- **OAuth session** from `claude /login` (recommended)
- **API key** for headless deployments

## Prerequisites

- OpenShift/Kubernetes 1.26+
- `kubectl` or `oc` CLI
- Git SSH key for repository access

---

## Quick Start (CLI-First)

### 1. Install Operator

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.1 \
  --namespace gastown-system \
  --create-namespace
```

### 2. Install kubectl-gt Plugin

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt

# Linux
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-linux-amd64
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
kubectl gt rig create myproject \
  --git-url git@github.com:myorg/myrepo.git \
  --prefix mp \
  -n gastown-system

# Dispatch work to a polecat
kubectl gt sling issue-123 myproject --name furiosa -n gastown-system

# Watch it work
kubectl gt polecat logs myproject/furiosa -f -n gastown-system
```

---

## kubectl-gt CLI (Recommended)

The **kubectl-gt** plugin provides an AI-native interface for Gas Town resources.

### Core Commands

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

**JSON/YAML Output** - Machine-parseable output for automation:
```bash
kubectl gt polecat list -o json | jq '.[] | .metadata.name'
```

**Themed Naming** - Memorable names for polecats:
```bash
kubectl gt sling issue-123 myproject --theme mad-max
# Creates polecat named "furiosa", "nux", "toast", etc.
```

**Wait for Ready** - Block until pod is running:
```bash
kubectl gt sling issue-123 myproject --wait-ready --timeout 5m
```

---

## Watch It Work

```bash
# Polecat status
kubectl gt polecat status myproject/furiosa -n gastown-system

# Stream logs
kubectl gt polecat logs myproject/furiosa -f -n gastown-system

# Pod status (raw)
kubectl get pods -n gastown-system
```

---

## E2E Proof: It Actually Works

### Test 3: CLI E2E (2026-01-25)

**Full end-to-end test using kubectl-gt CLI:**

```bash
$ kubectl gt rig create demo-rig --git-url git@github.com:boshu2/gastown-operator.git --prefix dm
Rig demo-rig created

$ kubectl gt sling dm-0001 demo-rig --name furiosa
Polecat furiosa created for bead dm-0001 in rig demo-rig

$ kubectl gt polecat list
NAME      RIG        PHASE     BEAD
furiosa   demo-rig   Working   dm-0001

$ kubectl gt sling dm-0002 demo-rig --theme mad-max
Polecat ace created for bead dm-0002 in rig demo-rig

$ kubectl gt polecat logs demo-rig/furiosa --tail 20
Claude credentials loaded...
Working on bead dm-0001...
```

### Test 2: PR Creation (2026-01-20)

**Full end-to-end test with git push and PR creation:**

```
$ kubectl get polecat feature-version-endpoint -n gastown
NAME                       RIG        MODE         AGENT         PHASE   BEAD
feature-version-endpoint   test-rig   kubernetes   claude-code   Done    go-cwl
```

**Result:** Claude implemented feature, committed, pushed, and created PR.

**PR Created:** https://github.com/boshu2/gastown-operator/pull/1

---

## Token Refresh

OAuth tokens expire after ~24 hours. To refresh:

```bash
# 1. Re-login on your laptop (opens browser)
claude /login

# 2. Re-sync credentials
kubectl gt auth sync --force -n gastown-system
```

---

## YAML Alternative

If you prefer declarative YAML for GitOps workflows:

### Polecat with OAuth Credentials

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-worker
  namespace: gastown-system
spec:
  rig: myproject
  beadID: issue-123
  taskDescription: |
    Add a /health endpoint that returns {"status": "ok"}.
    After implementing, commit and push the changes.
  desiredState: Working
  executionMode: kubernetes
  kubernetes:
    gitRepository: "git@github.com:myorg/myrepo.git"
    gitBranch: main
    workBranch: feature/issue-123
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

### Polecat with API Key (Headless)

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: api-worker
  namespace: gastown-system
spec:
  rig: myproject
  beadID: issue-456
  desiredState: Working
  executionMode: kubernetes
  kubernetes:
    gitRepository: "git@github.com:myorg/myrepo.git"
    gitBranch: main
    gitSecretRef:
      name: git-creds
    apiKeySecretRef:
      name: anthropic-api-key
      key: api-key
```

**Apply with:** `kubectl apply -f polecat.yaml`

See [templates/](../templates/) for more examples.

---

## Pod Architecture

Each Polecat pod has:

| Container | Purpose |
|-----------|---------|
| `git-init` (init) | Clone repo, create feature branch |
| `claude` (main) | Run Claude Code agent |

### Security Context

All pods run with OpenShift restricted SCC compliance:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

---

## Troubleshooting

### "OAuth token has expired"

```bash
kubectl gt auth sync --force -n gastown-system
```

### Pod stuck in Pending

```bash
kubectl gt polecat status myproject/my-polecat -n gastown-system
kubectl describe pod polecat-<name> -n gastown-system
```

Common causes:
- Secret doesn't exist
- Insufficient resources
- Node scheduling issues

### Git clone fails

```bash
kubectl gt polecat logs myproject/my-polecat -n gastown-system
```

Verify SSH key:
```bash
kubectl get secret git-creds -n gastown-system -o jsonpath='{.data.ssh-privatekey}' | base64 -d | head -1
```

Should show: `-----BEGIN OPENSSH PRIVATE KEY-----`

### Claude container exits immediately

```bash
kubectl gt polecat logs myproject/my-polecat -n gastown-system
```

Common causes:
- Invalid credentials format
- Network connectivity to Anthropic API
- Missing repository files (CLAUDE.md)

---

## Reference

- [CRD Reference](./CRD_REFERENCE.md) - Full spec/status documentation
- [Secret Management](./SECRET_MANAGEMENT.md) - Credential setup
- [Architecture](./architecture.md) - How the operator works
- [Development](./development.md) - Contributing guide
