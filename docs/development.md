# Development Guide

Setting up the Gas Town Operator for local development.

## Prerequisites

- Go 1.22+
- kubebuilder 3.x
- kubectl configured with cluster access
- gt CLI installed
- A Gas Town setup (`~/gt/` with at least one rig)

## Project Structure

```
gastown-operator/
├── api/v1alpha1/         # CRD type definitions
│   ├── rig_types.go
│   ├── polecat_types.go
│   └── convoy_types.go
├── cmd/
│   ├── manager/          # Production entrypoint
│   └── local/            # Local development entrypoint
├── config/
│   ├── crd/bases/        # Generated CRD manifests
│   ├── rbac/             # RBAC manifests
│   └── samples/          # Example CRs
├── internal/controller/  # Controller implementations
│   ├── rig_controller.go
│   ├── polecat_controller.go
│   ├── convoy_controller.go
│   └── beadssync_controller.go
├── pkg/
│   ├── gt/               # gt CLI wrapper
│   ├── errors/           # Error types
│   └── metrics/          # Prometheus metrics
├── helm/                 # Helm chart
└── docs/                 # Documentation
```

## First-Time Setup

### Install Pre-Push Hooks

The project uses pre-push hooks to validate code before pushing to remote:

```bash
make setup-hooks
```

This installs hooks that run `make validate` (lint + vet) before every push. This prevents CI failures by catching issues locally.

## Local Development

### 1. Install CRDs

```bash
make install
# or
kubectl apply -f config/crd/bases/
```

### 2. Run in Local Mode

Local mode connects to your cluster but runs controllers locally:

```bash
make run-local

# With custom paths:
make run-local GT_TOWN_ROOT=/path/to/gt GT_PATH=/path/to/gt
```

This:
- Uses your local `~/.kube/config`
- Connects controllers to your cluster
- Uses local gt CLI installation
- Enables leader election (disabled by default in local mode)

### 3. Create Test Resources

```bash
kubectl apply -f config/samples/gastown_v1alpha1_rig.yaml
kubectl apply -f config/samples/gastown_v1alpha1_polecat.yaml
kubectl apply -f config/samples/gastown_v1alpha1_convoy.yaml
```

## Running Tests

### Unit Tests

```bash
make test
```

### Integration Tests (envtest)

```bash
make test-integration

# Requires envtest binaries:
make envtest
```

### End-to-End Tests

```bash
# Start operator in local mode
make run-local &

# Run e2e tests
make test-e2e
```

## Adding a New CRD

### 1. Scaffold with kubebuilder

```bash
kubebuilder create api --group gastown --version v1alpha1 --kind NewResource
```

### 2. Define Types

Edit `api/v1alpha1/newresource_types.go`:

```go
type NewResourceSpec struct {
    // Add spec fields
}

type NewResourceStatus struct {
    Phase      string             `json:"phase,omitempty"`
    Conditions []metav1.Condition `json:"conditions,omitempty"`
}
```

### 3. Implement Controller

Edit `internal/controller/newresource_controller.go`:

```go
func (r *NewResourceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
    // 1. Fetch the resource
    // 2. Query gt CLI for current state
    // 3. Update status
    // 4. Return result
}
```

### 4. Register Controller

Add to `cmd/local/main.go` and `cmd/manager/main.go`:

```go
if err = (&controller.NewResourceReconciler{
    Client:   mgr.GetClient(),
    Scheme:   mgr.GetScheme(),
    GTClient: gtClient,
}).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "NewResource")
    os.Exit(1)
}
```

### 5. Regenerate Manifests

```bash
make manifests
make generate
```

## Code Style

### Controller Patterns

```go
// Always use context for gt CLI calls
status, err := r.GTClient.RigStatus(ctx, name)

// Return transient errors for retry
if errors.IsTransient(err) {
    return ctrl.Result{RequeueAfter: time.Second * 5}, nil
}

// Return permanent errors with condition update
if errors.IsPermanent(err) {
    meta.SetStatusCondition(&resource.Status.Conditions, metav1.Condition{
        Type:    "Ready",
        Status:  metav1.ConditionFalse,
        Reason:  "PermanentError",
        Message: err.Error(),
    })
    return ctrl.Result{}, nil // Don't requeue
}
```

### Metrics

Record metrics for observability:

```go
metrics.RecordReconcile("rig", time.Since(start), err)
metrics.RecordGTCLICall("rig_status", err)
```

### Error Handling

Use typed errors from `pkg/errors`:

```go
// Transient (will retry)
return errors.NewTransient("gt CLI timeout", err)

// Permanent (won't retry)
return errors.NewPermanent("invalid configuration", err)

// Validation
return errors.NewValidation("beadID is required when desiredState=Working")
```

## Debugging

### Check Controller Logs

```bash
# Local mode
make run-local 2>&1 | grep -E "(ERROR|INFO)"

# In-cluster
kubectl logs -n gastown-system deployment/gastown-operator-controller-manager
```

### Inspect CRD Status

```bash
kubectl get rigs -o yaml
kubectl describe polecat furiosa
kubectl get convoys -o jsonpath='{.items[*].status}'
```

### Force Reconcile

```bash
kubectl annotate rig fractal reconcile=$(date +%s) --overwrite
```

## Makefile Targets

| Target | Description |
|--------|-------------|
| `make setup-hooks` | Install git pre-push hooks |
| `make validate` | Run local validation (vet + lint) |
| `make build` | Build operator binary |
| `make build-local` | Build local mode binary |
| `make run` | Run operator (production mode) |
| `make run-local` | Run operator (local dev mode) |
| `make install` | Install CRDs to cluster |
| `make uninstall` | Remove CRDs from cluster |
| `make manifests` | Generate CRD manifests |
| `make generate` | Generate DeepCopy methods |
| `make test` | Run unit tests |
| `make lint` | Run golangci-lint |
| `make docker-build` | Build container image |
| `make docker-push` | Push container image |

## Troubleshooting

### CRD Not Found

```bash
# Reinstall CRDs
make install

# Check CRD status
kubectl get crds | grep gastown
```

### Controller Not Starting

```bash
# Check for port conflicts (metrics, health)
lsof -i :8080
lsof -i :8081

# Run with verbose logging
make run-local ARGS="--zap-log-level=debug"
```

### gt CLI Errors

```bash
# Verify gt CLI works
gt --version
gt rig list

# Check GT_PATH and GT_TOWN_ROOT
echo $GT_PATH
echo $GT_TOWN_ROOT
```
