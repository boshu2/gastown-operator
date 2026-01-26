# Troubleshooting Guide

This guide helps diagnose and resolve common issues with the Gas Town Operator.

**Use the kubectl-gt CLI** for most troubleshooting tasks.

## Table of Contents

- [Quick Diagnostics](#quick-diagnostics)
- [Operator Issues](#operator-issues)
- [Polecat Issues](#polecat-issues)
- [Rig Issues](#rig-issues)
- [Convoy Issues](#convoy-issues)
- [Authentication Issues](#authentication-issues)
- [Merge Conflicts](#merge-conflicts)

---

## Quick Diagnostics

```bash
# Check operator health
kubectl get pods -n gastown-system

# Check all resources via CLI
kubectl gt rig list -n gastown-system
kubectl gt polecat list -n gastown-system
kubectl gt convoy list -n gastown-system
kubectl gt auth status -n gastown-system

# Check CRDs installed
kubectl get crds | grep gastown
```

---

## Operator Issues

### Operator Pod Not Starting

**Symptoms**: Operator pod stuck in `Pending`, `CrashLoopBackOff`, or `Error` state.

**Diagnosis**:
```bash
kubectl describe pod -n gastown-system -l control-plane=controller-manager
kubectl logs -n gastown-system -l control-plane=controller-manager
```

**Common Causes**:

| Cause | Solution |
|-------|----------|
| Missing CRDs | Run `make install` to install CRDs |
| RBAC issues | Check ClusterRole and ClusterRoleBinding exist |
| Image pull error | Verify image exists and pull secrets are configured |
| Resource limits | Increase memory/CPU limits in deployment |

### Metrics Endpoint Not Responding

**Diagnosis**:
```bash
kubectl port-forward -n gastown-system svc/gastown-operator-controller-manager-metrics-service 8443:8443
curl -k https://localhost:8443/metrics
```

**Common Causes**:
- Metrics disabled (`--metrics-bind-address=0`)
- NetworkPolicy blocking access
- TLS certificate issues

---

## Polecat Issues

### Polecat Stuck in "Working" State

**Symptoms**: Polecat shows `phase: Working` for extended periods without progress.

**Diagnosis**:
```bash
# Check polecat status via CLI
kubectl gt polecat status <rig>/<name> -n gastown-system

# Stream logs
kubectl gt polecat logs <rig>/<name> -n gastown-system

# Check operator logs
kubectl logs -n gastown-system -l control-plane=controller-manager | grep <polecat-name>
```

**Resolution**:
1. Check logs for Claude activity
2. If truly stuck, terminate with nuke:
   ```bash
   kubectl gt polecat nuke <rig>/<name> -n gastown-system
   ```
3. Re-dispatch work:
   ```bash
   kubectl gt sling <bead-id> <rig> -n gastown-system
   ```

### Polecat Shows "Stuck" Phase

**Symptoms**: Witness controller detected idle polecat beyond threshold.

**Diagnosis**:
```bash
kubectl gt polecat status <rig>/<name> -n gastown-system
# Check conditions for "Stuck" reason
```

**Resolution**:
1. Check logs for Claude usage limits:
   ```bash
   kubectl gt polecat logs <rig>/<name> --tail 50 -n gastown-system
   ```
2. If unrecoverable, terminate and re-dispatch:
   ```bash
   kubectl gt polecat nuke <rig>/<name> -n gastown-system
   kubectl gt sling <bead-id> <rig> -n gastown-system
   ```

### Polecat Pod Fails to Start

**Diagnosis**:
```bash
kubectl gt polecat status <rig>/<name> -n gastown-system
kubectl describe pod polecat-<name> -n gastown-system
kubectl gt polecat logs <rig>/<name> -n gastown-system
```

**Common Causes**:

| Cause | Solution |
|-------|----------|
| Git clone failed | Verify git-creds secret has valid SSH key |
| Claude auth failed | Re-sync credentials: `kubectl gt auth sync --force` |
| Resource exhaustion | Increase resource limits in polecat spec |
| Security context | Ensure pod runs as nonroot (UID 65532) |

### Missing gitRepository Error

**Symptoms**: `sling` command creates polecat but pod fails with missing gitRepository.

**This was fixed in v0.4.1**. The sling command now fetches gitURL from the Rig spec.

If you see this error on v0.4.0:
```bash
# Upgrade kubectl-gt to v0.4.2
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.2/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt
```

---

## Rig Issues

### Rig Shows "Degraded" Phase

**Symptoms**: Rig condition `Ready=False` or phase `Degraded`.

**Diagnosis**:
```bash
kubectl gt rig status <name> -n gastown-system
# Check conditions for specific failure reason
```

**Common Causes**:

| Condition Reason | Cause | Solution |
|------------------|-------|----------|
| `PathNotFound` | localPath doesn't exist | Verify path exists |
| `GTCLIError` | gt CLI not responding | Check gt CLI connectivity |
| `RigNotRegistered` | Rig not in gt town | Create rig: `kubectl gt rig create <name> --git-url ...` |

### Rig Creation Fails

**Diagnosis**:
```bash
kubectl gt rig create <name> --git-url git@github.com:org/repo.git --prefix xx
# Check error message
```

**Common Causes**:
- Git URL unreachable
- Prefix already in use by another rig
- Invalid rig name (must be DNS-compatible)

---

## Convoy Issues

### Convoy Shows "Failed" Phase

**Diagnosis**:
```bash
kubectl gt convoy list -n gastown-system
# Find the failed convoy and check status
```

**Resolution**:
1. Identify which beads failed
2. Investigate individual polecat issues
3. Re-dispatch failed beads or close convoy if partial success is acceptable

### Convoy Progress Not Updating

**Symptoms**: Convoy status shows stale data.

**Diagnosis**:
- Check operator logs for convoy reconciliation errors
- Verify polecats are making progress

**Resolution**:
```bash
# Manually trigger reconciliation
kubectl annotate convoy <name> trigger-reconcile=$(date +%s) -n gastown-system
```

---

## Authentication Issues

### Claude Credentials Not Working

**Symptoms**: Polecat pod fails with authentication error.

**Diagnosis**:
```bash
kubectl gt auth status -n gastown-system
kubectl gt polecat logs <rig>/<name> -n gastown-system | grep -i auth
```

**Resolution**:
```bash
# Re-sync credentials (force refresh)
kubectl gt auth sync --force -n gastown-system
```

### OAuth Token Expired

OAuth tokens expire after ~24 hours.

**Resolution**:
```bash
# 1. Re-login on your laptop
claude /login

# 2. Re-sync to cluster
kubectl gt auth sync --force -n gastown-system
```

### Git Clone Fails

**Diagnosis**:
```bash
kubectl gt polecat logs <rig>/<name> -n gastown-system
```

**Verify SSH key**:
```bash
kubectl get secret git-creds -n gastown-system -o jsonpath='{.data.ssh-privatekey}' | base64 -d | head -1
# Should show: -----BEGIN OPENSSH PRIVATE KEY-----
```

---

## Merge Conflicts

### Refinery Stuck in "Error" State

**Symptoms**: Refinery shows merge failure.

**Diagnosis**:
```bash
kubectl describe refinery <name> -n gastown-system
# Check status.lastError
```

**Manual Resolution**:
```bash
# Navigate to rig
cd ~/gt/<rig>/mayor/rig

# Check status
git status

# Resolve conflicts manually
git checkout --theirs .beads/issues.jsonl  # For beads conflicts
git add .
git commit -m "merge: resolve conflict"

# Retry refinery
kubectl annotate refinery <name> retry=$(date +%s) -n gastown-system
```

### Beads Conflicts

Beads files (.beads/issues.jsonl) are append-only, so conflicts are usually safe to resolve with `--theirs`:

```bash
git checkout --theirs .beads/issues.jsonl
git add .beads/
bd sync
git commit -m "merge: resolve beads conflict"
```

---

## Getting Help

If issues persist:

1. Check operator logs for detailed error messages:
   ```bash
   kubectl logs -n gastown-system -l control-plane=controller-manager --tail 100
   ```
2. Get resource status in JSON for detailed analysis:
   ```bash
   kubectl gt polecat status <rig>/<name> -o json -n gastown-system
   ```
3. File an issue with:
   - Operator version (`kubectl get deployment -n gastown-system -o jsonpath='{.items[0].spec.template.spec.containers[0].image}'`)
   - Kubernetes version (`kubectl version`)
   - Relevant CLI output
   - Operator logs
