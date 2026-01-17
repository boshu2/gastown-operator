# Troubleshooting Guide

This guide helps diagnose and resolve common issues with the Gas Town Operator.

## Table of Contents

- [Operator Issues](#operator-issues)
- [Polecat Issues](#polecat-issues)
- [Rig Issues](#rig-issues)
- [Convoy Issues](#convoy-issues)
- [gt CLI Connectivity](#gt-cli-connectivity)
- [Merge Conflicts](#merge-conflicts)

---

## Operator Issues

### Operator Pod Not Starting

**Symptoms**: Operator pod stuck in `Pending`, `CrashLoopBackOff`, or `Error` state.

**Diagnosis**:
```bash
kubectl describe pod -n gastown-operator-system -l control-plane=controller-manager
kubectl logs -n gastown-operator-system -l control-plane=controller-manager
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
kubectl port-forward -n gastown-operator-system svc/gastown-operator-controller-manager-metrics-service 8443:8443
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
# Check polecat status
kubectl get polecat <name> -o yaml

# For local execution mode, check tmux session
tmux capture-pane -t gt-<rig>-<polecat> -p | tail -50

# Check operator logs for errors
kubectl logs -n gastown-operator-system -l control-plane=controller-manager | grep <polecat-name>
```

**Resolution**:
1. Check if the underlying bead is actually being worked on
2. Verify gt CLI connectivity (see below)
3. If truly stuck, reset the polecat:
   ```bash
   kubectl patch polecat <name> --type=merge -p '{"spec":{"desiredState":"Terminated"}}'
   ```

### Polecat Shows "Stuck" Phase

**Symptoms**: Witness controller detected idle polecat beyond threshold.

**Diagnosis**:
```bash
kubectl describe polecat <name>
# Check conditions for "Stuck" reason
```

**Resolution**:
1. Check tmux session for Claude usage limits
2. Nudge the polecat:
   ```bash
   tmux send-keys -t gt-<rig>-<polecat> "continue with your task" Enter
   ```
3. If unrecoverable, terminate and re-dispatch:
   ```bash
   kubectl patch polecat <name> --type=merge -p '{"spec":{"desiredState":"Terminated"}}'
   gt sling <bead-id> <rig>
   ```

### Polecat Pod Fails (Kubernetes Mode)

**Diagnosis**:
```bash
kubectl describe pod polecat-<name> -n <namespace>
kubectl logs polecat-<name> -n <namespace> -c git-init  # Init container
kubectl logs polecat-<name> -n <namespace> -c claude     # Main container
```

**Common Causes**:

| Cause | Solution |
|-------|----------|
| Git clone failed | Verify gitSecretRef has valid SSH key |
| Claude auth failed | Verify claudeCredsSecretRef has valid credentials |
| Resource exhaustion | Increase resource limits in spec.kubernetes.resources |
| Security context | Ensure pod runs as nonroot (UID 65532) |

---

## Rig Issues

### Rig Shows "Degraded" Phase

**Symptoms**: Rig condition `Ready=False` or phase `Degraded`.

**Diagnosis**:
```bash
kubectl describe rig <name>
# Check conditions for specific failure reason
```

**Common Causes**:

| Condition Reason | Cause | Solution |
|------------------|-------|----------|
| `PathNotFound` | localPath doesn't exist on operator host | Verify path exists |
| `GTCLIError` | gt CLI not responding | Check gt CLI connectivity |
| `RigNotRegistered` | Rig not in gt town | Run `gt rig add <name>` |

### Rig Not Syncing Beads

**Diagnosis**:
```bash
kubectl get beadstore -n <namespace>
kubectl describe beadstore <name>
```

**Resolution**:
1. Verify BeadStore is configured for the rig
2. Check gitSecretRef is valid
3. Manually sync: `bd sync` in the rig directory

---

## Convoy Issues

### Convoy Shows "Failed" Phase

**Diagnosis**:
```bash
kubectl describe convoy <name>
# Check status.pendingBeads vs status.completedBeads
```

**Resolution**:
1. Identify which beads failed: check `status.pendingBeads`
2. Investigate individual bead issues
3. Re-dispatch failed beads or close convoy if partial success is acceptable

### Convoy Progress Not Updating

**Symptoms**: Convoy status shows stale data.

**Diagnosis**:
- Check operator logs for convoy reconciliation errors
- Verify gt CLI can query convoy status

**Resolution**:
```bash
# Manually trigger reconciliation
kubectl annotate convoy <name> trigger-reconcile=$(date +%s)
```

---

## gt CLI Connectivity

### Verifying gt CLI Works

From the operator pod:
```bash
kubectl exec -n gastown-operator-system deploy/gastown-operator-controller-manager -- gt rig list --json
```

### Common gt CLI Errors

| Error | Cause | Solution |
|-------|-------|----------|
| `command not found: gt` | gt binary not installed | Set GT_PATH env var or install gt |
| `GT_TOWN_ROOT not set` | Missing configuration | Set GT_TOWN_ROOT env var |
| `permission denied` | File permissions | Ensure operator has access to town root |
| `timeout` | gt command hung | Check circuit breaker state, increase timeout |

### Circuit Breaker Tripped

When gt CLI fails repeatedly, the circuit breaker opens to prevent cascading failures.

**Symptoms**: Errors containing "circuit breaker is open"

**Diagnosis**:
```bash
# Check metrics
curl -k https://localhost:8443/metrics | grep gastown_gt_cli
```

**Resolution**:
1. Wait for reset timeout (default 30s)
2. Fix underlying gt CLI issue
3. The circuit will close automatically on successful calls

---

## Merge Conflicts

### Refinery Stuck in "Error" State

**Symptoms**: Refinery shows merge failure.

**Diagnosis**:
```bash
kubectl describe refinery <name>
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
kubectl annotate refinery <name> retry=$(date +%s)
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

1. Check operator logs for detailed error messages
2. Review metrics for patterns (error rates, durations)
3. File an issue with:
   - Operator version
   - Kubernetes version
   - Relevant CRD manifests
   - Operator logs
   - `kubectl describe` output for affected resources
