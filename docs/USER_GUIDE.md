# Gas Town Operator User Guide

Run Claude Code agents as Kubernetes pods. Scale your AI agent army beyond the laptop.

## How It Works

The operator uses a **laptop replica** pattern:

```
Your Laptop                          Kubernetes Cluster
┌─────────────────┐                  ┌─────────────────────────────┐
│ claude /login   │                  │     gastown-operator        │
│      ↓          │                  │            ↓                │
│ ~/.claude/      │  ──export───→    │   Secret: claude-home       │
│ (OAuth tokens)  │                  │            ↓                │
└─────────────────┘                  │   Polecat CR → Pod          │
                                     │   ┌─────────────────────┐   │
                                     │   │ git-init (clone)    │   │
                                     │   │ claude (agent)      │   │
                                     │   │  └─ OAuth auth      │   │
                                     │   │  └─ Execute work    │   │
                                     │   └─────────────────────┘   │
                                     └─────────────────────────────┘
```

Pods authenticate using the same OAuth session you get from `claude /login`.

## Prerequisites

- OpenShift/Kubernetes 1.26+
- `oc` or `kubectl` CLI
- Claude Code installed locally (`npm install -g @anthropic-ai/claude-code`)
- Git SSH key for repository access

## Quick Start

### 1. Export Claude Credentials

On macOS, credentials are stored in Keychain:

```bash
# Get your OAuth tokens
security find-generic-password -s "Claude Code-credentials" -w
```

Create the Kubernetes secret:

```bash
CREDS=$(security find-generic-password -s "Claude Code-credentials" -w)
oc create secret generic claude-home -n gastown-workers \
  --from-literal=.credentials.json="$CREDS"
```

### 2. Create Git SSH Secret

```bash
oc create secret generic git-ssh-key -n gastown-workers \
  --from-file=ssh-privatekey=$HOME/.ssh/id_rsa
```

### 3. Deploy the Operator

```bash
# Install CRDs
oc apply -f https://github.com/boshu2/gastown-operator/releases/download/v0.1.2/install.yaml

# Verify
oc get pods -n gastown-system
```

### 4. Create a Polecat

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-worker
  namespace: gastown-workers
spec:
  rig: my-project
  beadID: issue-123
  desiredState: Working
  executionMode: kubernetes
  kubernetes:
    gitRepository: "git@github.com:myorg/myrepo.git"
    gitBranch: main
    gitSecretRef:
      name: git-ssh-key
    claudeCredsSecretRef:
      name: claude-home
    activeDeadlineSeconds: 3600
    resources:
      requests:
        cpu: "500m"
        memory: "1Gi"
      limits:
        cpu: "2"
        memory: "4Gi"
```

```bash
oc apply -f polecat.yaml
```

### 5. Watch It Work

```bash
# Pod status
oc get pods -n gastown-workers

# Claude execution logs
oc logs polecat-my-worker -c claude -n gastown-workers -f

# Polecat status
oc get polecat my-worker -n gastown-workers -o yaml
```

## E2E Proof: It Actually Works

**Tested 2026-01-19 on OpenShift**

### Operator Deployment

```
$ oc get pods -n gastown-system
NAME                                                   READY   STATUS    RESTARTS   AGE
gastown-operator-controller-manager-5dd4dcb775-kxs7g   1/1     Running   0          24h

$ oc get crd | grep gastown
beadstores.gastown.gastown.io    2026-01-17T22:07:04Z
convoys.gastown.gastown.io       2026-01-16T02:27:49Z
polecats.gastown.gastown.io      2026-01-16T02:27:50Z
refineries.gastown.gastown.io    2026-01-17T22:07:05Z
rigs.gastown.gastown.io          2026-01-16T02:27:51Z
witnesses.gastown.gastown.io     2026-01-17T22:07:05Z
```

### Polecat Execution

```
$ oc apply -f polecat-proof-demo.yaml
polecat.gastown.gastown.io/proof-demo created

$ oc get pods -n gastown-workers
NAME                 READY   STATUS    RESTARTS   AGE
polecat-proof-demo   1/1     Running   0          14s

$ oc logs polecat-proof-demo -c git-init -n gastown-workers
Cloning git@github.com:boshu2/gastown-operator.git branch main...
Cloning into '/workspace/repo'...
Switched to a new branch 'feature/demo-proof'
Git setup complete. Working branch: feature/demo-proof

