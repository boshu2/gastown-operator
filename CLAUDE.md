# gastown-operator

Kubernetes operator that runs Gas Town AI coding agents (polecats) as pods.

## Identity

| Attribute | Value |
|-----------|-------|
| **Name** | gastown-operator |
| **Version** | See `VERSION` file |
| **Repository** | boshu2/gastown-operator |
| **Language** | Go |
| **Framework** | controller-runtime (Kubebuilder) |

## Purpose

Extends [Gas Town](https://github.com/steveyegge/gastown) to Kubernetes. Instead of running polecats locally via tmux, they run as pods in a cluster, enabling horizontal scaling to 50+ parallel AI workers.

## Architecture

### Core Principle: CRDs as Views

The operator treats Kubernetes CRDs as **views into Gas Town's truth**, not as the source of truth itself.

```
┌─────────────────────────────────────────────────┐
│           Kubernetes Cluster                     │
│  ┌─────────┐  ┌─────────┐  ┌─────────┐         │
│  │   Rig   │  │ Polecat │  │ Convoy  │  ← CRDs │
│  └────┬────┘  └────┬────┘  └────┬────┘         │
│       └───────────┬────────────┘               │
│            ┌──────┴──────┐                     │
│            │  Operator   │                     │
│            │ Controllers │                     │
│            └─────────────┘                     │
└─────────────────────────────────────────────────┘
```

### Custom Resources

| CR | Scope | Purpose | Creates Pods? |
|----|-------|---------|---------------|
| **Rig** | Cluster | Project workspace, auto-creates Witness + Refinery | No |
| **Polecat** | Namespace | AI worker, runs Claude Code in a pod | **Yes** |
| **Convoy** | Namespace | Batch tracking for parallel execution | No |
| **Witness** | Namespace | Health monitoring, stuck detection | No |
| **Refinery** | Namespace | Merge queue processor | No |
| **BeadStore** | Namespace | Issue tracking backend config | No |

### Controllers

| Controller | File | Purpose |
|------------|------|---------|
| Rig | `internal/controller/rig_controller.go` | Aggregates status, auto-provisions Witness/Refinery |
| Polecat | `internal/controller/polecat_controller.go` | State machine for pod lifecycle |
| Convoy | `internal/controller/convoy_controller.go` | Tracks batch execution progress |
| Witness | `internal/controller/witness_controller.go` | Health checks, stuck detection, escalation |
| Refinery | `internal/controller/refinery_controller.go` | Git merge workflow (rebase + test + push) |
| BeadStore | `internal/controller/beadstore_controller.go` | Issue tracking configuration |

## Code Organization

```
gastown-operator/
├── api/v1alpha1/           # CRD types (spec/status definitions)
├── cmd/
│   ├── main.go             # Operator entrypoint
│   └── kubectl-gt/         # kubectl plugin CLI
├── internal/
│   ├── controller/         # Controller implementations
│   └── git/                # Git operations for Refinery
├── pkg/
│   ├── pod/                # Pod builder for polecats
│   ├── errors/             # Error handling
│   ├── metrics/            # Prometheus metrics
│   └── health/             # Health endpoints
├── config/                 # Kustomize manifests
├── helm/                   # Helm chart
├── images/polecat-agent/   # Pre-built polecat container
├── templates/              # YAML templates for users
└── deploy/tekton/          # CI/CD pipelines
```

## Key Design Decisions

### State vs Phase Naming

Deliberate naming to distinguish user intent from system reality:

- **State** (in spec): What the user WANTS (`DesiredState: Working`)
- **Phase** (in status): What the system OBSERVES (`Phase: Stuck`)

### Condition Backward Compatibility

Controllers set BOTH old and new conditions:
- Old: `Ready`, `Working` (deprecated)
- New: `Available`, `Progressing`, `Degraded` (standard)

Witness and Refinery check new conditions first, fall back to old.

### Kubernetes-Only Execution

Local/tmux execution mode was removed. The operator only supports Kubernetes execution. This simplifies the codebase and improves security (no host access needed).

## Build & Test

```bash
# Build
make build                  # Build operator binary
make docker-build           # Build container image
make kubectl-gt-install     # Install kubectl plugin

# Test
make test                   # Unit tests
make test-e2e               # E2E tests (requires kind)

# Generate
make manifests              # Regenerate CRDs
make generate               # Regenerate DeepCopy methods

# Validate
make fmt                    # Format code
make vet                    # Run go vet
make lint                   # Run linter (if configured)
```

## Common Operations

### Adding a New CRD Field

1. Edit type definition in `api/v1alpha1/<type>_types.go`
2. Run `make generate` to update DeepCopy methods
3. Run `make manifests` to regenerate CRD YAML
4. Update controller logic in `internal/controller/<type>_controller.go`
5. Add tests in `internal/controller/<type>_controller_test.go`

### Modifying a Controller

1. Edit controller in `internal/controller/<name>_controller.go`
2. Run unit tests: `go test ./internal/controller/... -run <TestName>`
3. Check RBAC markers (`+kubebuilder:rbac`) if permissions change
4. Run `make manifests` if RBAC changed

### Adding kubectl-gt Subcommand

1. Create new file in `cmd/kubectl-gt/cmd/<command>.go`
2. Register in `cmd/kubectl-gt/cmd/root.go`
3. Add tests in `cmd/kubectl-gt/cmd/<command>_test.go`
4. Update `cmd/kubectl-gt/README.md`

## Testing

### Unit Tests

```bash
make test                   # All unit tests
go test ./internal/controller/... -v  # Controller tests only
go test ./pkg/... -v        # Package tests only
```

### E2E Tests

```bash
make test-e2e               # Full E2E (creates kind cluster)
```

E2E tests create a kind cluster, deploy the operator, and verify CRD behavior.

### Test Patterns

- Controllers use envtest (fake K8s API server)
- Git operations use interface for test injection
- Pod builder has dedicated test suite

## Documentation

| Document | Purpose |
|----------|---------|
| `README.md` | User-facing docs, CLI reference |
| `AGENT_INSTRUCTIONS.md` | AI agent setup guide |
| `FRICTION_POINTS.md` | Common mistakes |
| `docs/architecture.md` | Internal architecture |
| `docs/CRD_REFERENCE.md` | CRD field reference |
| `cmd/kubectl-gt/README.md` | kubectl plugin docs |

## Issue Tracking

Uses [Beads](https://github.com/steveyegge/beads) for git-based issue tracking.

```bash
bd ready                  # Unblocked issues
bd show <id>              # Full context
bd sync && git push       # ALWAYS before stopping
```

## Related Projects

- [Gas Town](https://github.com/steveyegge/gastown) - Local execution (gt CLI)
- [Beads](https://github.com/steveyegge/beads) - Issue tracking (bd CLI)
- [Claude Code](https://github.com/anthropics/claude-code) - AI agent
