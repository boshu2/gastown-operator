# Security Guide

This document covers security considerations, threat models, and best practices for the Gas Town Operator.

## Table of Contents

- [Overview](#overview)
- [Claude CLI Permissions](#claude-cli-permissions)
- [SSH Host Key Verification](#ssh-host-key-verification)
- [RBAC Recommendations](#rbac-recommendations)
- [Pod Security](#pod-security)
- [Secret Management](#secret-management)
- [Threat Model](#threat-model)

---

## Overview

The Gas Town Operator runs AI coding agents in Kubernetes pods. This creates a unique security context:

1. **AI agents execute arbitrary code** - The agent is designed to write and execute code
2. **Agents need Git write access** - To push changes back to repositories
3. **Headless operation** - Agents run without human approval of individual actions

This guide helps you understand and mitigate the associated risks.

---

## Claude CLI Permissions

### The `--dangerously-skip-permissions` Flag

**Location**: `pkg/pod/builder.go:374`

The operator runs Claude CLI with the `--dangerously-skip-permissions` flag:

```bash
exec claude --print --dangerously-skip-permissions "$PROMPT"
```

### Why This Flag Is Required

Claude Code CLI has a built-in permission system that prompts users before:
- Reading files outside the current directory
- Writing files
- Executing shell commands
- Making network requests

In interactive mode, users approve each action. In headless Kubernetes execution, there's no terminal to display prompts, so the flag bypasses this permission system.

### What Permissions Are Granted

With `--dangerously-skip-permissions`, the Claude agent can:

| Permission | Description | Risk Level |
|------------|-------------|------------|
| File Read | Read any file the container user can access | Medium |
| File Write | Write/modify any writable path | High |
| Shell Exec | Execute arbitrary shell commands | High |
| Network | Make outbound network requests | Medium |

### Mitigation Strategies

Since we cannot use interactive permissions, we mitigate risk through:

1. **Container isolation** (see [Pod Security](#pod-security))
   - Read-only root filesystem
   - Non-root user
   - Dropped capabilities
   - Limited filesystem mounts

2. **Network policies** (recommended)
   - Restrict outbound traffic to Git servers only
   - Block access to cloud metadata endpoints

3. **Resource limits**
   - CPU/memory limits prevent resource exhaustion
   - Active deadline terminates runaway agents

4. **Git-level controls**
   - Branch protection rules
   - Required reviews before merge
   - Signed commits (if configured)

5. **RBAC** (see [RBAC Recommendations](#rbac-recommendations))
   - Limit which namespaces can create polecats
   - Restrict access to credential secrets

### Code Comment

A security comment is added at the flag usage site in the Pod builder to ensure future maintainers understand this decision:

```go
// SECURITY: --dangerously-skip-permissions is required for headless operation.
// This grants elevated privileges to the Claude agent. Mitigations:
// - Pod runs as non-root with read-only root filesystem
// - Network policies should restrict outbound traffic
// - RBAC should limit polecat creation to trusted namespaces
// See docs/SECURITY.md for full threat model.
```

---

## SSH Host Key Verification

### Default Behavior

The operator uses **pre-verified SSH host keys** for common Git providers (GitHub, GitLab, Bitbucket). These keys are compiled into the operator and verified against official documentation:

- GitHub: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/githubs-ssh-key-fingerprints
- GitLab: https://docs.gitlab.com/ee/user/gitlab_com/index.html#ssh-host-keys-fingerprints
- Bitbucket: https://support.atlassian.com/bitbucket-cloud/docs/configure-ssh-and-two-step-verification/

This prevents MITM (Man-in-the-Middle) attacks on first connection (TOFU vulnerability).

### Configuration Options

#### SSHStrictHostKeyChecking

Controls SSH host key verification behavior:

| Value | Behavior | Security Level |
|-------|----------|----------------|
| `yes` (default) | Only connect to hosts in known_hosts | Highest |
| `accept-new` | Accept and save new host keys, reject changed keys | Medium |
| `no` | Accept any key (**NOT RECOMMENDED**) | Low |

```yaml
spec:
  kubernetes:
    sshStrictHostKeyChecking: "yes"  # Default, most secure
```

#### SSHKnownHostsConfigMapRef

For private Git servers not in the pre-verified list, provide your own known_hosts:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: my-git-known-hosts
data:
  known_hosts: |
    git.internal.example.com ssh-ed25519 AAAAC3Nza...
---
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
spec:
  kubernetes:
    sshKnownHostsConfigMapRef:
      name: my-git-known-hosts
```

### Security Recommendation

1. Always use `sshStrictHostKeyChecking: yes` (default)
2. For private Git servers, create a ConfigMap with verified host keys
3. Never use `sshStrictHostKeyChecking: no` in production

---

## RBAC Recommendations

### Namespace Isolation

Create dedicated namespaces for polecat workloads:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: gastown-polecats
  labels:
    pod-security.kubernetes.io/enforce: restricted
```

### Operator RBAC

The operator needs cluster-wide permissions for CRD management but should be limited in what it can do with secrets and pods:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: gastown-operator
rules:
  # CRD access (cluster-scoped)
  - apiGroups: ["gastown.gastown.io"]
    resources: ["*"]
    verbs: ["*"]
  # Limited pod/secret access (in operator namespace only)
  - apiGroups: [""]
    resources: ["pods", "secrets"]
    verbs: ["get", "list", "watch", "create", "update", "delete"]
```

### User RBAC

Limit which users/service accounts can create polecats:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: polecat-creator
  namespace: gastown-polecats
rules:
  - apiGroups: ["gastown.gastown.io"]
    resources: ["polecats"]
    verbs: ["create", "get", "list", "watch"]
  # Explicitly deny access to secrets
```

### Service Account per Polecat (Advanced)

For maximum isolation, each polecat can use its own service account with access to only its required secrets:

```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: polecat-myproject
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: polecat-myproject-secrets
rules:
  - apiGroups: [""]
    resources: ["secrets"]
    resourceNames: ["myproject-git-key", "myproject-claude-creds"]
    verbs: ["get"]
```

---

## Pod Security

### Security Context

All polecat pods run with restricted security context:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532        # nonroot user
  runAsGroup: 65532
  fsGroup: 65532
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

### What This Provides

| Control | Protection |
|---------|------------|
| `runAsNonRoot` | Prevents container breakout via root |
| `readOnlyRootFilesystem` | Limits persistence of malicious changes |
| `capabilities: drop ALL` | Removes dangerous Linux capabilities |
| `seccompProfile: RuntimeDefault` | Syscall filtering |
| `allowPrivilegeEscalation: false` | Prevents setuid/sudo |

### Network Policies (Recommended)

Restrict polecat network access:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: polecat-egress
  namespace: gastown-polecats
spec:
  podSelector:
    matchLabels:
      gastown.io/polecat: ""
  policyTypes:
    - Egress
  egress:
    # Allow Git SSH (adjust IPs for your Git provider)
    - to:
        - ipBlock:
            cidr: 140.82.112.0/20  # GitHub
      ports:
        - port: 22
          protocol: TCP
    # Allow HTTPS for Claude API
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - port: 443
          protocol: TCP
    # Block cloud metadata endpoints
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
            except:
              - 169.254.169.254/32  # AWS/GCP metadata
              - 100.100.100.200/32  # Azure metadata
```

---

## Secret Management

See [SECRET_MANAGEMENT.md](SECRET_MANAGEMENT.md) for detailed guidance on:

- Creating Git SSH keys
- Managing Claude credentials
- Secret rotation procedures
- Monitoring for credential issues

### Key Security Practices

1. **Use separate credentials per repository** when possible
2. **Deploy keys with minimal scope** (single repo, no admin)
3. **Rotate credentials regularly** (every 90 days recommended)
4. **Enable Kubernetes secret encryption at rest**
5. **Audit secret access** via Kubernetes audit logs

---

## Threat Model

### Assets

| Asset | Value | Location |
|-------|-------|----------|
| Source code | High | Git repositories |
| Git credentials | High | Kubernetes secrets |
| Claude API credentials | Medium | Kubernetes secrets |
| Build artifacts | Medium | Container registry |

### Threat Actors

| Actor | Motivation | Capability |
|-------|------------|------------|
| Malicious prompt | Code injection | Craft prompts that manipulate agent |
| Compromised dependency | Supply chain | Agent pulls malicious packages |
| Insider threat | Data exfiltration | Authorized user misuses access |
| Network attacker | Credential theft | MITM, DNS poisoning |

### Attack Vectors & Mitigations

| Attack | Vector | Mitigation |
|--------|--------|------------|
| Prompt injection | Malicious issue/PR description | Review agent output, branch protection |
| Credential theft | Pod compromise | Non-root, read-only FS, network policy |
| Code exfiltration | Outbound network | Network policy, audit logs |
| Supply chain | npm/pip install | Use private registries, lock files |
| MITM on Git | First SSH connection | Pre-verified host keys |

### Residual Risks

The following risks are accepted when using AI coding agents:

1. **Agent may write vulnerable code** - Mitigate with code review, security scanning
2. **Agent may expose secrets in logs** - Mitigate with log filtering, secret detection
3. **Agent may install malicious packages** - Mitigate with dependency scanning, private registries

---

## Incident Response

### Suspected Credential Compromise

1. **Revoke immediately**: Remove deploy key from Git provider, delete Kubernetes secret
2. **Audit**: Check Git history for unauthorized commits
3. **Rotate**: Generate new credentials following [Secret Management](SECRET_MANAGEMENT.md)
4. **Review**: Check polecat logs for suspicious activity

### Suspected Malicious Agent Behavior

1. **Terminate**: Delete the polecat resource
2. **Isolate**: Apply network policy to block all egress
3. **Preserve**: Capture pod logs before deletion
4. **Investigate**: Review agent conversation logs, Git commits
5. **Report**: Follow your security incident process

---

## Compliance Considerations

### SOC 2

| Control | Implementation |
|---------|----------------|
| Access control | RBAC, namespace isolation |
| Encryption | Secrets encrypted at rest |
| Logging | Kubernetes audit logs, pod logs |
| Change management | Git branch protection, PR reviews |

### HIPAA / PCI-DSS

If processing sensitive data, additional controls may be required:
- Dedicated cluster/namespace for polecats
- Enhanced network isolation
- Data loss prevention scanning on Git pushes
- Additional audit logging

Consult your compliance team before using AI coding agents with regulated data.
