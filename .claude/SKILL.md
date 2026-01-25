---
name: gastown-operator
version: 1.0.0
tier: solo
context: repo
description: >
  Kubernetes operator for Gas Town multi-agent orchestration. Triggers on
  "create polecat", "spawn worker", "kubernetes polecat", "deploy convoy".
triggers:
  - "create polecat"
  - "deploy polecat"
  - "spawn polecat"
  - "kubernetes polecat"
  - "k8s polecat"
  - "create convoy"
  - "deploy convoy"
  - "create witness"
  - "create refinery"
  - "gastown kubernetes"
  - "gastown k8s"
allowed-tools:
  - Read
  - Write
  - Bash(kubectl:*, helm:*, oc:*)
  - Grep
  - Glob
context-budget:
  skill-md: 3KB
  references-total: 25KB
  typical-session: 8KB
---

# Gas Town Operator Skill

Create Gas Town Kubernetes resources quickly and correctly.

**Use the kubectl-gt CLI** for normal workflows. YAML templates are for GitOps.

---

## Critical Facts

| Fact | Value |
|------|-------|
| API Group | `gastown.gastown.io/v1alpha1` |
| Operator NS | `gastown-system` |
| CRDs | Rig, Polecat, Convoy, Witness, Refinery, BeadStore |
| CLI Plugin | `kubectl-gt` |

---

## Golden Commands (CLI-First)

### Create Rig

```bash
kubectl gt rig create {{RIG}} \
  --git-url git@github.com:org/repo.git \
  --prefix {{PREFIX}} \
  -n gastown-system
```

### Dispatch Work (Sling)

```bash
# With explicit name
kubectl gt sling {{BEAD_ID}} {{RIG}} --name {{NAME}} -n gastown-system

# With themed name (mad-max, minerals, wasteland)
kubectl gt sling {{BEAD_ID}} {{RIG}} --theme mad-max -n gastown-system

# Wait for pod ready
kubectl gt sling {{BEAD_ID}} {{RIG}} --wait-ready --timeout 5m -n gastown-system
```

**Variables:** `{{BEAD_ID}}` (e.g., at-1234), `{{RIG}}` (e.g., athena), `{{NAME}}` (e.g., furiosa)

### Convoy (Batch Tracking)

```bash
kubectl gt convoy create "Wave 1" {{BEAD1}} {{BEAD2}} {{BEAD3}} -n gastown-system
```

### Monitor

```bash
kubectl gt polecat list -n gastown-system
kubectl gt polecat logs {{RIG}}/{{NAME}} -f -n gastown-system
kubectl gt polecat status {{RIG}}/{{NAME}} -n gastown-system
```

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `kubectl gt rig list` | List rigs |
| `kubectl gt rig create <name>` | Create rig |
| `kubectl gt polecat list` | List polecats |
| `kubectl gt polecat logs <rig>/<name>` | Stream logs |
| `kubectl gt sling <bead> <rig>` | Dispatch work |
| `kubectl gt convoy list` | List convoys |
| `kubectl gt auth sync` | Sync Claude creds |

All commands support `-o json` and `-o yaml` output.

---

## Templates (YAML Alternative)

If you need declarative YAML (GitOps), templates are in `templates/`:

| Template | Use |
|----------|-----|
| `polecat-minimal.yaml` | Minimal K8s polecat (5 vars) |
| `polecat-kubernetes.yaml` | Full K8s polecat (all options) |
| `convoy.yaml` | Batch tracking |
| `secret-*.yaml` | Credentials |

**Validate:** `./scripts/validate-template.sh <file>`

---

## Quick Checks

```bash
kubectl gt rig list -n gastown-system          # List rigs
kubectl gt polecat list -n gastown-system      # List polecats
kubectl gt auth status -n gastown-system       # Check creds
kubectl get secrets -n gastown-system          # Check secrets
```

---

## Common Errors

| Error | Fix |
|-------|-----|
| `rig not found` | Create rig first: `kubectl gt rig create <name> --git-url ...` |
| `secret not found` | Create secrets: `kubectl create secret generic git-creds ...` |
| `missing gitRepository` | Fixed in v0.4.1 - sling now fetches from Rig |
| Stuck in Working | Check logs: `kubectl gt polecat logs <rig>/<name>` |

---

## JIT Load References

Load these when you need deeper context:

| Topic | Reference |
|-------|-----------|
| Full CRD specs | `.claude/references/CRD_REFERENCE.md` |
| Kubernetes mode | `.claude/references/KUBERNETES_MODE.md` |
| Troubleshooting | `.claude/references/TROUBLESHOOTING.md` |
| Anti-patterns | `FRICTION_POINTS.md` |
| Full templates | `templates/*.yaml` |
