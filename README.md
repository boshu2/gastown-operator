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
# Set up pre-push hooks (validates before push)
make setup-hooks

# Run local validation (lint + vet)
make validate

# Run locally
make run-local GT_TOWN_ROOT=~/gt GT_PATH=/usr/local/bin/gt

# Run tests
make test

# Build container
make docker-build IMG=myregistry/gastown-operator:latest
```

The pre-push hook runs `make validate` automatically. Lint/complexity checks run locally, not in CI.

See [Development Guide](docs/development.md) for details.

## CI/CD

This project uses **Tekton Pipelines** for CI/CD. The pipeline runs on OpenShift in the `olympus-ci` namespace.

### Running the Pipeline

```bash
# Apply Tasks and Pipeline (first time or after changes)
oc apply -f deploy/tekton/tasks/ -n olympus-ci
oc apply -f deploy/tekton/pipeline.yaml -n olympus-ci

# Create a PipelineRun
oc create -f deploy/tekton/pipelinerun.yaml -n olympus-ci

# Watch progress
tkn pipelinerun logs -f --last -n olympus-ci
```

### Pipeline Stages

1. **Clone** - Fetch source from GitLab
2. **Parallel Stage**:
   - `go-test` - Unit tests with envtest
   - `scan-secrets` - Trivy filesystem scan (secrets, misconfigs)
   - `lint-dockerfile` - Hadolint Dockerfile linting
   - `build-gt-cli` - Build gt CLI from source
3. **Build** - Kaniko container build + push to DPR
4. **Scan** - Trivy image vulnerability scan + SBOM generation

### ClusterTasks Used

| Task | Purpose |
|------|---------|
| `git-clone` | Clone source repository |
| `jren-kaniko-build` | Build container image (no Docker daemon) |
| `jren-trivy-fs` | Filesystem security scan |
| `jren-hadolint-scan` | Dockerfile linting |
| `jren-trivy-image` | Container vulnerability scan + SBOM |

### Local Tasks

- `go-test-gastown` - Controller tests with envtest
- `build-gt-cli` - Build gt CLI from boshu2/gastown fork

See `deploy/tekton/` for full configuration.

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
