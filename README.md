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

```bash
# Add the helm chart and install
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace
```

That's it. The operator is now running.

### Verify Installation

```bash
kubectl get pods -n gastown-system
# NAME                                                   READY   STATUS
# gastown-operator-controller-manager-xxxxx-xxxxx        1/1     Running
```

### Create Your First Rig

```bash
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: my-project
spec:
  gitURL: "git@github.com:myorg/my-project.git"
  beadsPrefix: "proj"
EOF
```

### Spawn a Polecat

```bash
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: furiosa
  namespace: gastown-workers
spec:
  rig: my-project
  beadID: proj-abc123
  desiredState: Working
  kubernetes:
    gitRepository: "git@github.com:myorg/my-project.git"
    gitBranch: main
    gitSecretRef:
      name: git-ssh-key
    claudeCredsSecretRef:
      name: claude-creds
EOF

# Watch it work
kubectl logs -f polecat-furiosa -n gastown-workers
```

## What Is This?

[Gas Town](https://github.com/steveyegge/gastown) is a multi-agent orchestration framework - it runs AI agents (polecats) locally via tmux. This operator extends Gas Town to Kubernetes, so polecats run as pods instead of local processes.

**Why?**
- Scale beyond your laptop's tmux sessions
- Let Kubernetes handle scheduling and lifecycle
- Run polecats closer to your infrastructure

Supports multiple coding agents: **opencode** (default), **claude-code**, **aider**, or **custom**.

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
