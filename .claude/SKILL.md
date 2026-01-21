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

---

## Critical Facts

| Fact | Value |
|------|-------|
| API Group | `gastown.gastown.io/v1alpha1` |
| Operator NS | `gastown-system` |
| Workers NS | `gastown-workers` |
| CRDs | Rig, Polecat, Convoy, Witness, Refinery, BeadStore |

---

## Golden Command: Create Polecat

```bash
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: {{NAME}}
  namespace: gastown-system
spec:
  rig: {{RIG}}
  desiredState: Working
  beadID: "{{BEAD_ID}}"
  executionMode: local
EOF
```

**Variables:** `{{NAME}}` (e.g., furiosa), `{{RIG}}` (e.g., athena), `{{BEAD_ID}}` (e.g., at-1234)

---

## Templates

All in `templates/` with `{{VARIABLE}}` markers:

| Template | Use |
|----------|-----|
| `polecat-minimal.yaml` | Local execution (3 vars) |
| `polecat-kubernetes.yaml` | K8s execution (full) |
| `convoy.yaml` | Batch tracking |
| `secret-*.yaml` | Credentials |

**Validate:** `./scripts/validate-template.sh <file>`

---

## Quick Checks

```bash
kubectl get polecats -n gastown-system     # List polecats
kubectl get rig {{RIG}}                     # Verify rig exists
kubectl get secrets -n gastown-workers      # Check secrets (K8s mode)
```

---

## Common Errors

| Error | Fix |
|-------|-----|
| `rig not found` | Create rig first: `kubectl apply -f templates/rig.yaml` |
| `secret not found` | Create secrets in `gastown-workers` namespace |
| Stuck in Working | Check tmux (`tmux attach -t gt-{{RIG}}-{{NAME}}`) or pod logs |

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
