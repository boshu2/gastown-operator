# Secret Management Guide

This guide covers creating, rotating, and managing secrets for the Gas Town Operator.

## Table of Contents

- [Overview](#overview)
- [Git SSH Keys](#git-ssh-keys)
- [Claude Credentials](#claude-credentials)
- [Secret Rotation](#secret-rotation)
- [Monitoring](#monitoring)

---

## Overview

The operator uses two types of secrets:

| Secret Type | Purpose | Used By |
|-------------|---------|---------|
| Git SSH Key | Clone and push to repositories | Polecat, Refinery, BeadStore |
| Claude Credentials | Authenticate to Claude API | Polecat (kubernetes mode) |

Both secrets are referenced by name in CRD specs, allowing different polecats to use different credentials.

---

## Git SSH Keys

### Creating a Git SSH Secret

```bash
# Generate SSH key pair (if needed)
ssh-keygen -t ed25519 -C "gastown-operator" -f ./gastown-git-key -N ""

# Create Kubernetes secret
kubectl create secret generic git-credentials \
  --from-file=ssh-privatekey=./gastown-git-key \
  --from-file=ssh-publickey=./gastown-git-key.pub \
  -n <namespace>

# Clean up local files
rm ./gastown-git-key ./gastown-git-key.pub
```

### Required Secret Keys

The secret must contain at least one of:
- `ssh-privatekey` (standard k8s SSH key format)
- `id_rsa` (common alternative format)

### Adding to Git Provider

Add the public key to your Git provider:

**GitHub**:
1. Go to repository Settings → Deploy keys
2. Add new deploy key with **write access**

**GitLab**:
1. Go to project Settings → Repository → Deploy keys
2. Add new deploy key with **write access**

### Referencing in CRDs

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-polecat
spec:
  kubernetes:
    gitSecretRef:
      name: git-credentials  # Reference by name
```

---

## Claude Credentials

### Creating Claude Credentials Secret

The Claude credentials secret should contain the contents of `~/.claude/`:

```bash
# Create secret from existing Claude config
kubectl create secret generic claude-credentials \
  --from-file=credentials.json=$HOME/.claude/credentials.json \
  --from-file=settings.json=$HOME/.claude/settings.json \
  -n <namespace>
```

### Required Files

| File | Purpose | Required |
|------|---------|----------|
| `credentials.json` | API authentication | Yes |
| `settings.json` | Claude configuration | Optional |

### Referencing in CRDs

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-polecat
spec:
  kubernetes:
    claudeCredsSecretRef:
      name: claude-credentials
```

---

## Secret Rotation

### When to Rotate

| Trigger | Action Required |
|---------|-----------------|
| Suspected compromise | Immediate rotation |
| Team member departure | Rotate if they had access |
| Every 90 days | Recommended rotation interval |
| Before expiry | For time-limited tokens |

### Rotating Git SSH Keys

1. **Generate new key**:
   ```bash
   ssh-keygen -t ed25519 -C "gastown-operator-$(date +%Y%m)" -f ./new-key -N ""
   ```

2. **Add new public key to Git provider** (before removing old)

3. **Update Kubernetes secret**:
   ```bash
   kubectl create secret generic git-credentials-new \
     --from-file=ssh-privatekey=./new-key \
     -n <namespace>
   ```

4. **Update CRDs to use new secret**:
   ```bash
   kubectl patch polecat <name> --type=merge \
     -p '{"spec":{"kubernetes":{"gitSecretRef":{"name":"git-credentials-new"}}}}'
   ```

5. **Verify new credentials work** (create test polecat)

6. **Remove old credentials**:
   ```bash
   # Remove from Git provider
   # Delete old secret
   kubectl delete secret git-credentials -n <namespace>
   # Rename new secret (optional)
   ```

### Rotating Claude Credentials

1. **Generate new credentials via Claude login**:
   ```bash
   claude login
   ```

2. **Create new secret**:
   ```bash
   kubectl create secret generic claude-credentials-new \
     --from-file=credentials.json=$HOME/.claude/credentials.json \
     -n <namespace>
   ```

3. **Update CRDs to use new secret**

4. **Delete old secret**

### Zero-Downtime Rotation Pattern

For production deployments, use a rolling update approach:

1. Create new secret with different name
2. Update polecats in batches to use new secret
3. Verify each batch works before continuing
4. Delete old secret only after all polecats updated

---

## Monitoring

### Checking Secret Usage

```bash
# Find which CRDs reference a secret
kubectl get polecats -A -o jsonpath='{range .items[*]}{.metadata.name}: {.spec.kubernetes.gitSecretRef.name}{"\n"}{end}'
```

### Monitoring for Issues

Watch for these indicators of credential problems:

| Metric/Log | Indicates |
|------------|-----------|
| `gastown_gt_cli_errors_total{command="sling"}` increasing | Git auth failures |
| Polecat `phase: Stuck` | Potential auth issues |
| Pod logs "Permission denied (publickey)" | SSH key problem |
| Pod logs "authentication failed" | Claude auth problem |

### Alerting Recommendations

Set up alerts for:

1. **High error rate**: `rate(gastown_gt_cli_errors_total[5m]) > 0.1`
2. **Stuck polecats**: Polecats in `Working` phase > 30 minutes
3. **Authentication failures**: Match "Permission denied" or "authentication failed" in logs

---

## Security Best Practices

1. **Least privilege**: Use deploy keys with minimal permissions (write access only to specific repos)
2. **Namespace isolation**: Keep secrets in the same namespace as the CRDs that use them
3. **No wildcards**: Reference secrets explicitly by name, don't use patterns
4. **Audit access**: Regularly review who has access to secrets
5. **Encrypt at rest**: Ensure Kubernetes secret encryption is enabled
6. **Avoid sharing**: Each polecat should use its own credentials when possible

---

## Troubleshooting

### "Permission denied (publickey)"

1. Verify secret exists: `kubectl get secret <name> -n <namespace>`
2. Check secret has correct keys: `kubectl get secret <name> -o jsonpath='{.data}' | jq 'keys'`
3. Verify public key is added to Git provider
4. Test SSH manually:
   ```bash
   ssh -i /path/to/key -T git@github.com
   ```

### "authentication failed" for Claude

1. Verify secret exists and has `credentials.json`
2. Check if credentials are expired (re-run `claude login`)
3. Verify credentials format is correct
