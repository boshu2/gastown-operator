<div align="center">

```
  ____           _____                    ___                       _
 / ___| __ _ ___  |_   _|____      ___ __  / _ \ _ __   ___ _ __ __ _| |_ ___  _ __
| |  _ / _` / __|   | |/ _ \ \ /\ / / '_ \| | | | '_ \ / _ \ '__/ _` | __/ _ \| '__|
| |_| | (_| \__ \   | | (_) \ V  V /| | | | |_| | |_) |  __/ | | (_| | || (_) | |
 \____|\__,_|___/   |_|\___/ \_/\_/ |_| |_|\___/| .__/ \___|_|  \__,_|\__\___/|_|
                                                |_|
```

# Kubernetes Operator for Gas Town

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
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace
```

Then tell Claude:

> "Set up gastown-operator on my cluster. Read [AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md)."

Claude will handle the secrets, the Polecat CRs, everything. You don't write YAML - the agents do.

**For AI Agents:** See [.claude/SKILL.md](.claude/SKILL.md) for copy-paste templates and [templates/](templates/) for all resource examples.

## What Is This?

[Gas Town](https://github.com/steveyegge/gastown) runs AI agents (polecats) locally via tmux. This operator extends that to Kubernetes - **polecats run as pods instead of local processes**.

**The workflow:**
1. Install the operator (above)
2. Tell Claude to set it up using the docs
3. Sling work: `gt sling issue-123 my-rig --mode kubernetes`
4. Watch: `gt convoy list` or `kubectl logs -f polecat-issue-123`

**Why?**
- Scale beyond your laptop's tmux sessions
- Run agents closer to your infrastructure
- Parallel execution across a cluster
- Kubernetes handles scheduling and lifecycle

Supports: **claude-code** (default), **opencode**, **aider**, or **custom** agents.

## Installation Options

### Option 1: Helm (Recommended)

```bash
# Standard Kubernetes
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace
```

### Option 2: Helm for OpenShift

OpenShift requires stricter security settings:

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
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
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace \
  --set image.tag=0.3.2-fips \
  --set securityContext.allowPrivilegeEscalation=false \
  --set securityContext.runAsNonRoot=true \
  --set securityContext.runAsUser=null \
  --set securityContext.readOnlyRootFilesystem=true \
  --set volumes.enabled=false
```

### Option 3: From Source

```bash
make install      # Install CRDs
make deploy IMG=ghcr.io/boshu2/gastown-operator:0.3.2
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
| `image.tag` | `0.3.2` | Image tag |
| `replicaCount` | `1` | Number of replicas |
| `volumes.enabled` | `true` | Mount host path for gt CLI |
| `volumes.hostPath` | `/home/core/gt` | Path to Gas Town on host |

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
| **Image Tag** | `0.3.2` | `0.3.2-fips` |

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
