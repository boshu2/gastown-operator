# Gas Town Operator

> *"Who runs Bartertown? The Kubernetes Operator runs Bartertown."*

[![OpenShift](https://img.shields.io/badge/OpenShift-Native-EE0000?logo=redhatopenshift)](https://www.redhat.com/en/technologies/cloud-computing/openshift)
[![FIPS](https://img.shields.io/badge/FIPS-Compliant-blue)](https://csrc.nist.gov/projects/cryptographic-module-validation-program)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)

Kubernetes operator for [Gas Town](https://github.com/steveyegge/gastown) multi-agent orchestration. Run your AI agent army in the cloud.

```
     ____           _____
    / ___| __ _ ___| ____|_ ____      ___ __
   | |  _ / _` / __||  _| \ \/ / \ /\ / / '_ \
   | |_| | (_| \__ \| |___ >  <|  V  V /| | | |
    \____|\__,_|___/|_____/_/\_\\_/\_/ |_| |_|
                                    OPERATOR
```

## What Is This?

Gas Town Operator brings [steveyegge's Gas Town](https://github.com/steveyegge/gastown) multi-agent orchestration to Kubernetes. Instead of running polecats (autonomous AI workers) on your laptop, run them as pods in your cluster.

**Why?**
- Scale beyond your laptop's tmux sessions
- Let Kubernetes handle scheduling and lifecycle
- Run polecats closer to your infrastructure
- Enterprise-grade security (OpenShift SCCs, FIPS crypto)

## Design Philosophy

### OpenShift-Native

We don't just "support" OpenShift - we're built for it. Every pod runs with:

```yaml
securityContext:
  runAsNonRoot: true
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

No privileged containers. No security compromises. Passes `restricted` SCC out of the box.

### FIPS-Compliant

Built with Go's BoringCrypto on Red Hat UBI9:

- **Build**: `registry.access.redhat.com/ubi9/go-toolset:1.22`
- **Runtime**: `registry.access.redhat.com/ubi9/ubi-micro:9.3`
- **Crypto**: `GOEXPERIMENT=boringcrypto`

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
apiVersion: gastown.io/v1alpha1
kind: Rig
metadata:
  name: my-project
spec:
  gitURL: "git@github.com:myorg/my-project.git"
  beadsPrefix: "proj"
EOF

# Spawn a Polecat
kubectl apply -f - <<EOF
apiVersion: gastown.io/v1alpha1
kind: Polecat
metadata:
  name: furiosa
  namespace: gastown-workers
spec:
  rig: my-project
  beadID: proj-abc123
  kubernetes:
    gitRepository: "git@github.com:myorg/my-project.git"
    gitBranch: main
    gitSecretRef:
      name: git-ssh-key
    claudeCredsSecretRef:
      name: claude-api-key
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
make deploy IMG=ghcr.io/boshu2/gastown-operator:v0.1.0
```

## Requirements

- Kubernetes 1.26+ (OpenShift 4.13+ recommended)
- `gt` CLI accessible to operator (for local mode)
- Git SSH credentials (for polecat git operations)
- Claude API credentials (for polecat AI operations)

## Related Projects

- [Gas Town](https://github.com/steveyegge/gastown) - The multi-agent orchestration framework
- [gastown-gui](https://github.com/web3dev1337/gastown-gui) - Web UI dashboard (we're integrating!)
- [Beads](https://github.com/steveyegge/beads) - Git-based issue tracking

## Status

**v0.1.0** - Early release. Core CRDs and controllers working. Deployment proof coming soon.

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
