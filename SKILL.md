---
name: gastown-operator
version: 1.0.0
tier: solo
context: repo
description: >
  Create and manage Gas Town Kubernetes resources (Polecat, Convoy, Witness, Refinery).
  Triggers on "create polecat", "deploy convoy", "gastown k8s", "kubernetes polecat".
triggers:
  - "create polecat"
  - "deploy polecat"
  - "gastown kubernetes"
  - "gastown k8s"
  - "create convoy"
  - "deploy convoy"
  - "create witness"
  - "create refinery"
  - "kubernetes polecat"
  - "k8s polecat"
allowed-tools:
  - Read
  - Write
  - Bash(kubectl:*, helm:*, oc:*)
---

# Gas Town Operator Skill

Create Gas Town Kubernetes resources quickly and correctly.

---

## Quick Reference

| Resource | Template | Purpose |
|----------|----------|---------|
| Polecat (minimal) | [templates/polecat-minimal.yaml](templates/polecat-minimal.yaml) | Quick local polecat |
| Polecat (k8s) | [templates/polecat-kubernetes.yaml](templates/polecat-kubernetes.yaml) | Full k8s execution |
| Convoy | [templates/convoy.yaml](templates/convoy.yaml) | Batch tracking |
| Witness | [templates/witness.yaml](templates/witness.yaml) | Health monitoring |
| Refinery | [templates/refinery.yaml](templates/refinery.yaml) | Merge processing |
| Git Secret | [templates/secret-git-ssh.yaml](templates/secret-git-ssh.yaml) | SSH key for git |
| Claude Secret | [templates/secret-claude-creds.yaml](templates/secret-claude-creds.yaml) | Claude credentials |

---

## Minimal Polecat (Copy-Paste Ready)

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: {{POLECAT_NAME}}
  namespace: gastown-system
spec:
  rig: {{RIG_NAME}}
  desiredState: Working
  beadID: "{{BEAD_ID}}"
  executionMode: local
```

**Variables to replace:**
- `{{POLECAT_NAME}}` - e.g., `furiosa`, `nux` (from mad-max namepool)
- `{{RIG_NAME}}` - e.g., `athena`, `daedalus` (must exist)
- `{{BEAD_ID}}` - e.g., `at-1234`, `gt-5678` (bead to work on)

---

## Common Operations

### Create a Working Polecat

```bash
# 1. Verify rig exists
kubectl get rig {{RIG_NAME}}

# 2. Create polecat
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: furiosa
  namespace: gastown-system
spec:
  rig: athena
  desiredState: Working
  beadID: "at-1234"
  executionMode: local
EOF

# 3. Watch status
kubectl get polecat furiosa -n gastown-system -w
```

### Check Polecat Status

```bash
kubectl get polecats -n gastown-system
kubectl describe polecat {{POLECAT_NAME}} -n gastown-system
```

### Terminate a Polecat

```bash
kubectl patch polecat {{POLECAT_NAME}} -n gastown-system \
  --type=merge -p '{"spec":{"desiredState":"Terminated"}}'
```

---

## API Reference

**API Group:** `gastown.gastown.io`
**Version:** `v1alpha1`

| CRD | Scope | Purpose |
|-----|-------|---------|
| Rig | Cluster | Project workspace |
| Polecat | Namespaced | Autonomous worker |
| Convoy | Namespaced | Batch tracking |
| Witness | Namespaced | Health monitoring |
| Refinery | Namespaced | Merge processing |
| BeadStore | Namespaced | Issue sync config |

---

## Common Fields

### Polecat Spec

| Field | Required | Description |
|-------|----------|-------------|
| `rig` | Yes | Parent rig name |
| `desiredState` | Yes | `Idle`, `Working`, `Terminated` |
| `beadID` | No | Bead to work on |
| `executionMode` | No | `local` (tmux) or `kubernetes` (Pod) |
| `agent` | No | `claude-code`, `opencode`, `aider` |
| `kubernetes.*` | If k8s | Git repo, secrets, resources |

### Convoy Spec

| Field | Required | Description |
|-------|----------|-------------|
| `description` | Yes | Human-readable name |
| `trackedBeads` | Yes | List of bead IDs |
| `parallelism` | No | Max concurrent polecats |
| `rigRef` | No | Target rig |

---

## Validation

Before applying, run:

```bash
./scripts/validate-template.sh templates/polecat-kubernetes.yaml
```

---

## Anti-Patterns

See [FRICTION_POINTS.md](FRICTION_POINTS.md) for:
- Missing secrets
- Wrong namespace
- Invalid bead IDs
- Rig not found errors

---

## Full Documentation

| Topic | Location |
|-------|----------|
| CRD Reference | [docs/CRD_REFERENCE.md](docs/CRD_REFERENCE.md) |
| Secret Management | [docs/SECRET_MANAGEMENT.md](docs/SECRET_MANAGEMENT.md) |
| Troubleshooting | [docs/TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) |
| Architecture | [docs/architecture.md](docs/architecture.md) |
