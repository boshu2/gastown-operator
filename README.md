# Gas Town Operator

Kubernetes operator for Gas Town multi-agent orchestration primitives.

## Overview

The Gas Town Operator exposes Gas Town concepts as Kubernetes Custom Resources:

- **Rig** - Project workspace (cluster-scoped)
- **Polecat** - Autonomous worker agent
- **Convoy** - Batch tracking for parallel execution

The operator acts as a **view layer** - the `gt` CLI remains the source of truth for all operations.

## Quick Start

```bash
# Install CRDs
kubectl apply -f config/crd/bases/

# Run operator (local mode)
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
  localPath: "/Users/me/gt/my-project"
EOF

# Check status
kubectl get rigs
kubectl describe rig my-project
```

## Installation

### Helm

```bash
helm install gastown-operator ./helm/gastown-operator \
  --namespace gastown-system \
  --create-namespace \
  --set gtConfig.townRoot=/path/to/gt
```

### From Source

```bash
make install    # Install CRDs
make run        # Run operator
```

## Architecture

```
┌─────────────────────────────────────────┐
│              Kubernetes                  │
│  ┌─────┐  ┌─────────┐  ┌────────┐       │
│  │ Rig │  │ Polecat │  │ Convoy │       │
│  └──┬──┘  └────┬────┘  └───┬────┘       │
│     └──────────┼───────────┘            │
│                │                         │
│         ┌──────┴──────┐                  │
│         │  Operator   │                  │
│         └──────┬──────┘                  │
└────────────────┼────────────────────────┘
                 │
          ┌──────┴──────┐
          │   gt CLI    │  ← Source of Truth
          └─────────────┘
```

The operator queries `gt` CLI for state and executes commands through it. CRDs provide Kubernetes-native access to Gas Town without duplicating state.

## Documentation

- [Quick Start](docs/quickstart.md) - Get up and running
- [CRD Reference](docs/crds.md) - Full API documentation
- [Architecture](docs/architecture.md) - How it works
- [Development](docs/development.md) - Contributing guide

## Requirements

- Kubernetes 1.26+
- Go 1.22+ (for development)
- `gt` CLI installed on operator host
- Gas Town setup (`~/gt/` directory structure)

## Development

```bash
# Run locally
make run-local GT_TOWN_ROOT=~/gt GT_PATH=/usr/local/bin/gt

# Run tests
make test

# Build container
make docker-build IMG=myregistry/gastown-operator:latest
```

See [Development Guide](docs/development.md) for details.

## License

Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
