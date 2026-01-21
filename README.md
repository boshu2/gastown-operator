<div align="center">

# Kubernetes Operator for Gas Town

```
                            .  .
                         .  |  |  .
                  .  .  . \|__|____|/ .  .  .
                   \| \|  .' .''. '.  |/ |/
                    \__|_/  / /\ \  \_|__/
              .      |  | |  \/  | |  |      .
         . \|/ .   .'|  |_|      |_|  |'.   . \|/ .
          \===/ .'   '.  __    __  .'   '. \===/
           |H|  |  .   \/  \  /  \/   .  |  |H|
           |H|  |  |\  |    \/    |  /|  |  |H|
          /===\ '.  \ \|          |/ /  .' /===\
         ' /|\ '  '._\  GASTOWN   /_.'  ' /|\ '
              '      | OPERATOR  |      '
                     |    ___    |            WITNESS ME!
                     |   [___]   |
          _.====._   '.   V8   .'   _.====._      SHINY AND
         [________]    '------'    [________]      CHROME!
             ||                        ||
         ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
              WHO RUNS BARTERTOWN? KUBERNETES RUNS BARTERTOWN.
         ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~
```

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
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace
```

### 2. Create Secrets

```bash
# Git SSH key for cloning/pushing
kubectl create secret generic git-credentials -n gastown-system \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519

# Claude API key
kubectl create secret generic claude-credentials -n gastown-system \
  --from-literal=api-key=$ANTHROPIC_API_KEY
```

### 3. Sling Work to Kubernetes

From your Mayor session, just sling issues like normal - but to k8s:

```bash
# The Mayor dispatches work to kubernetes polecats
gt sling proj-123 my-rig --mode kubernetes

# Claude figures out the Polecat CR, the operator creates the pod
# You don't write YAML - the agents handle it
```

That's it. The polecat runs as a pod, clones your repo, does the work, pushes, and exits.

### 4. Watch Progress

```bash
# Check convoy status (same as local polecats)
gt convoy list

# Or peek at the pod directly
kubectl logs -f polecat-proj-123 -n gastown-workers
```

## What Is This?

[Gas Town](https://github.com/steveyegge/gastown) runs AI agents (polecats) locally via tmux. This operator extends that to Kubernetes - polecats run as pods instead of local processes.

**The key insight:** You don't manually write Polecat CRs. The Mayor slings work with `gt sling`, Claude generates the appropriate Kubernetes resources, and the operator handles the rest.

**Why run polecats in k8s?**
- Scale beyond your laptop's tmux sessions
- Run agents closer to your infrastructure
- Let Kubernetes handle scheduling and lifecycle
- Parallel execution across a cluster

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
