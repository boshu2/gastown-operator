# kubectl-gt

A kubectl plugin for managing Gas Town resources in Kubernetes.

kubectl-gt follows the CNPG pattern: the CLI creates CRDs (Rig, Polecat, Convoy) that the gastown-operator reconciles into actual workloads. This enables cloud-native execution of Gas Town workflows.

## Installation

### Using Krew (Recommended)

```bash
kubectl krew install gt
```

### From Binary

Download the latest release from [GitHub Releases](https://github.com/olympus-cloud/gastown-operator/releases) and extract to your PATH:

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/olympus-cloud/gastown-operator/releases/latest/download/kubectl-gt-darwin-arm64.tar.gz
tar -xzf kubectl-gt-darwin-arm64.tar.gz
mv kubectl-gt /usr/local/bin/

# Linux (amd64)
curl -LO https://github.com/olympus-cloud/gastown-operator/releases/latest/download/kubectl-gt-linux-amd64.tar.gz
tar -xzf kubectl-gt-linux-amd64.tar.gz
mv kubectl-gt /usr/local/bin/
```

### From Source

```bash
cd cmd/kubectl-gt
make install
```

## Prerequisites

- Kubernetes cluster with gastown-operator installed
- kubectl configured with cluster access
- Claude credentials (for auth sync)

## Quick Start

```bash
# 1. Sync your Claude credentials to the cluster
kubectl gt auth sync

# 2. Create a rig
kubectl gt rig create my-rig \
  --git-url https://github.com/org/repo.git \
  --prefix mr \
  --local-path /path/to/repo

# 3. Dispatch work to a polecat
kubectl gt sling be-0001 my-rig --wait

# 4. Check polecat status
kubectl gt polecat list my-rig

# 5. Track batch work with convoys
kubectl gt convoy create "Wave 1" be-0001 be-0002 be-0003
kubectl gt convoy status cv-xxxx
```

## Commands

### rig - Manage Gas Town rigs

```bash
# List all rigs
kubectl gt rig list
kubectl gt rig list -o yaml

# Show rig details
kubectl gt rig status my-rig

# Create a rig
kubectl gt rig create my-rig \
  --git-url https://github.com/org/repo.git \
  --prefix mr \
  --local-path /path/to/repo
```

### polecat - Manage polecat workers

```bash
# List polecats (optionally filter by rig)
kubectl gt polecat list
kubectl gt polecat list my-rig

# Show polecat details
kubectl gt polecat status my-rig/polecat-name

# Stream logs
kubectl gt polecat logs my-rig/polecat-name -f

# Terminate a polecat
kubectl gt polecat nuke my-rig/polecat-name
kubectl gt polecat nuke my-rig/polecat-name --force
```

### sling - Dispatch work to a polecat

```bash
# Basic dispatch
kubectl gt sling be-0001 my-rig

# Wait for polecat to start
kubectl gt sling be-0001 my-rig --wait

# Custom timeout
kubectl gt sling be-0001 my-rig --wait --timeout=5m
```

### convoy - Manage convoy (batch) tracking

```bash
# List all convoys
kubectl gt convoy list

# Create a convoy
kubectl gt convoy create "Wave 1 tasks" be-0001 be-0002 be-0003

# Check convoy progress
kubectl gt convoy status cv-xxxx
```

### auth - Manage Claude authentication

```bash
# Sync ~/.claude/ to Kubernetes Secret
kubectl gt auth sync
kubectl gt auth sync --force

# Check credential status
kubectl gt auth status
```

## Architecture

```
┌─────────────────┐     ┌─────────────────┐     ┌─────────────────┐
│   kubectl-gt    │────>│   Kubernetes    │────>│ gastown-operator│
│   (CLI)         │     │   API Server    │     │ (Controller)    │
└─────────────────┘     └─────────────────┘     └─────────────────┘
        │                       │                       │
        │ Creates CRDs          │ Stores CRDs           │ Reconciles
        ▼                       ▼                       ▼
   ┌─────────┐            ┌─────────┐            ┌─────────┐
   │ Rig CR  │            │Polecat  │            │  Pods   │
   │         │            │  CRs    │            │(claude) │
   └─────────┘            └─────────┘            └─────────┘
```

**Comparison with local `gt` CLI:**

| Feature | `gt` CLI (local) | `kubectl gt` (cloud) |
|---------|------------------|----------------------|
| Execution | tmux sessions | Kubernetes Pods |
| Storage | Local filesystem | PVCs + Git |
| Scaling | Single machine | Multi-node cluster |
| Creds | ~/.claude/ | K8s Secrets |
| Use case | Development | Production |

## Troubleshooting

### Polecat stuck in "Pending"

Check if claude-creds Secret exists:
```bash
kubectl gt auth status -n gastown-system
# If missing:
kubectl gt auth sync -n gastown-system
```

### Rig shows "Degraded" phase

The operator can't verify the local path. This is expected when running cloud-native (the path only needs to exist on the operator host for local mode).

### "Rig not found" error

Rigs are cluster-scoped. Don't use `-n namespace` for rig operations:
```bash
kubectl gt rig list        # Correct
kubectl gt rig list -n ns  # Works, but namespace is ignored for rigs
```

### Debug with verbose output

```bash
kubectl gt --v=6 rig list
```

## Global Flags

All commands support standard kubectl flags:

- `--kubeconfig` - Path to kubeconfig file
- `-n, --namespace` - Target namespace
- `--context` - Kubeconfig context
- `-s, --server` - API server address

## License

Apache License 2.0
