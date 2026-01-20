# Gas Town Operator User Guide

Run AI coding agents as Kubernetes pods. Scale your agent army beyond the laptop.

## Supported Agents

The operator supports multiple coding agents:

| Agent | Description | Default |
|-------|-------------|---------|
| `claude-code` | Anthropic's Claude Code CLI | **Yes** |
| `opencode` | Open-source coding agent using LiteLLM | No |
| `aider` | AI pair programming in your terminal | No |
| `custom` | Your own agent implementation | No |

**Default is `claude-code`** - Anthropic's official coding agent.

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

Authentication varies by agent:
- **opencode**: API key via LiteLLM endpoint
- **claude-code**: OAuth session from `claude /login` or API key

## Prerequisites

- OpenShift/Kubernetes 1.26+
- `oc` or `kubectl` CLI
- Git SSH key for repository access

## Quick Start (Claude Code - Default)

### 1. Create Git SSH Secret

```bash
oc create secret generic git-ssh-key -n gastown-workers \
  --from-file=ssh-privatekey=$HOME/.ssh/id_rsa
```

### 2. Create Claude Credentials

**Option A: OAuth Credentials (recommended)**

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

**Option B: API Key (Headless)**

```bash
oc create secret generic anthropic-api-key -n gastown-workers \
  --from-literal=api-key="sk-ant-api03-..."
```

### 3. Deploy the Operator

```bash
# Install CRDs
oc apply -f https://github.com/boshu2/gastown-operator/releases/download/v0.1.2/install.yaml

# Verify
oc get pods -n gastown-system
```

### 4. Create a Polecat

With OAuth credentials and explicit task description:

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: my-worker
  namespace: gastown-workers
spec:
  rig: my-project
  beadID: issue-123
  # taskDescription provides explicit instructions when beads aren't synced to repo
  taskDescription: |
    Add a /health endpoint that returns {"status": "ok"}.
    After implementing, commit and push the changes.
  desiredState: Working
  executionMode: kubernetes
  # agent: claude-code  # default
  kubernetes:
    gitRepository: "git@github.com:myorg/myrepo.git"
    gitBranch: main
    workBranch: feature/issue-123  # auto-generated if not set
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

**Note**: The `taskDescription` field is optional but recommended when beads aren't synced to the target repository. Claude will use this description to understand the task.

### Git Push Capability

Polecats can push commits to remote repositories. The git credentials are mounted in the claude container, allowing:
- `git commit` with configured user (Gas Town Polecat)
- `git push origin HEAD` to push the work branch
- PR creation (if `gh` CLI is available)

Or with API key (headless):

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
    apiKeySecretRef:
      name: anthropic-api-key
      key: api-key
```

```bash
oc apply -f polecat.yaml
```

---

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

### Test 2: PR Creation (2026-01-20)

**Full end-to-end test with git push and PR creation:**

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: feature-version-endpoint
  namespace: gastown
spec:
  rig: test-rig
  beadID: go-cwl
  taskDescription: |
    Add a /version endpoint to the operator webhook server.
    The endpoint should return JSON with version, commit, and buildTime.
  desiredState: Working
  executionMode: kubernetes
  agent: claude-code
  kubernetes:
    gitRepository: "git@github.com:boshu2/gastown-operator.git"
    gitBranch: main
    workBranch: feature/go-cwl-version-endpoint
    gitSecretRef:
      name: git-credentials
    claudeCredsSecretRef:
      name: claude-credentials
```

**Result:**
```
$ oc get polecat feature-version-endpoint -n gastown
NAME                       RIG        MODE         AGENT         PHASE   BEAD
feature-version-endpoint   test-rig   kubernetes   claude-code   Done    go-cwl

$ oc logs polecat-feature-version-endpoint -c claude --tail=30
...
Git SSH key configured
...
The task is complete.

**Changes made:**
1. Created `pkg/version/version.go` - version info with HTTP handler
2. Modified `cmd/main.go` - register /version endpoint

**Git operations:**
- Committed: feat(go-cwl): add /version endpoint to webhook server
- Pushed to: feature/go-cwl-version-endpoint branch
```

