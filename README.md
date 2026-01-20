# Kubernetes Operator for Gas Town

> *"Who runs Bartertown? Kubernetes runs Bartertown."*

[![OpenShift](https://img.shields.io/badge/OpenShift-Native-EE0000?logo=redhatopenshift)](https://www.redhat.com/en/technologies/cloud-computing/openshift)
[![FIPS](https://img.shields.io/badge/FIPS-Compliant-blue)](https://csrc.nist.gov/projects/cryptographic-module-validation-program)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

A Kubernetes operator that runs [Gas Town](https://github.com/steveyegge/gastown) polecats as pods. Scale your AI agent army beyond the laptop.

Supports multiple coding agents: **opencode** (default), **claude-code**, **aider**, or **custom**.

```
    ╔═══════════════════════════════════════╗
    ║   GAS TOWN + KUBERNETES = SCALE       ║
    ║                                       ║
    ║   Polecats in pods. Convoys in CRDs.  ║
    ║   Same gt CLI. Cloud-native runtime.  ║
    ╚═══════════════════════════════════════╝
```

## What Is This?

[Gas Town](https://github.com/steveyegge/gastown) is a multi-agent orchestration framework - it runs AI agents (polecats) locally via tmux. This operator extends Gas Town to Kubernetes, so polecats run as pods instead of local processes.

**Why?**
- Scale beyond your laptop's tmux sessions
- Let Kubernetes handle scheduling and lifecycle
- Run polecats closer to your infrastructure

## Two Editions

We provide **two build profiles** - use what fits your environment:

| | **Community Edition** | **Enterprise Edition** |
|---|---|---|
| **Target** | Vanilla Kubernetes | OpenShift / Regulated environments |
| **Base Image** | `golang:alpine` / `distroless` | Red Hat UBI9 |
| **Crypto** | Standard Go | FIPS-validated (BoringCrypto) |
| **Security** | Standard PSS | Restricted SCC compliant |
| **Image Tag** | `:latest`, `:v0.1.2` | `:latest-fips`, `:v0.1.2-fips` |

### Community Edition (Vanilla K8s)

Lightweight, runs anywhere:

```bash
# Standard Kubernetes
kubectl apply -f https://github.com/boshu2/gastown-operator/releases/download/v0.1.2/install.yaml
```

### Enterprise Edition (OpenShift + FIPS)

For regulated environments (FedRAMP, HIPAA, government):

```bash
# OpenShift with FIPS
oc apply -f https://github.com/boshu2/gastown-operator/releases/download/v0.1.2/install-fips.yaml
```

**What makes it enterprise-ready:**

```yaml
# Every pod runs with restricted SCC compliance
securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

**FIPS-compliant build:**
- `registry.access.redhat.com/ubi9/go-toolset:1.22` (build)
- `registry.access.redhat.com/ubi9/ubi-micro:9.3` (runtime)
- `GOEXPERIMENT=boringcrypto` (FIPS-validated crypto)

For when your compliance officer asks "but is it FIPS?"

## Custom Resources

| CRD | Description |
|-----|-------------|
| **Rig** | Project workspace (cluster-scoped) |
| **Polecat** | Autonomous worker agent pod |
| **Convoy** | Batch tracking for parallel execution |
| **Refinery** | Merge queue processor |
| **Witness** | Worker lifecycle monitor |
| **BeadStore** | Issue tracking backend |

## Quick Start

```bash
# Install CRDs
kubectl apply -f config/crd/bases/

# Run operator
make run-local

# Create a Rig
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: my-project
spec:
  gitURL: "git@github.com:myorg/my-project.git"
  beadsPrefix: "proj"
  localPath: "/home/user/workspaces/my-project"
EOF

# Spawn a Polecat (opencode - default)
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
  executionMode: kubernetes
  # agent: opencode  # default
  agentConfig:
    provider: litellm
    model: claude-sonnet-4
    modelProvider:
      apiKeySecretRef:
        name: litellm-api-key
        key: api-key
  kubernetes:
    gitRepository: "git@github.com:myorg/my-project.git"
    gitBranch: main
    gitSecretRef:
      name: git-ssh-key
EOF

# Watch it work
kubectl logs -f polecat-furiosa -n gastown-workers
```

## Architecture

```
┌────────────────────────────────────────────────────────┐
│                  OpenShift Cluster                      │
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

## Installation

### Helm (Recommended)

```bash
helm repo add gastown https://boshu2.github.io/gastown-operator
helm install gastown-operator gastown/gastown-operator \
  --namespace gastown-system \
  --create-namespace
```

### From Source

```bash
make install      # Install CRDs
make deploy IMG=ghcr.io/boshu2/gastown-operator:v0.1.2
```

## Requirements

- Kubernetes 1.26+ (OpenShift 4.13+ recommended)
- `gt` CLI accessible to operator (for local mode)
- Git SSH credentials (for polecat git operations)
- LLM API credentials (LiteLLM, Anthropic, OpenAI, or Ollama)

## Related Projects

- [Gas Town](https://github.com/steveyegge/gastown) - The multi-agent orchestration framework
- [opencode](https://github.com/opencode-ai/opencode) - Open-source coding agent (default)
- [gastown-gui](https://github.com/web3dev1337/gastown-gui) - Web UI dashboard (we're integrating!)
- [Beads](https://github.com/steveyegge/beads) - Git-based issue tracking

## Status

**v0.1.2** - E2E validated. Polecat pods successfully run AI coding agents in OpenShift. Full lifecycle tested: CR creation → pod spawn → git clone → agent execution → completion. Supports opencode (default), claude-code, aider, and custom agents.

Feedback welcome! See [steveyegge/gastown#668](https://github.com/steveyegge/gastown/issues/668) for discussion.

## Contributing

PRs welcome. Please:
1. Run `make validate` before pushing
2. Add tests for new controllers
3. Keep the Mad Max references tasteful

## License

Apache 2.0. See [LICENSE](LICENSE).

---

*Built with mass production*