$ oc logs polecat-proof-demo -c claude -n gastown-workers
Claude credentials copied to /home/nonroot/.claude/
Installing Claude Code CLI...
added 3 packages in 4s
2.1.12 (Claude Code)
Starting Claude Code agent...
Working on issue: demo-proof
I've completed the `demo-proof` issue. Here's a summary of the changes made:
## Summary
Fixed Docker Hub rate limit issues by configuring all CI-related files...
### Files Modified
1. **Dockerfile** - Changed default GO_IMAGE to private registry
2. **Dockerfile.tekton** - Changed default GO_IMAGE to private registry
3. **.devcontainer/devcontainer.json** - Added documentation comment
4. **.github/workflows/lint.yml** - Added documentation comment
5. **CLAUDE.md** - Updated configuration section
```

### Final Status

```
$ oc get pod polecat-proof-demo -n gastown-workers
NAME                 READY   STATUS      RESTARTS   AGE
polecat-proof-demo   0/1     Completed   0          2m51s

$ oc get polecat proof-demo -n gastown-workers -o yaml
status:
  assignedBead: demo-proof
  conditions:
  - lastTransitionTime: "2026-01-20T00:10:37Z"
    message: Pod created successfully
    reason: PodCreated
    status: "True"
    type: Ready
  - lastTransitionTime: "2026-01-20T00:13:19Z"
    message: Work completed
    reason: Completed
    status: "False"
    type: Working
  phase: Done
  podName: polecat-proof-demo
```

**Result**: Pod created → Claude authenticated → Work executed → Completed successfully

## Token Refresh

OAuth tokens expire after ~24 hours. To refresh:

```bash
# 1. Re-login on your laptop (opens browser)
claude /login

# 2. Export fresh tokens
CREDS=$(security find-generic-password -s "Claude Code-credentials" -w)

# 3. Update the secret
oc delete secret claude-home -n gastown-workers
oc create secret generic claude-home -n gastown-workers \
  --from-literal=.credentials.json="$CREDS"
```

## Alternative: API Key Authentication

For headless deployments, you can use an Anthropic API key instead of OAuth:

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: api-key-worker
  namespace: gastown-workers
spec:
  rig: my-project
  beadID: issue-456
  desiredState: Working
  executionMode: kubernetes
  kubernetes:
    gitRepository: "git@github.com:myorg/myrepo.git"
    gitBranch: main
    gitSecretRef:
      name: git-ssh-key
    apiKeySecretRef:
      name: anthropic-api-key
      key: api-key
```

Create the API key secret:

```bash
oc create secret generic anthropic-api-key -n gastown-workers \
  --from-literal=api-key="sk-ant-api03-..."
```

## Pod Architecture

Each Polecat pod has:

| Container | Purpose |
|-----------|---------|
| `git-init` (init) | Clone repo, create feature branch |
| `claude` (main) | Run Claude Code agent |

### Security Context

All pods run with OpenShift restricted SCC compliance:

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65532
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
  capabilities:
    drop: ["ALL"]
  seccompProfile:
    type: RuntimeDefault
```

## Troubleshooting

### "OAuth token has expired"

Refresh credentials (see Token Refresh section above).

### Pod stuck in Pending

Check events:
```bash
oc describe pod polecat-<name> -n gastown-workers
```

Common causes:
- Secret doesn't exist
- Insufficient resources
- Node scheduling issues

### Git clone fails

Verify SSH key:
```bash
oc get secret git-ssh-key -n gastown-workers -o jsonpath='{.data.ssh-privatekey}' | base64 -d | head -1
```

Should show: `-----BEGIN OPENSSH PRIVATE KEY-----`

### Claude container exits immediately

Check logs:
```bash
oc logs polecat-<name> -c claude -n gastown-workers
```

Common causes:
- Invalid credentials format
- Network connectivity to Anthropic API
- Missing repository files (CLAUDE.md)

## Reference

- [CRD Reference](./CRD_REFERENCE.md) - Full spec/status documentation
- [Secret Management](./SECRET_MANAGEMENT.md) - Credential setup
- [Architecture](./architecture.md) - How the operator works
- [Development](./development.md) - Contributing guide
