# Agent Instructions for gastown-operator

This file provides context for AI coding assistants working on gastown-operator.

## Project Overview

gastown-operator is a Kubernetes operator that runs Gas Town polecats as pods.
It extends the [Gas Town](https://github.com/steveyegge/gastown) multi-agent
orchestration framework from local tmux sessions to Kubernetes.

## Key Concepts

| Term | Description |
|------|-------------|
| **Gas Town** | Multi-agent orchestration framework (upstream) |
| **Polecat** | Autonomous AI worker agent |
| **Rig** | Project workspace (cluster-scoped CRD) |
| **Convoy** | Batch tracking for parallel polecat execution |
| **Witness** | Worker lifecycle monitor |
| **Refinery** | Merge queue processor |
| **BeadStore** | Issue tracking backend |

## Repository Structure

```
gastown-operator/
├── api/v1alpha1/       # CRD type definitions
├── cmd/                # Entry points
├── config/
│   ├── crd/           # CustomResourceDefinitions
│   ├── rbac/          # Role-based access control
│   └── manager/       # Operator deployment
├── deploy/
│   └── tekton/        # CI pipeline definitions
├── docs/              # Documentation
├── internal/
│   └── controller/    # Reconciliation logic
└── pkg/               # Shared packages
```

## Development Commands

```bash
make build           # Build operator binary
make test            # Run unit tests
make lint            # Run linters
make manifests       # Generate CRDs from Go types
make install         # Install CRDs to cluster
make run             # Run operator locally
make docker-build    # Build container image
```

## Code Patterns

### Controller Reconciliation

Controllers follow the standard controller-runtime pattern:

```go
func (r *PolecatReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the resource
    // 2. Check if deleted (handle finalizer)
    // 3. Reconcile desired state
    // 4. Update status
    return ctrl.Result{}, nil
}
```

### CRD Modifications

When modifying CRDs in `api/v1alpha1/`:

1. Update the Go struct with kubebuilder markers
2. Run `make manifests` to regenerate YAML
3. Run `make install` to apply to cluster

### Testing

- Unit tests use envtest (etcd + kube-apiserver)
- Run `make test` before submitting PRs
- Add test cases for new controller logic

## Two Editions

| Edition | Base Image | Use Case |
|---------|------------|----------|
| Community | distroless | Vanilla Kubernetes |
| Enterprise | UBI9 + FIPS | OpenShift, regulated |

Build with `EDITION=fips make docker-build` for enterprise.

## Common Tasks

### Add a new CRD field

1. Edit `api/v1alpha1/<type>_types.go`
2. Add kubebuilder validation markers
3. Run `make manifests`
4. Update controller logic
5. Add tests

### Debug controller

```bash
# Run with debug logging
make run ARGS="--zap-log-level=debug"

# Check operator logs in cluster
kubectl logs -n gastown-system deployment/gastown-operator-controller-manager
```

## Dependencies

- controller-runtime v0.22.x
- kubebuilder v4
- Go 1.24+

## Related Documentation

- [Gas Town](https://github.com/steveyegge/gastown) - Upstream framework
- [Kubebuilder Book](https://book.kubebuilder.io/) - Operator patterns
- [Controller Runtime](https://pkg.go.dev/sigs.k8s.io/controller-runtime) - API docs
