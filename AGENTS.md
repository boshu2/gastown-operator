# Agent Instructions

See **[AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md)** for complete agent context.

This file exists for compatibility with tools that look for AGENTS.md.

## Quick Reference (CLI-First)

```bash
# Install operator
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.2 \
  --namespace gastown-system \
  --create-namespace

# Install kubectl-gt plugin
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.2/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt

# Set up credentials
kubectl create secret generic git-creds -n gastown-system \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519
kubectl gt auth sync -n gastown-system

# Create rig and dispatch work
kubectl gt rig create my-project --git-url git@github.com:org/repo.git --prefix mp
kubectl gt sling issue-123 my-project --name furiosa
kubectl gt polecat logs my-project/furiosa -f
```

## CLI Commands

| Command | Description |
|---------|-------------|
| `kubectl gt rig list` | List rigs |
| `kubectl gt rig create <name>` | Create rig |
| `kubectl gt polecat list` | List polecats |
| `kubectl gt polecat logs <rig>/<name>` | Stream logs |
| `kubectl gt sling <bead> <rig>` | Dispatch work |
| `kubectl gt convoy list` | List convoys |
| `kubectl gt auth sync` | Sync Claude creds |

## Full Documentation

- [README.md](README.md) - Main docs with CLI reference
- [AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md) - Agent context
- [docs/USER_GUIDE.md](docs/USER_GUIDE.md) - Complete walkthrough

## Key Point

**Use the kubectl-gt CLI** for normal workflows. YAML templates are available in [templates/](templates/) for advanced use cases or GitOps.
