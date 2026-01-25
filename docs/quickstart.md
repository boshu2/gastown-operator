# Quick Start

Get the Gas Town Operator running in minutes.

**Use the kubectl-gt CLI** for normal workflows. YAML templates are available in [templates/](../templates/) for GitOps use cases.

---

## Prerequisites

- Kubernetes 1.26+ or OpenShift 4.13+
- Helm 3.8+
- `kubectl`/`oc` configured with cluster access
- Git SSH key for repository access
- Claude API key or OAuth credentials

---

## 1. Install Operator

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.1 \
  --namespace gastown-system \
  --create-namespace
```

For OpenShift with restricted SCC:

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.1 \
  --namespace gastown-system \
  --create-namespace \
  --set securityContext.allowPrivilegeEscalation=false \
  --set securityContext.runAsNonRoot=true \
  --set securityContext.runAsUser=null \
  --set securityContext.readOnlyRootFilesystem=true
```

---

## 2. Install kubectl-gt Plugin

```bash
# macOS (Apple Silicon)
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt

# macOS (Intel)
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-darwin-amd64
chmod +x kubectl-gt-darwin-amd64 && sudo mv kubectl-gt-darwin-amd64 /usr/local/bin/kubectl-gt

# Linux
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.1/kubectl-gt-linux-amd64
chmod +x kubectl-gt-linux-amd64 && sudo mv kubectl-gt-linux-amd64 /usr/local/bin/kubectl-gt
```

---

## 3. Set Up Credentials

```bash
# Create git credentials secret
kubectl create secret generic git-creds -n gastown-system \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519

# Sync Claude credentials from local ~/.claude/
kubectl gt auth sync -n gastown-system
```

---

## 4. Verify Installation

```bash
# Check operator
kubectl get pods -n gastown-system

# Check CRDs
kubectl get crds | grep gastown

# Check auth status
kubectl gt auth status -n gastown-system
```

You should see:
- `gastown-operator-controller-manager` pod running
- Six CRDs: `rigs`, `polecats`, `convoys`, `witnesses`, `refineries`, `beadstores`
- Auth status showing credentials synced

---

## 5. Create Your First Rig

```bash
kubectl gt rig create myproject \
  --git-url git@github.com:myorg/myproject.git \
  --prefix mp \
  -n gastown-system
```

Verify:

```bash
kubectl gt rig list -n gastown-system
kubectl gt rig status myproject -n gastown-system
```

---

## 6. Create a Polecat

Polecats are workers that execute bead issues. Dispatch work with `sling`:

```bash
# Dispatch with explicit name
kubectl gt sling mp-abc-123 myproject --name furiosa -n gastown-system

# Or with themed naming (mad-max, minerals, wasteland)
kubectl gt sling mp-abc-123 myproject --theme mad-max -n gastown-system

# Wait for pod to be ready
kubectl gt sling mp-abc-123 myproject --wait-ready --timeout 5m -n gastown-system
```

Watch it work:

```bash
kubectl gt polecat logs myproject/furiosa -f -n gastown-system
```

---

## 7. Create a Convoy (Batch Tracking)

Convoys track batches of beads for parallel execution:

```bash
kubectl gt convoy create "Wave 1 implementation" mp-abc-123 mp-def-456 mp-ghi-789 -n gastown-system
```

Check progress:

```bash
kubectl gt convoy list -n gastown-system
```

---

## CLI Commands Reference

| Command | Description |
|---------|-------------|
| `kubectl gt rig list` | List all rigs |
| `kubectl gt rig status <name>` | Show rig details |
| `kubectl gt rig create <name>` | Create a new rig |
| `kubectl gt polecat list [rig]` | List polecats |
| `kubectl gt polecat status <rig>/<name>` | Show polecat details |
| `kubectl gt polecat logs <rig>/<name>` | Stream polecat logs |
| `kubectl gt polecat nuke <rig>/<name>` | Terminate a polecat |
| `kubectl gt sling <bead-id> <rig>` | Dispatch work to a polecat |
| `kubectl gt convoy list` | List convoy batches |
| `kubectl gt convoy create <desc> <beads...>` | Create convoy |
| `kubectl gt auth sync` | Sync Claude creds to cluster |
| `kubectl gt auth status` | Check credential status |

All commands support `-o json` and `-o yaml` for machine-parseable output.

---

## Namespace Strategy

The operator uses a straightforward architecture:

| Namespace | Purpose | Resources |
|-----------|---------|-----------|
| `gastown-system` | Control plane + workloads | Operator, Polecats, Secrets |
| Cluster-scoped | Global resources | Rig CRDs |

---

## YAML Alternative

If you prefer declarative YAML for GitOps workflows, templates are in [templates/](../templates/):

| Template | Use |
|----------|-----|
| `polecat-minimal.yaml` | Minimal polecat (5 vars) |
| `polecat-kubernetes.yaml` | Full polecat (all options) |
| `convoy.yaml` | Batch tracking |
| `rig.yaml` | Project rig |

**Validate:** `./scripts/validate-template.sh <file>`

---

## Next Steps

- [User Guide](./USER_GUIDE.md) - Complete walkthrough with E2E examples
- [CRD Reference](./CRD_REFERENCE.md) - Full spec/status documentation
- [Architecture](./architecture.md) - How the operator works
- [Development](./development.md) - Contributing and local setup
