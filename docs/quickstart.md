# Quick Start

Get the Gas Town Operator running in minutes.

## Prerequisites

- Kubernetes cluster (1.26+)
- Helm 3.x
- `kubectl` configured with cluster access
- `gt` CLI installed and configured
- A Gas Town setup (`~/gt/` with rigs)

## Namespace Strategy

The Gas Town Operator uses a 3-namespace architecture for separation of concerns:

| Namespace | Purpose | Resources |
|-----------|---------|-----------|
| `gastown-system` | Control plane | Operator deployment, controller-manager |
| `gastown-workers` | Workloads | Polecat pods (kubernetes mode), Secrets |
| Cluster-scoped | Global resources | Rig CRDs |

### Why This Architecture?

1. **Security isolation**: Workloads run separately from control plane
2. **RBAC scoping**: Different permissions for operators vs workers
3. **Resource quotas**: Apply limits per namespace for workloads
4. **Network policies**: Restrict egress differently per namespace

### Namespace Setup

```bash
# Create namespaces
kubectl create namespace gastown-system
kubectl create namespace gastown-workers

# Apply network policies (optional but recommended)
kubectl label namespace gastown-system pod-security.kubernetes.io/enforce=restricted
kubectl label namespace gastown-workers pod-security.kubernetes.io/enforce=baseline
```

### Which Namespace for What?

- **Operator install** → `gastown-system`
- **Polecat CRs** → `gastown-workers` (for kubernetes mode)
- **Convoy CRs** → `gastown-workers`
- **Witness/Refinery CRs** → `gastown-system`
- **Secrets (git, claude)** → Same namespace as the Polecats referencing them
- **Rig CRs** → No namespace (cluster-scoped)

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
  namespace: gastown-workers
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
  namespace: gastown-workers
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
