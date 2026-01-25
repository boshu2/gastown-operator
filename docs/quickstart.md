# Quick Start

Get the Gas Town Operator running in minutes.

## Prerequisites

- Kubernetes 1.26+ or OpenShift 4.13+
- Helm 3.8+
- `kubectl`/`oc` configured with cluster access
- Git SSH key for repository access
- Claude API key or OAuth credentials

## Namespace Strategy

The Gas Town Operator uses a 3-namespace architecture for separation of concerns:

| Namespace | Purpose | Resources |
|-----------|---------|-----------|
| `gastown-system` | Control plane | Operator deployment, controller-manager |
| `gastown-workers` | Workloads | Polecat pods, Secrets |
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
- **Polecat CRs** → `gastown-workers`
- **Convoy CRs** → `gastown-workers`
- **Witness/Refinery CRs** → `gastown-system`
- **Secrets (git, claude)** → Same namespace as the Polecats referencing them
- **Rig CRs** → No namespace (cluster-scoped)

## Installation

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace
```

For OpenShift with restricted SCC:

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace \
  --set securityContext.allowPrivilegeEscalation=false \
  --set securityContext.runAsNonRoot=true \
  --set securityContext.runAsUser=null \
  --set securityContext.readOnlyRootFilesystem=true
```

## Verify Installation

```bash
kubectl get pods -n gastown-system
kubectl get crds | grep gastown
```

You should see:
- `gastown-operator-controller-manager` pod running
- Six CRDs: `rigs`, `polecats`, `convoys`, `witnesses`, `refineries`, `beadstores`

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

Polecats are workers that execute beads issues. They run Claude Code as Kubernetes pods.

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
  kubernetes:
    gitRepository: "git@github.com:myorg/myproject.git"
    gitBranch: main
    gitSecretRef:
      name: git-creds
    claudeCredsSecretRef:
      name: claude-creds
```

```bash
kubectl apply -f worker-polecat.yaml
```

The operator will create a Pod that clones the repo and runs Claude Code on the bead.

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
  rigRef: myproject
  notifyOnComplete: "mayor"  # mail address for completion notification
```

```bash
kubectl apply -f wave-1-convoy.yaml
```

## Next Steps

- [CRD Reference](./CRD_REFERENCE.md) - Full spec/status documentation
- [Architecture](./architecture.md) - How the operator works
- [Development](./development.md) - Contributing and local setup
