# Quick Start

Get the Gas Town Operator running in minutes.

## Prerequisites

- Kubernetes cluster (1.26+)
- Helm 3.x
- `kubectl` configured with cluster access
- `gt` CLI installed and configured
- A Gas Town setup (`~/gt/` with rigs)

## Installation

### 1. Add the Helm repository (or install from local chart)

```bash
# From local chart
helm install gastown-operator ./helm/gastown-operator \
  --namespace gastown-system \
  --create-namespace
```

### 2. Verify installation

```bash
kubectl get pods -n gastown-system
kubectl get crds | grep gastown
```

You should see:
- `gastown-operator-controller-manager` pod running
- Three CRDs: `rigs.gastown.gastown.io`, `polecats.gastown.gastown.io`, `convoys.gastown.gastown.io`

## Create Your First Rig

### 1. Define a Rig resource

```yaml
# my-rig.yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: myproject
spec:
  gitURL: "git@github.com:myorg/myproject.git"
  beadsPrefix: "frac"
  localPath: "/home/user/workspaces/myproject"
  settings:
    namepoolTheme: "mad-max"
    maxPolecats: 5
```

### 2. Apply the Rig

```bash
kubectl apply -f my-rig.yaml
```

### 3. Check status

```bash
kubectl get rigs
kubectl describe rig myproject
```

The operator will:
1. Verify the local path exists
2. Query `gt rig status myproject` for current state
3. Update the Rig status with polecat count, active convoys, etc.

## Create a Polecat

Polecats are workers that execute beads issues.

```yaml
# worker-polecat.yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: worker-1
  namespace: default
spec:
  rig: myproject
  desiredState: Working
  beadID: "mp-abc-123"
```

```bash
kubectl apply -f worker-polecat.yaml
```

The operator will call `gt sling mp-abc-123 myproject` to spawn the polecat.

## Create a Convoy

Convoys track batches of beads for parallel execution.

```yaml
# wave-1-convoy.yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Convoy
metadata:
  name: wave-1
  namespace: default
spec:
  description: "Wave 1 implementation tasks"
  trackedBeads:
    - "mp-abc-123"
    - "mp-def-456"
    - "mp-ghi-789"
  notifyOnComplete: true
```

```bash
kubectl apply -f wave-1-convoy.yaml
```

## Next Steps

- [CRD Reference](./crds.md) - Full spec/status documentation
- [Architecture](./architecture.md) - How the operator works
- [Development](./development.md) - Contributing and local setup
