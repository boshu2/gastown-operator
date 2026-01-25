# Troubleshooting Guide

Common issues and solutions for Gas Town Operator.

---

## Quick Diagnosis

```bash
# Check operator health
kubectl get pods -n gastown-system -l app.kubernetes.io/name=gastown-operator
kubectl logs -n gastown-system -l app.kubernetes.io/name=gastown-operator --tail=50

# Check polecat status
kubectl get polecats -A
kubectl describe polecat <name> -n gastown-system

# Check events
kubectl get events -n gastown-system --sort-by='.lastTimestamp' | tail -20
kubectl get events -n gastown-workers --sort-by='.lastTimestamp' | tail -20
```

---

## Polecat Issues

### Polecat Stuck in Pending

**Symptoms:** Polecat stays in `Pending` phase, no pod created.

**Causes:**
1. Rig doesn't exist
2. Missing secrets (Kubernetes mode)
3. Operator not running

**Diagnosis:**
```bash
kubectl describe polecat <name> -n gastown-system | grep -A5 "Conditions:"
kubectl get events -n gastown-system --field-selector involvedObject.name=<name>
```

**Solutions:**
```bash
# Check rig exists
kubectl get rig <rig-name>

# Create rig if missing
kubectl apply -f templates/rig.yaml

# Check secrets (Kubernetes mode)
kubectl get secrets -n gastown-workers

# Check operator logs
kubectl logs -n gastown-system -l app.kubernetes.io/name=gastown-operator --tail=100
```

---

### Polecat Stuck in Working

**Symptoms:** Polecat stays in `Working` phase for too long.

**Causes:**
1. Agent stuck or crashed
2. No timeout set
3. Git push failing
4. Task too complex

**Diagnosis:**
```bash
# Check pod logs
kubectl logs -n gastown-workers -l polecat=<name> -f

# Exec into pod for debugging
kubectl exec -it -n gastown-workers -l polecat=<name> -- /bin/sh
```

**Solutions:**
```bash
# Force terminate
kubectl patch polecat <name> -n gastown-system \
  --type=merge -p '{"spec":{"desiredState":"Terminated"}}'

# For future: always set timeout
# spec.kubernetes.activeDeadlineSeconds: 3600
```

---

### Polecat Failed

**Symptoms:** Polecat in `Failed` phase.

**Diagnosis:**
```bash
kubectl describe polecat <name> -n gastown-system
kubectl get polecat <name> -n gastown-system -o jsonpath='{.status.message}'
```

**Common Causes:**

| Error Message | Cause | Fix |
|---------------|-------|-----|
| `rig "x" not found` | Rig doesn't exist | Create rig first |
| `secret "x" not found` | Missing secret | Create required secret |
| `authentication failed` | Invalid credentials | Update secret |
| `repository not found` | Wrong git URL | Fix gitRepository |
| `permission denied` | SSH key not authorized | Add key to git provider |
| `deadline exceeded` | Timeout reached | Increase timeout or simplify task |

---

## Git Issues

### Permission Denied (publickey)

**Symptoms:** Pod fails with "Permission denied (publickey)" in init container.

**Diagnosis:**
```bash
kubectl logs -n gastown-workers -l polecat=<name> -c git-clone
```

**Solutions:**
```bash
# Verify secret exists and has correct key
kubectl get secret git-credentials -n gastown-workers -o jsonpath='{.data.ssh-privatekey}' | base64 -d | head -1
# Should show: -----BEGIN OPENSSH PRIVATE KEY-----

# Verify key is added to git provider
ssh -T git@github.com  # Test locally first

# Recreate secret
kubectl delete secret git-credentials -n gastown-workers
kubectl create secret generic git-credentials -n gastown-workers \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519
```

---

### Repository Not Found

**Symptoms:** Git clone fails with 404 or "repository not found".

**Diagnosis:**
```bash
kubectl logs -n gastown-workers -l polecat=<name> -c git-clone
```

