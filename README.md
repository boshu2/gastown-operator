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

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.0 \
  --namespace gastown-system \
  --create-namespace
```

Then tell Claude:

> "Set up gastown-operator on my cluster. Read [AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md)."

Claude will handle the secrets, the Polecat CRs, everything. You don't write YAML - the agents do.

**For AI Agents:** See [.claude/SKILL.md](.claude/SKILL.md) for copy-paste templates and [templates/](templates/) for all resource examples.

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

Supports: **claude-code** (default), **opencode**, **aider**, or **custom** agents.

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

### Local Mode (Alternative)

If the operator runs on a host with Gas Town installed, it can also manage local polecats:

```yaml
spec:
  executionMode: local  # Uses tmux on host instead of pods
```

In this mode, the operator calls `gt sling` on the host filesystem. This is useful for hybrid setups where you want K8s to orchestrate but execution stays local.

## Installation Options

### Option 1: Helm (Recommended)

```bash
# Standard Kubernetes
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.0 \
  --namespace gastown-system \
  --create-namespace
```

### Option 2: Helm for OpenShift

OpenShift requires stricter security settings:

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.0 \
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
  --version 0.4.0 \
  --namespace gastown-system \
  --create-namespace \
  --set image.tag=0.4.0-fips \
  --set securityContext.allowPrivilegeEscalation=false \
  --set securityContext.runAsNonRoot=true \
  --set securityContext.runAsUser=null \
  --set securityContext.readOnlyRootFilesystem=true \
  --set volumes.enabled=false
```

### Option 3: From Source

```bash
make install      # Install CRDs
make deploy IMG=ghcr.io/boshu2/gastown-operator:0.4.0
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
| [polecat-minimal.yaml](templates/polecat-minimal.yaml) | Quick local polecat (3 variables) |
| [polecat-kubernetes.yaml](templates/polecat-kubernetes.yaml) | Full K8s execution with all options |
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
| `image.tag` | `0.4.0` | Image tag |
| `replicaCount` | `1` | Number of replicas |
| `volumes.enabled` | `false` | Mount host path (for local execution mode only) |
| `volumes.hostPath` | `/home/core/gt` | Path to Gas Town on host |

**Note:** `volumes.enabled` defaults to `false` because most users use Kubernetes execution mode where polecats run as pods. Enable host volumes only for local execution mode:

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.0 \
  --namespace gastown-system \
  --create-namespace \
  --set volumes.enabled=true \
  --set volumes.hostPath=/path/to/your/gt
```

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
│   │  │Claude │  │         │  │Claude │  │             │
│   │  │ Code  │  │         │  │ Code  │  │             │
│   │  └───────┘  │         │  └───────┘  │             │
│   └─────────────┘         └─────────────┘             │
└────────────────────────────────────────────────────────┘
          │
          │ Claims work via webhook
          ▼
┌────────────────────────────────────────────────────────┐
│              Local Gas Town (gt CLI)                    │
│  - Source of truth for state                           │
│  - Beads, mail, molecules                              │
└────────────────────────────────────────────────────────┘
```

The operator is a **view layer** - `gt` CLI remains authoritative. Kubernetes handles scheduling, scaling, and lifecycle.

## Container Images

All images are published to GHCR with SBOM, Trivy scans, and provenance attestations.

| Image | Purpose | Tags |
|-------|---------|------|
| `ghcr.io/boshu2/gastown-operator` | Kubernetes operator | `0.4.0`, `latest`, `0.4.0-fips` |
| `ghcr.io/boshu2/polecat-agent` | Pre-built polecat agent | `0.4.0`, `latest` |
| `ghcr.io/boshu2/charts/gastown-operator` | Helm chart (OCI) | `0.4.0` |

### Polecat Agent Image

The `polecat-agent` image comes with Claude Code pre-installed:

```bash
docker pull ghcr.io/boshu2/polecat-agent:0.4.0
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
| **Image Tag** | `0.4.0` | `0.4.0-fips` |

## Related Projects

- [Gas Town](https://github.com/steveyegge/gastown) - The multi-agent orchestration framework
- [opencode](https://github.com/opencode-ai/opencode) - Open-source coding agent (default)
- [Beads](https://github.com/steveyegge/beads) - Git-based issue tracking

## Contributing

PRs welcome. Please:
1. Run `make validate` before pushing
2. Add tests for new controllers
3. Keep the Mad Max references tasteful

## License

Apache 2.0. See [LICENSE](LICENSE).

---

*Built with mass production*
