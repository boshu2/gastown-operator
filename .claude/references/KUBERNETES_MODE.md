# Kubernetes Execution Mode

Complete guide for running polecats as Kubernetes pods.

---

## Overview

When `executionMode: kubernetes`, the operator:
1. Creates a Pod in `gastown-workers` namespace
2. Clones the git repository
3. Runs the configured agent (claude-code, opencode, aider)
4. Agent works on the task
5. Commits and pushes to work branch
6. Pod terminates, polecat status updates

---

## Required Secrets

### Git SSH Key

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-credentials
  namespace: gastown-workers
type: Opaque
stringData:
  ssh-privatekey: |
    -----BEGIN OPENSSH PRIVATE KEY-----
    ...your private key...
    -----END OPENSSH PRIVATE KEY-----
```

**Create from file:**
```bash
kubectl create secret generic git-credentials -n gastown-workers \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519
```

**Requirements:**
- Key must be added to GitHub/GitLab as deploy key or user key
- For private repos, needs read/write access
- Ed25519 or RSA format

### Claude Credentials

**Option A: API Key (recommended for automation)**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: claude-credentials
  namespace: gastown-workers
type: Opaque
stringData:
  api-key: sk-ant-api03-...
```

```bash
kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=api-key=$ANTHROPIC_API_KEY
```

**Option B: OAuth Credentials (from claude login)**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: claude-credentials
  namespace: gastown-workers
type: Opaque
stringData:
  .credentials.json: |
    {"oauth_token":"...","refresh_token":"...","expires_at":"..."}
```

```bash
# macOS - extract from Keychain
CREDS=$(security find-generic-password -s "Claude Code-credentials" -w)
kubectl create secret generic claude-credentials -n gastown-workers \
  --from-literal=.credentials.json="$CREDS"
```

**Note:** OAuth tokens expire in 24 hours. API keys are preferred for long-running automation.

---

## Full Polecat Spec for Kubernetes

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: implement-feature-x
  namespace: gastown-system
  labels:
    app.kubernetes.io/managed-by: gastown-operator
spec:
  # Parent rig (must exist)
  rig: athena

  # Task assignment
  beadID: "at-1234"
  taskDescription: |
    Implement the /health endpoint that returns JSON:
    {"status": "ok", "timestamp": "<iso8601>"}

    Requirements:
    - Add endpoint to main router
    - Include unit tests
    - Update API documentation

  # Execution config
  desiredState: Working
  executionMode: kubernetes

  # Agent selection
  agent: claude-code  # or: opencode, aider, custom

  # Agent configuration
  agentConfig:
    provider: anthropic
    model: claude-sonnet-4-20250514
    maxTokens: 8192
    temperature: 0.7

  # Kubernetes-specific config
  kubernetes:
    # Git repository (SSH URL for private repos)
    gitRepository: "git@github.com:myorg/myrepo.git"

    # Base branch to start from
    gitBranch: main

    # Branch for polecat's changes
    workBranch: polecat/implement-feature-x

    # Git credentials secret
    gitSecretRef:
      name: git-credentials

    # Claude credentials (choose one)
    apiKeySecretRef:
      name: claude-credentials
      key: api-key
    # OR
    # claudeCredsSecretRef:
    #   name: claude-credentials

    # REQUIRED: Timeout to prevent runaway costs
    activeDeadlineSeconds: 3600  # 1 hour

    # Resource allocation
    resources:
      requests:
        cpu: "500m"
        memory: "1Gi"
      limits:
        cpu: "2"
        memory: "4Gi"

    # Optional: Custom image
    # image: ghcr.io/myorg/custom-agent:v1.0.0

    # Optional: Environment variables
    # env:
    #   - name: CUSTOM_VAR
    #     value: "value"

    # Optional: Node selector
    # nodeSelector:
    #   kubernetes.io/arch: amd64

    # Optional: Tolerations
    # tolerations:
    #   - key: "dedicated"
    #     operator: "Equal"
    #     value: "ai-workloads"
    #     effect: "NoSchedule"
```

---

## Pod Lifecycle

### 1. Init Container: git-clone

Clones repository and checks out work branch:
```
git clone --depth=1 -b main <repo>
git checkout -b polecat/<name>
```

### 2. Main Container: agent

Runs the configured agent:
- **claude-code**: `claude --task "<description>" --branch polecat/<name>`
- **opencode**: `opencode --task "<description>"`
- **aider**: `aider --yes --message "<description>"`

### 3. Sidecar: git-push (optional)

Monitors for commits and pushes:
```
while true; do
  git push origin polecat/<name>
  sleep 60
done
```

---

## Resource Guidelines

| Workload | CPU Request | Memory Request | CPU Limit | Memory Limit |
|----------|-------------|----------------|-----------|--------------|
| Light (docs, config) | 250m | 512Mi | 1 | 2Gi |
| Medium (features) | 500m | 1Gi | 2 | 4Gi |
| Heavy (refactoring) | 1 | 2Gi | 4 | 8Gi |

**Timeout Guidelines:**
| Task Type | activeDeadlineSeconds |
|-----------|----------------------|
| Quick fix | 1800 (30 min) |
| Feature | 3600 (1 hour) |
| Large refactor | 7200 (2 hours) |
| Never exceed | 14400 (4 hours) |

---

## Monitoring

### Check Pod Status

```bash
# List polecat pods
kubectl get pods -n gastown-workers -l app.kubernetes.io/managed-by=gastown-operator

# Describe specific pod
kubectl describe pod -n gastown-workers -l polecat=<name>

# Get events
kubectl get events -n gastown-workers --sort-by='.lastTimestamp' | grep <name>
```

### View Logs

```bash
# All containers
kubectl logs -n gastown-workers -l polecat=<name> --all-containers -f

# Init container (git clone)
kubectl logs -n gastown-workers -l polecat=<name> -c git-clone

# Main container (agent)
kubectl logs -n gastown-workers -l polecat=<name> -c agent
```

### Check Git Progress

```bash
# Exec into pod
kubectl exec -it -n gastown-workers -l polecat=<name> -- /bin/sh

# Inside pod
git log --oneline -5
git status
git diff
```

---

## Cleanup

### Terminate Polecat

```bash
kubectl patch polecat <name> -n gastown-system \
  --type=merge -p '{"spec":{"desiredState":"Terminated"}}'
```

### Delete Polecat (and Pod)

```bash
kubectl delete polecat <name> -n gastown-system
```

### Clean Orphaned Pods

```bash
kubectl delete pods -n gastown-workers -l app.kubernetes.io/managed-by=gastown-operator \
  --field-selector=status.phase=Succeeded

kubectl delete pods -n gastown-workers -l app.kubernetes.io/managed-by=gastown-operator \
  --field-selector=status.phase=Failed
```

---

## Security Considerations

1. **Secrets**: Never log or expose secret values
2. **Network**: Pods need egress to git provider and LLM API
3. **RBAC**: Pods run with minimal permissions
4. **Isolation**: Each polecat runs in separate pod
5. **Timeout**: Always set `activeDeadlineSeconds` to prevent cost overruns

### Network Policy (Recommended)

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: polecat-egress
  namespace: gastown-workers
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/managed-by: gastown-operator
  policyTypes:
    - Egress
  egress:
    - to:
        - ipBlock:
            cidr: 0.0.0.0/0
      ports:
        - protocol: TCP
          port: 443  # HTTPS (git, API)
        - protocol: TCP
          port: 22   # SSH (git)
```