**Solutions:**
```bash
# Check URL format
# WRONG: https://github.com/org/repo
# RIGHT: git@github.com:org/repo.git

# Verify access locally
git ls-remote git@github.com:org/repo.git

# Check deploy key permissions (needs read access at minimum)
```

---

### Push Rejected

**Symptoms:** Agent completes but push fails.

**Diagnosis:**
```bash
kubectl logs -n gastown-workers -l polecat=<name> | grep -i push
```

**Solutions:**
```bash
# Branch protection - push to different branch
# Update spec.kubernetes.workBranch

# Force push needed - usually means branch diverged
# May need manual intervention

# SSH key needs write access
# Add deploy key with write permission
```

---

## Secret Issues

### Claude Authentication Failed

**Symptoms:** Agent fails to start with auth error.

**Diagnosis:**
```bash
kubectl logs -n gastown-workers -l polecat=<name> -c agent | head -20
```

**Solutions:**

**API Key:**
```bash
# Verify key format
kubectl get secret claude-credentials -n gastown-workers -o jsonpath='{.data.api-key}' | base64 -d
# Should start with: sk-ant-

# Recreate with valid key
kubectl delete secret claude-credentials -n gastown-workers
kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=api-key=$ANTHROPIC_API_KEY
```

**OAuth Credentials:**
```bash
# OAuth tokens expire in 24 hours
# Re-run claude login and update secret
claude login
CREDS=$(security find-generic-password -s "Claude Code-credentials" -w)
kubectl delete secret claude-credentials -n gastown-workers
kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=.credentials.json="$CREDS"
```

---

## Operator Issues

### Operator Not Running

**Symptoms:** No pods in gastown-system, polecats not processing.

**Diagnosis:**
```bash
kubectl get pods -n gastown-system
kubectl get deploy -n gastown-system
```

**Solutions:**
```bash
# Check deployment
kubectl describe deploy gastown-operator -n gastown-system

# Check for image pull issues
kubectl get events -n gastown-system | grep -i pull

# Reinstall
helm upgrade --install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 -n gastown-system
```

---

### Operator Crash Loop

**Symptoms:** Operator pod in CrashLoopBackOff.

**Diagnosis:**
```bash
kubectl logs -n gastown-system -l app.kubernetes.io/name=gastown-operator --previous
kubectl describe pod -n gastown-system -l app.kubernetes.io/name=gastown-operator
```

**Common Causes:**
1. CRDs not installed
2. RBAC permissions missing
3. Invalid configuration

**Solutions:**
```bash
# Reinstall CRDs
kubectl apply -f config/crd/bases/

# Check RBAC
kubectl auth can-i --list --as=system:serviceaccount:gastown-system:gastown-operator

# Check helm values
helm get values gastown-operator -n gastown-system
```

---

## Resource Issues

### Pod OOMKilled

**Symptoms:** Pod terminated with OOMKilled.

**Diagnosis:**
```bash
kubectl describe pod -n gastown-workers -l polecat=<name> | grep -A5 "Last State"
```

**Solutions:**
```yaml
# Increase memory limits
spec:
  kubernetes:
    resources:
      requests:
        memory: "2Gi"
      limits:
        memory: "8Gi"
```

---

### Pod Evicted

**Symptoms:** Pod evicted due to node pressure.

**Solutions:**
```yaml
# Add resource requests to ensure scheduling
spec:
  kubernetes:
    resources:
      requests:
        cpu: "500m"
        memory: "1Gi"

# Or add node selector for dedicated nodes
spec:
  kubernetes:
    nodeSelector:
      dedicated: ai-workloads
```

---

## Debugging Commands

```bash
# Full polecat state
kubectl get polecat <name> -n gastown-system -o yaml

# Operator logs (verbose)
kubectl logs -n gastown-system -l app.kubernetes.io/name=gastown-operator -f

# All gastown resources
kubectl get rigs,polecats,convoys,witnesses,refineries -A

# Events timeline
kubectl get events -A --sort-by='.lastTimestamp' | grep -i gastown

# Resource usage
kubectl top pods -n gastown-workers
```
