<div align="center">

# Gas Town Operator

**Kubernetes operator for Gas Town polecats**

[![Release](https://img.shields.io/github/v/release/boshu2/gastown-operator?logo=github)](https://github.com/boshu2/gastown-operator/releases/latest)
[![Helm](https://img.shields.io/badge/Helm-OCI-blue?logo=helm)](https://ghcr.io/boshu2/charts/gastown-operator)
[![GHCR](https://img.shields.io/badge/GHCR-Container-purple?logo=github)](https://ghcr.io/boshu2/gastown-operator)
[![OpenShift](https://img.shields.io/badge/OpenShift-Native-EE0000?logo=redhatopenshift)](https://www.redhat.com/en/technologies/cloud-computing/openshift)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

*A Kubernetes operator that runs [Gas Town](https://github.com/steveyegge/gastown) polecats as pods.*
*Scale your AI agent army beyond the laptop.*

</div>

## Quick Start

### 1. Install the Operator

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.1 \
  --namespace gastown-system \
  --create-namespace
```

### 2. Install the kubectl Plugin

```bash
# Download from releases
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m)
chmod +x kubectl-gt-* && sudo mv kubectl-gt-* /usr/local/bin/kubectl-gt
```

### 3. Create a Rig and Dispatch Work

```bash
# Create a project rig
kubectl gt rig create my-project \
  --git-url https://github.com/org/repo.git \
  --prefix mp \
  -n gastown-system

# Sync your Claude credentials
kubectl gt auth sync -n gastown-system

# Dispatch a polecat to work on an issue
kubectl gt sling issue-123 my-project --name furiosa -n gastown-system

# Watch it work
kubectl gt polecat logs my-project/furiosa -f -n gastown-system
```

**For AI Agents:** See [AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md) for setup instructions.

---

## kubectl-gt CLI (New in v0.4.1)

**The recommended way to interact with Gas Town.** AI-native interface designed for both humans and agents.

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

**JSON/YAML Output** - All commands support `-o json` and `-o yaml` for machine parsing:

```bash
# Parse polecat list in scripts
kubectl gt polecat list -o json | jq '.[] | .metadata.name'

# Get rig status as YAML
kubectl gt rig status my-rig -o yaml
```

**Themed Naming** - Memorable names for your polecats:

```bash
# Named polecat
kubectl gt sling at-1234 athena --name furiosa

# Random themed name (mad-max, minerals, wasteland)
kubectl gt sling at-1234 athena --theme mad-max
# → Creates polecat named "capable" or "toast" etc.
```

**Wait for Ready** - Block until polecat pod is running:

```bash
# Wait for pod to be scheduled and ready
kubectl gt sling at-1234 athena --wait-ready --timeout 5m
```

**Native Log Streaming** - Stream logs directly without kubectl delegation:

```bash
# Follow logs
kubectl gt polecat logs athena/furiosa -f

# Specific container
kubectl gt polecat logs athena/furiosa -c claude -f
```

### Installation

**From Release:**
```bash
# macOS (Apple Silicon)
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt

# macOS (Intel)
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-darwin-amd64
chmod +x kubectl-gt-darwin-amd64 && sudo mv kubectl-gt-darwin-amd64 /usr/local/bin/kubectl-gt

# Linux (amd64)
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-linux-amd64
chmod +x kubectl-gt-linux-amd64 && sudo mv kubectl-gt-linux-amd64 /usr/local/bin/kubectl-gt
```

**From Source:**
```bash
make kubectl-gt-install
```

---

## YAML Templates (Alternative)

If you prefer declarative YAML over the CLI:

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-task
spec:
  rig: my-project
  desiredState: Working
  kubernetes:
    gitRepository: "git@github.com:org/repo.git"
    gitSecretRef:
      name: git-ssh-key
    claudeCredsSecretRef:
      name: claude-creds
  task:
    description: "Implement feature X"
```

See [templates/](templates/) for all resource examples.

## What You Get

| Feature | What It Means |
|---------|---------------|
| **Scale to 100+ agents** | Run 10, 50, or 100 polecats in parallel. No tmux limits. |
| **One-command install** | `helm install` and you're running in 30 seconds |
| **Instant agent startup** | Pre-built images with Claude Code. No npm install at runtime. |
| **Enterprise-ready** | OpenShift native, FIPS-compliant edition, security-scanned |
| **Agent-first docs** | Tell Claude to set it up. It reads the docs and configures itself. |
| **Full lifecycle management** | Health monitoring, auto-cleanup, merge queue, convoy tracking |
| **Multi-arch** | Runs on amd64 and arm64. Your infra, your choice. |
| **Supply chain security** | SBOM, Trivy scans, provenance attestations on every image |

## What Is This?

[Gas Town](https://github.com/steveyegge/gastown) runs AI agents (polecats) locally via tmux. This operator extends that to Kubernetes - **polecats run as pods instead of local processes**.

**Why Kubernetes?**
- **Horizontal scale**: Your laptop runs 4-8 agents. A cluster runs hundreds.
- **Zero infrastructure**: No tmux, no screen, no SSH. Just pods.
- **Built-in resilience**: Kubernetes restarts failed agents automatically.
- **Resource isolation**: Each polecat gets dedicated CPU/memory.

Supports **Claude Code** agents running as Kubernetes pods.

### Terminology

New to Gas Town? Here's the jargon:

| Term | What It Is | Required? |
|------|------------|-----------|
| **Polecat** | An AI worker pod that executes tasks | Yes |
| **Rig** | A project workspace (cluster-scoped) | Yes |
| **Convoy** | A batch of tasks for parallel execution | Optional |
| **Witness** | Health monitor for polecats | Optional |
| **Refinery** | Merge queue processor | Optional |
| **Beads** | Git-based issue tracker ([separate project](https://github.com/steveyegge/beads)) | Optional |

### Standalone Mode

You can use this operator **without** the full Gas Town ecosystem. Just provide a task description:

```yaml
spec:
  taskDescription: "Implement feature X"  # No beadID needed
  kubernetes:
    gitRepository: "git@github.com:org/repo.git"
    # ...
```

The operator works standalone with just Kubernetes + Claude Code credentials.

## How It Works

The operator is **CRD-driven**. You create a Polecat custom resource, and the operator handles the rest.

### Happy Path (Kubernetes Mode)

```
You                          Kubernetes                      Git
 │                              │                              │
 ├─ kubectl apply polecat.yaml ─►                              │
 │   (executionMode: kubernetes) │                              │
 │                              │                              │
 │                    Operator creates Pod                     │
 │                              │                              │
 │                    ┌─────────▼─────────┐                    │
 │                    │   Polecat Pod     │                    │
 │                    │  ┌─────────────┐  │                    │
 │                    │  │ Claude Code │  │                    │
 │                    │  │  - clones   │──┼── git clone ───────►
 │                    │  │  - works    │  │                    │
 │                    │  │  - commits  │──┼── git push ────────►
 │                    │  └─────────────┘  │                    │
 │                    └───────────────────┘                    │
 │                              │                              │
 ◄── kubectl get polecat ───────┤                              │
     (status: Done)             │                              │
```

**Step by step:**

1. **Create secrets** for git SSH key and Claude credentials
2. **Apply a Polecat CR** with your task:
   ```yaml
   apiVersion: gastown.gastown.io/v1alpha1
   kind: Polecat
   metadata:
     name: my-task
   spec:
     rig: my-project
     executionMode: kubernetes
     desiredState: Working
     kubernetes:
       gitRepository: "git@github.com:org/repo.git"
       gitSecretRef:
         name: git-ssh-key
       claudeCredsSecretRef:
         name: claude-creds
     task:
       description: "Implement feature X"
   ```
3. **Operator creates a Pod** with Claude Code pre-installed
4. **Pod runs Claude**, which clones, implements, commits, and pushes
5. **Results appear in git** - the branch is pushed, PR created if configured
6. **Check status**: `kubectl get polecat my-task`

### What Goes Where

| Data | Location | How to Access |
|------|----------|---------------|
| Work progress | Pod logs | `kubectl logs polecat-my-task` |
| Final code | Git remote | Pushed to branch |
| Polecat status | CRD status | `kubectl get polecat -o yaml` |

## Installation Options

### Option 1: Helm (Recommended)

```bash
# Standard Kubernetes
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.1 \
  --namespace gastown-system \
  --create-namespace
```

### Option 2: Helm for OpenShift

OpenShift requires stricter security settings:

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.1 \
  --namespace gastown-system \
  --create-namespace \
  --set securityContext.allowPrivilegeEscalation=false \
  --set securityContext.runAsNonRoot=true \
  --set securityContext.runAsUser=null \
  --set securityContext.readOnlyRootFilesystem=true \
  --set volumes.enabled=false
```

Or use the FIPS-compliant image for regulated environments:

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.1 \
  --namespace gastown-system \
  --create-namespace \
  --set image.tag=0.4.1-fips \
  --set securityContext.allowPrivilegeEscalation=false \
  --set securityContext.runAsNonRoot=true \
  --set securityContext.runAsUser=null \
  --set securityContext.readOnlyRootFilesystem=true \
  --set volumes.enabled=false
```

### Option 3: From Source

```bash
make install      # Install CRDs
make deploy IMG=ghcr.io/boshu2/gastown-operator:0.4.1
```

## Custom Resources

| CRD | Description |
|-----|-------------|
| **Rig** | Project workspace (cluster-scoped) |
| **Polecat** | Autonomous worker agent pod |
| **Convoy** | Batch tracking for parallel execution |
| **Refinery** | Merge queue processor |
| **Witness** | Worker lifecycle monitor |
| **BeadStore** | Issue tracking backend |

## Templates

Copy-paste YAML templates for all resources:

| Template | Purpose |
|----------|---------|
| [polecat-minimal.yaml](templates/polecat-minimal.yaml) | Simple polecat example |
| [polecat-kubernetes.yaml](templates/polecat-kubernetes.yaml) | Full polecat with all options |
| [convoy.yaml](templates/convoy.yaml) | Batch tracking |
| [witness.yaml](templates/witness.yaml) | Health monitoring |
| [refinery.yaml](templates/refinery.yaml) | Merge processing |
| [secret-git-ssh.yaml](templates/secret-git-ssh.yaml) | Git SSH credentials |
| [secret-claude-creds.yaml](templates/secret-claude-creds.yaml) | Claude API credentials |

Validate before applying:

```bash
./scripts/validate-template.sh templates/polecat-kubernetes.yaml
```

See [FRICTION_POINTS.md](FRICTION_POINTS.md) for common mistakes and fixes.

## Configuration

### Helm Values

| Parameter | Default | Description |
|-----------|---------|-------------|
| `image.repository` | `ghcr.io/boshu2/gastown-operator` | Container image |
| `image.tag` | `0.4.1` | Image tag |
| `replicaCount` | `1` | Number of replicas |

See [values.yaml](helm/gastown-operator/values.yaml) for full configuration.

## Architecture

```
┌────────────────────────────────────────────────────────┐
│                  Kubernetes Cluster                     │
│                                                         │
│   ┌─────────────────────────────────────────────────┐  │
│   │              gastown-operator                    │  │
│   │                                                  │  │
│   │   Rig       Polecat     Convoy     Witness      │  │
│   │ Controller  Controller  Controller  Controller  │  │
│   └──────────────────┬──────────────────────────────┘  │
│                      │                                  │
│          ┌───────────┴───────────┐                     │
│          ▼                       ▼                     │
│   ┌─────────────┐         ┌─────────────┐             │
│   │   Polecat   │   ...   │   Polecat   │             │
│   │    (Pod)    │         │    (Pod)    │             │
│   │  ┌───────┐  │         │  ┌───────┐  │             │
│   │  │Claude │  │         │  │Claude │  │  ───► Git   │
│   │  │ Code  │  │         │  │ Code  │  │             │
│   │  └───────┘  │         │  └───────┘  │             │
│   └─────────────┘         └─────────────┘             │
└────────────────────────────────────────────────────────┘
```

The operator manages the full polecat lifecycle. Kubernetes handles scheduling, scaling, and restarts.

## Container Images

All images are published to GHCR with SBOM, Trivy scans, and provenance attestations.

| Image | Purpose | Tags |
|-------|---------|------|
| `ghcr.io/boshu2/gastown-operator` | Kubernetes operator | `0.4.1`, `latest`, `0.4.1-fips` |
| `ghcr.io/boshu2/polecat-agent` | Pre-built polecat agent | `0.4.1`, `latest` |
| `ghcr.io/boshu2/charts/gastown-operator` | Helm chart (OCI) | `0.4.1` |

### Polecat Agent Image

The `polecat-agent` image comes with Claude Code pre-installed:

```bash
docker pull ghcr.io/boshu2/polecat-agent:0.4.1
```

**Benefits:**
- Instant startup (no runtime npm install)
- Pinned, documented component versions
- Security-scanned (Trivy) with SBOM
- No external network dependencies at runtime

**Included components:**
| Component | Version |
|-----------|---------|
| Claude Code | 2.0.22 (native binary) |
| gt CLI | latest |
| git, ssh, jq | system |

See [images/polecat-agent/](images/polecat-agent/) for build details and [CUSTOMIZING.md](images/polecat-agent/CUSTOMIZING.md) to extend the image with your own tools.

## Requirements

- Kubernetes 1.26+ or OpenShift 4.13+
- Helm 3.8+
- Git SSH credentials (for polecat git operations)
- LLM API credentials (Anthropic, OpenAI, or Ollama)

## Editions

| | **Community** | **Enterprise (FIPS)** |
|---|---|---|
| **Target** | Vanilla Kubernetes | OpenShift / Regulated |
| **Base Image** | `distroless` | Red Hat UBI9 |
| **Crypto** | Standard Go | FIPS-validated (BoringCrypto) |
| **Image Tag** | `0.4.1` | `0.4.1-fips` |

## Related Projects

- [Gas Town](https://github.com/steveyegge/gastown) - The multi-agent orchestration framework
- [Claude Code](https://github.com/anthropics/claude-code) - AI coding agent from Anthropic
- [Beads](https://github.com/steveyegge/beads) - Git-based issue tracking

## Contributing

PRs welcome. Please:
1. Run `make validate` before pushing
2. Add tests for new controllers
3. Keep the Mad Max references tasteful

See [CONTRIBUTING.md](CONTRIBUTING.md) for details.

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for version history.

## License

Apache 2.0. See [LICENSE](LICENSE).

---

*Built with mass production*
