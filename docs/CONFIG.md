# Configuration

Configuration reference for the Gas Town Operator.

---

## Operator Flags

Command-line flags for the controller manager:

| Flag | Default | Description |
|------|---------|-------------|
| `--metrics-bind-address` | `0` | Metrics endpoint address. Use `:8443` for HTTPS, `:8080` for HTTP, or `0` to disable |
| `--health-probe-bind-address` | `:8081` | Health probe endpoint address |
| `--leader-elect` | `false` | Enable leader election for HA deployments |
| `--metrics-secure` | `true` | Serve metrics over HTTPS |
| `--webhook-cert-path` | - | Directory containing webhook TLS certificate |
| `--webhook-cert-name` | `tls.crt` | Webhook certificate filename |
| `--webhook-cert-key` | `tls.key` | Webhook key filename |
| `--metrics-cert-path` | - | Directory containing metrics server TLS certificate |
| `--metrics-cert-name` | `tls.crt` | Metrics certificate filename |
| `--metrics-cert-key` | `tls.key` | Metrics key filename |
| `--enable-http2` | `false` | Enable HTTP/2 for metrics and webhook servers |
| `--zap-devel` | `true` | Development mode logging (human-readable) |
| `--zap-log-level` | `info` | Log level (debug, info, error) |

---

## Environment Variables

| Variable | Description |
|----------|-------------|
| `KUBECONFIG` | Path to kubeconfig file (for out-of-cluster operation) |
| `WATCH_NAMESPACE` | Namespace to watch (empty = all namespaces) |

---

## Resource Requirements

Default resources for the controller manager:

```yaml
resources:
  limits:
    cpu: 500m
    memory: 128Mi
  requests:
    cpu: 10m
    memory: 64Mi
```

For production deployments with many Polecats, consider increasing memory:

```yaml
resources:
  limits:
    cpu: "1"
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

---

## RBAC Configuration

The operator requires specific RBAC permissions. Pre-defined roles are available:

### ClusterRoles

| Role | Purpose | Permissions |
|------|---------|-------------|
| `rig-admin` | Full Rig management | create, delete, get, list, patch, update, watch |
| `rig-editor` | Modify Rigs | get, list, patch, update, watch |
| `rig-viewer` | View Rigs | get, list, watch |
| `polecat-admin` | Full Polecat management | create, delete, get, list, patch, update, watch |
| `polecat-editor` | Modify Polecats | get, list, patch, update, watch |
| `polecat-viewer` | View Polecats | get, list, watch |
| `convoy-admin` | Full Convoy management | create, delete, get, list, patch, update, watch |
| `convoy-editor` | Modify Convoys | get, list, patch, update, watch |
| `convoy-viewer` | View Convoys | get, list, watch |

### Controller Permissions

The controller manager ServiceAccount requires:

```yaml
# Core API
- apiGroups: [""]
  resources: [events]
  verbs: [create, patch]
- apiGroups: [""]
  resources: [pods]
  verbs: [create, delete, get, list, patch, update, watch]
- apiGroups: [""]
  resources: [secrets]
  verbs: [get, list, watch]

# Gas Town CRDs
- apiGroups: [gastown.gastown.io]
  resources: [rigs, polecats, convoys, witnesses, refineries, beadstores]
  verbs: [create, delete, get, list, patch, update, watch]
- apiGroups: [gastown.gastown.io]
  resources: [*/status]
  verbs: [get, patch, update]
- apiGroups: [gastown.gastown.io]
  resources: [*/finalizers]
  verbs: [update]
```

---

## Secrets Configuration

### Git Credentials Secret

For Polecats running in Kubernetes mode:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: git-creds
  namespace: gastown-system
type: kubernetes.io/ssh-auth
data:
  # Base64-encoded SSH private key
  # Generate with: cat ~/.ssh/id_rsa | base64 -w0
  ssh-privatekey: <base64-encoded-ssh-private-key>
```

### Claude Credentials Secret

For Polecats to authenticate with Claude:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: claude-creds
  namespace: gastown-system
type: Opaque
data:
  # Base64-encoded contents of ~/.claude/ files
  # Generate with:
  #   kubectl create secret generic claude-creds \
  #     --from-file=credentials.json=$HOME/.claude/credentials.json \
  #     --from-file=settings.json=$HOME/.claude/settings.json \
  #     --dry-run=client -o yaml
  credentials.json: <base64-encoded-credentials>
  settings.json: <base64-encoded-settings>
```

---

## CRD Defaults

### Polecat Defaults

| Field | Default | Notes |
|-------|---------|-------|
| `desiredState` | `Idle` | Initial state |
| `executionMode` | `local` | Run via tmux |
| `kubernetes.gitBranch` | `main` | Base branch |
| `kubernetes.activeDeadlineSeconds` | `3600` | 1 hour max runtime |

### Rig Defaults

| Field | Default | Notes |
|-------|---------|-------|
| `settings.maxPolecats` | `8` | Per-rig limit |

### Convoy Defaults

| Field | Default | Notes |
|-------|---------|-------|
| `parallelism` | `0` | Unlimited |

### Witness Defaults

| Field | Default | Notes |
|-------|---------|-------|
| `healthCheckInterval` | `30s` | Check frequency |
| `stuckThreshold` | `15m` | Idle timeout |
| `escalationTarget` | `mayor` | Alert destination |

### Refinery Defaults

| Field | Default | Notes |
|-------|---------|-------|
| `targetBranch` | `main` | Merge target |
| `parallelism` | `1` | Sequential merges |

### BeadStore Defaults

| Field | Default | Notes |
|-------|---------|-------|
| `syncInterval` | `5m` | Git sync frequency |

---

## High Availability

For HA deployments, enable leader election:

```yaml
spec:
  replicas: 2
  template:
    spec:
      containers:
      - name: manager
        args:
        - --leader-elect=true
```

---

## Security Configuration

### Pod Security

The operator runs with restricted Pod Security Standards:

```yaml
securityContext:
  runAsNonRoot: true
  seccompProfile:
    type: RuntimeDefault
  containers:
  - securityContext:
      readOnlyRootFilesystem: true
      allowPrivilegeEscalation: false
      capabilities:
        drop:
        - "ALL"
```

### Network Policy

Recommended NetworkPolicy for the operator:

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: gastown-operator
  namespace: gastown-system
spec:
  podSelector:
    matchLabels:
      control-plane: controller-manager
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - ports:
    - port: 8081  # Health probes
  egress:
  - to: []  # Allow egress to API server
```

---

## Monitoring

### Metrics

Prometheus metrics are available at the metrics endpoint (default disabled, set `--metrics-bind-address=:8443`).

Available metrics:
- Standard controller-runtime metrics
- Workqueue metrics
- Reconciliation latency

### Health Endpoints

| Endpoint | Port | Purpose |
|----------|------|---------|
| `/healthz` | 8081 | Liveness probe |
| `/readyz` | 8081 | Readiness probe |

---

## Logging

Configure logging with zap flags:

```bash
# Development mode (human-readable)
--zap-devel=true

# Production mode (JSON)
--zap-devel=false

# Debug level
--zap-log-level=debug
```
