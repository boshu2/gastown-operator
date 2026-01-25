# Friction Points: Common Mistakes and Fixes

This document catalogs common mistakes when creating Gas Town Kubernetes resources and how to fix them.

---

## Quick Reference: Error to Fix

| Error | Likely Cause | Fix |
|-------|--------------|-----|
| `rig "xyz" not found` | Rig doesn't exist | Create rig first or check name spelling |
| `secret "xyz" not found` | Missing secret | Create secret before polecat |
| `invalid bead ID format` | Wrong bead ID pattern | Use `prefix-xxxx` format (e.g., `at-1234`) |
| `permission denied (publickey)` | Git SSH key invalid | Check secret has correct key, add to git provider |
| `authentication failed` | Claude creds expired | Re-run `claude login`, update secret |
| `namespace not found` | Missing namespace | Create namespace first |
| `polecat stuck in Working` | Various causes | Check pod logs for details |

---

## Anti-Pattern 1: Creating Polecat Before Rig

**Wrong:**
```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: furiosa
spec:
  rig: athena  # Rig doesn't exist yet!
  desiredState: Working
  beadID: "at-1234"
```

**Error:**
```
Error from server: error creating polecat: rig "athena" not found
```

**Fix:**
```bash
# 1. Create rig first
kubectl apply -f - <<EOF
apiVersion: gastown.gastown.io/v1alpha1
kind: Rig
metadata:
  name: athena
spec:
  gitURL: "git@github.com:myorg/myproject.git"
  beadsPrefix: "at"
  localPath: "/home/user/gt/athena"
EOF

# 2. Wait for rig to be ready
kubectl wait --for=condition=Ready rig/athena --timeout=60s

# 3. Then create polecat
kubectl apply -f polecat.yaml
```

---

## Anti-Pattern 2: Missing Secrets in Kubernetes Mode

**Wrong:**
```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: k8s-worker
spec:
  rig: athena
  desiredState: Working
  beadID: "at-1234"
  executionMode: kubernetes
  kubernetes:
    gitRepository: "git@github.com:myorg/repo.git"
    # gitSecretRef missing!
    # claudeCredsSecretRef missing!
```

**Error:**
```
Error from server: spec.kubernetes.gitSecretRef is required when executionMode is kubernetes
```

**Fix:**
```bash
# 1. Create git secret
kubectl create secret generic git-credentials \
  --from-file=ssh-privatekey=$HOME/.ssh/gastown-key \
  -n gastown-system

# 2. Create claude secret
kubectl create secret generic claude-credentials \
  --from-file=credentials.json=$HOME/.claude/credentials.json \
  -n gastown-system

# 3. Reference them in polecat
spec:
  kubernetes:
    gitSecretRef:
      name: git-credentials
    claudeCredsSecretRef:
      name: claude-credentials
```

---

## Anti-Pattern 3: Wrong Namespace

**Wrong:**
```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: furiosa
  namespace: default  # Wrong namespace!
spec:
  # ...
```

**Symptoms:**
- Polecat created but operator doesn't see it
- No reconciliation happens
- Status never updates

**Fix:**
```yaml
metadata:
  name: furiosa
  namespace: gastown-system  # Operator's namespace
```

Or deploy operator to watch all namespaces (not recommended for production).

---

## Anti-Pattern 4: Invalid Bead ID Format

**Wrong:**
```yaml
spec:
  beadID: "1234"       # Missing prefix
  # or
  beadID: "AT-1234"    # Wrong case (uppercase)
  # or
  beadID: "athena-1234" # Prefix too long
```

**Error:**
```
Warning: bead ID "1234" does not match expected pattern [a-z]{2,10}-[a-z0-9]+
```

**Fix:**
```yaml
spec:
  beadID: "at-1234"  # Correct: lowercase prefix + hyphen + ID
```

**Pattern:** `^[a-z]{2,10}-[a-z0-9]+$`

---

## Anti-Pattern 5: Git SSH Key Not Added to Provider

**Wrong setup:**
```bash
# Created secret
kubectl create secret generic git-credentials \
  --from-file=ssh-privatekey=./my-key \
  -n gastown-system

# But forgot to add public key to GitHub/GitLab!
```

**Error (in polecat pod logs):**
```
Permission denied (publickey).
fatal: Could not read from remote repository.
```

