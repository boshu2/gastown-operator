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
---

# Gas Town Operator Skill

Create Gas Town Kubernetes resources (Polecat, Convoy, Witness, Refinery) quickly and correctly.

---

## Critical Facts (Memorize)

### API Group

**API Group:** `gastown.gastown.io`
**Version:** `v1alpha1`
**Namespace:** `gastown-system` (operator), `gastown-workers` (polecats)

### CRDs

| CRD | Scope | Purpose |
|-----|-------|---------|
| Rig | Cluster | Project workspace |
| Polecat | Namespaced | Autonomous worker agent |
| Convoy | Namespaced | Batch tracking |
| Witness | Namespaced | Health monitoring |
| Refinery | Namespaced | Merge processing |
| BeadStore | Namespaced | Issue sync config |

### Required Secrets (for Kubernetes mode)

| Secret | Namespace | Purpose |
|--------|-----------|---------|
| `git-credentials` | gastown-workers | SSH key for git clone/push |
| `claude-credentials` | gastown-workers | API key or OAuth creds |

---

## Golden Command Templates

### Minimal Polecat (Local Execution)

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

**Variables:**
- `{{POLECAT_NAME}}` - Unique name (e.g., `furiosa`, `nux`)
- `{{RIG_NAME}}` - Parent rig (e.g., `athena`, `daedalus`)
- `{{BEAD_ID}}` - Bead to work on (e.g., `at-1234`)

### Create Polecat One-Liner

```bash
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
```

---

## Common Failures

### "rig not found"

Rig doesn't exist. Create it first:
```bash
kubectl get rigs  # Check existing rigs
kubectl apply -f templates/rig.yaml  # Create rig
```

### "secret not found"

Missing credentials for Kubernetes mode:
```bash
# Check secrets
kubectl get secrets -n gastown-workers

# Create git credentials
kubectl create secret generic git-credentials -n gastown-workers \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519

# Create Claude credentials
kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=api-key=$ANTHROPIC_API_KEY
```

### Polecat stuck in Pending

Check pod events:
```bash
kubectl describe polecat/{{NAME}} -n gastown-system
kubectl get events -n gastown-workers --sort-by='.lastTimestamp' | tail -10
```

### Permission denied (publickey)

SSH key not configured with git provider:
```bash
# Check key format
kubectl get secret git-credentials -n gastown-workers -o jsonpath='{.data.ssh-privatekey}' | base64 -d | head -1
# Should show: -----BEGIN OPENSSH PRIVATE KEY-----
```

---

## Templates (Copy-Paste Ready)

All templates in `templates/` with `{{VARIABLE}}` markers:

| Template | Purpose |
|----------|---------|
| `polecat-minimal.yaml` | Quick local polecat (3 variables) |
| `polecat-kubernetes.yaml` | Full K8s execution with all options |
| `convoy.yaml` | Batch tracking |
| `witness.yaml` | Health monitoring |
| `refinery.yaml` | Merge processing |
| `secret-git-ssh.yaml` | Git SSH credentials |
| `secret-claude-creds.yaml` | Claude API credentials |

**Validate before applying:**
```bash
./scripts/validate-template.sh templates/polecat-kubernetes.yaml
```

---

## Monitoring

```bash
# List polecats
kubectl get polecats -n gastown-system

# Watch polecat status
kubectl get polecat {{NAME}} -n gastown-system -w

# Check polecat details
kubectl describe polecat {{NAME}} -n gastown-system

# View tmux session (local mode)
tmux attach -t gt-{{RIG}}-{{POLECAT}}

# View pod logs (kubernetes mode)
kubectl logs -n gastown-workers -l polecat={{NAME}} -f
```

---

## Checklist Before Creating Polecat

- [ ] Rig exists: `kubectl get rig {{RIG_NAME}}`
- [ ] Bead exists: `bd show {{BEAD_ID}}`
- [ ] Namespace exists: `kubectl get ns gastown-system`
- [ ] For K8s mode: secrets exist in `gastown-workers`
- [ ] For K8s mode: `activeDeadlineSeconds` set (prevent runaway)

---

## JIT Load

| Topic | Location |
|-------|----------|
| Full templates | `templates/*.yaml` |
| Anti-patterns | `FRICTION_POINTS.md` |
| CRD Reference | `docs/CRD_REFERENCE.md` |
| Troubleshooting | `docs/TROUBLESHOOTING.md` |
| User Guide | `docs/USER_GUIDE.md` |