**PR Created:** https://github.com/boshu2/gastown-operator/pull/1

---

### Test 1: Basic Execution (2026-01-20)

### Operator Deployment

```
$ oc get pods -n gastown-system
NAME                                                   READY   STATUS    RESTARTS   AGE
gastown-operator-controller-manager-5dd4dcb775-kxs7g   1/1     Running   0          48h

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
$ cat polecat-claude-test.yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
metadata:
  name: claude-test
  namespace: gastown-workers
spec:
  rig: gastown-operator
  beadID: doc-test
  desiredState: Working
  executionMode: kubernetes
  agent: claude-code
  kubernetes:
    gitRepository: "git@github.com:boshu2/gastown-operator.git"
    gitBranch: main
    workBranch: feature/claude-test
    gitSecretRef:
      name: git-ssh-key
    claudeCredsSecretRef:
      name: claude-home
    activeDeadlineSeconds: 600

$ oc apply -f polecat-claude-test.yaml
polecat.gastown.gastown.io/claude-test created

$ oc get pods -n gastown-workers
NAME                  READY   STATUS    RESTARTS   AGE
polecat-claude-test   1/1     Running   0          24s

$ oc logs polecat-claude-test -c git-init -n gastown-workers
Cloning git@github.com:boshu2/gastown-operator.git branch main...
Cloning into '/workspace/repo'...
Switched to a new branch 'feature/claude-test'
Git setup complete. Working branch: feature/claude-test

$ oc logs polecat-claude-test -c claude -n gastown-workers
Claude credentials copied to /home/nonroot/.claude/
Installing Claude Code CLI...
added 3 packages in 3s
2.1.12 (Claude Code)
Starting Claude Code agent...
Working on issue: doc-test
The implementation is complete. Let me provide a summary of the work done for the `doc-test` issue.

## Summary

I've implemented documentation testing for the gastown-operator project. Here's what was done:

### Files Created
- **`test/doctest/doc_test.go`** - New test file that validates YAML examples in documentation

### Files Modified
- **`Makefile`** - Added `test-docs` target and included it in `validate` target
- **`.gitlab-ci.yml`** - Added `validate:docs` job to CI pipeline
- **`docs/development.md`** - Added `test-docs` to the Makefile targets table
- **`CONTRIBUTING.md`** - Updated testing section to include doc tests
- **`config/samples/gastown_v1alpha1_rig.yaml`** - Fixed sample with complete spec fields
- **`config/samples/gastown_v1alpha1_convoy.yaml`** - Fixed sample with complete spec fields

### Features of the doc-test
1. **TestDocumentationYAML** - Extracts and validates all YAML code blocks from markdown files
2. **TestSampleYAMLFiles** - Validates all YAML files in `config/samples/`
3. **TestYAMLExamplesHaveRequiredFields** - Checks that YAML examples include required CRD fields
```

### Final Status

```
$ oc get pod polecat-claude-test -n gastown-workers
NAME                  READY   STATUS      RESTARTS   AGE
polecat-claude-test   0/1     Completed   0          5m51s

$ oc get polecat claude-test -n gastown-workers -o yaml
status:
  assignedBead: doc-test
  conditions:
  - lastTransitionTime: "2026-01-20T02:08:18Z"
    message: Pod created successfully
    reason: PodCreated
    status: "True"
    type: Ready
  - lastTransitionTime: "2026-01-20T02:08:18Z"
    message: Polecat is working on assigned bead
    reason: Working
    status: "True"
    type: Working
  lastActivity: "2026-01-20T02:08:18Z"
  phase: Working
  podName: polecat-claude-test
  sessionActive: true
```

**Result**: Pod created → Git cloned → Claude Code installed → Work executed → Completed successfully

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