**Fix:**
```bash
# 1. Get the public key
ssh-keygen -y -f ./my-key > ./my-key.pub

# 2. Add to GitHub: Settings -> Deploy keys -> Add
# 3. IMPORTANT: Check "Allow write access"

# 4. Verify
ssh -i ./my-key -T git@github.com
# Should say: "Hi <user>! You've successfully authenticated..."
```

---

## Anti-Pattern 6: Expired Claude Credentials

**Symptoms:**
- Polecat pod starts but Claude fails to authenticate
- Logs show "authentication failed" or "token expired"

**Pod logs:**
```
Error: authentication failed - credentials may be expired
```

**Fix:**
```bash
# 1. Re-login locally
claude login

# 2. Update secret
kubectl delete secret claude-credentials -n gastown-system
kubectl create secret generic claude-credentials \
  --from-file=credentials.json=$HOME/.claude/credentials.json \
  -n gastown-system

# 3. Restart polecat
kubectl delete polecat <name> -n gastown-system
kubectl apply -f polecat.yaml
```

---

## Anti-Pattern 7: Convoy with Non-Existent Beads

**Wrong:**
```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Convoy
metadata:
  name: wave-1
spec:
  description: "Wave 1"
  trackedBeads:
    - "at-9999"  # Doesn't exist!
    - "at-0000"  # Doesn't exist!
```

**Symptoms:**
- Convoy stays in `Pending` forever
- No polecats created
- Progress shows 0/2

**Fix:**
```bash
# 1. Verify beads exist
bd show at-9999

# 2. If not found, create them first
bd create "Task description" --type feature --id at-9999

# 3. Or use existing bead IDs
bd ready --parent=<epic>  # List ready beads
```

---

## Anti-Pattern 8: Resources Too Low

**Wrong:**
```yaml
spec:
  kubernetes:
    resources:
      limits:
        cpu: "100m"     # Too low!
        memory: "256Mi" # Too low!
```

**Symptoms:**
- Pod OOMKilled
- Claude times out
- Slow performance

**Fix:**
```yaml
spec:
  kubernetes:
    resources:
      requests:
        cpu: "500m"
        memory: "1Gi"
      limits:
        cpu: "2"
        memory: "4Gi"
```

**Recommended minimums:**
| Resource | Request | Limit |
|----------|---------|-------|
| CPU | 500m | 2 |
| Memory | 1Gi | 4Gi |

---

## Anti-Pattern 9: Forgetting to Push Work

**Symptoms:**
- Polecat completes work
- Work never appears in main branch
- `gt convoy list` shows complete but changes missing

**Root cause:** Polecat finished but changes weren't pushed.

**Verification:**
```bash
# Check polecat branch status
kubectl get polecat <name> -o jsonpath='{.status.cleanupStatus}'
# If shows "has_unpushed" - problem!

# Check branch
git -C ~/gt/<rig>/polecats/<name> log --oneline -5
git -C ~/gt/<rig>/polecats/<name> status
```

**Fix:**
```bash
# Push the work
git -C ~/gt/<rig>/polecats/<name> push -u origin HEAD

# Then merge via refinery or manually
git -C ~/gt/<rig>/mayor/rig fetch origin
git -C ~/gt/<rig>/mayor/rig merge origin/polecat/<branch>
```

---

## Debugging Checklist

When something goes wrong, check in order:

1. **Operator running?**
   ```bash
   kubectl get pods -n gastown-system -l control-plane=controller-manager
   ```

2. **Resource exists?**
   ```bash
   kubectl get polecat,convoy,rig -n gastown-system
   ```

3. **Conditions healthy?**
   ```bash
   kubectl get polecat <name> -o jsonpath='{.status.conditions}'
   ```

4. **Operator logs?**
   ```bash
   kubectl logs -n gastown-system -l control-plane=controller-manager --tail=100
   ```

5. **Pod logs?**
   ```bash
   kubectl logs polecat-<name> -n <namespace> --all-containers
   ```

---

## See Also

- [TROUBLESHOOTING.md](docs/TROUBLESHOOTING.md) - Full troubleshooting guide
- [CRD_REFERENCE.md](docs/CRD_REFERENCE.md) - Complete field reference
- [SECRET_MANAGEMENT.md](docs/SECRET_MANAGEMENT.md) - Credential setup
