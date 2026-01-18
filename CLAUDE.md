# Gas Town Operator - Kubernetes Native Polecats

Kubernetes operator that runs Gas Town polecats as pods. Scale your AI agent army beyond the laptop.

> *"Who runs Bartertown? Kubernetes runs Bartertown."*

---

## CI/CD Images - DPR Registry

**REQUIRED:** All CI pipelines must use images from DPR (DeepSky Private Registry) to avoid Docker Hub rate limits.

### Registry Configuration

```yaml
# In .gitlab-ci.yml variables:
DPR_REGISTRY: "dprusocplvjmp01.deepsky.lab:5000"

# Required image variables:
GO_IMAGE: "${DPR_REGISTRY}/ci-images/golang:1.24"
GO_LINT_IMAGE: "${DPR_REGISTRY}/ci-images/golangci-lint:v2.7.2"
KUBECTL_IMAGE: "${DPR_REGISTRY}/ci-images/kubectl:latest"
YAMLLINT_IMAGE: "${DPR_REGISTRY}/ci-images/yamllint:latest"
```

### Current Issue

The .gitlab-ci.yml uses Docker Hub images directly. **This must be fixed** to use DPR:

```yaml
# WRONG (current):
image: golangci/golangci-lint:v1.62-alpine
image: golang:${GO_VERSION}
image: bitnami/kubectl:latest

# CORRECT (required):
image: ${DPR_REGISTRY}/ci-images/golangci-lint:v1.62-alpine
image: ${DPR_REGISTRY}/ci-images/golang:${GO_VERSION}
image: ${DPR_REGISTRY}/ci-images/kubectl:latest
```

### Mirroring New Images

```bash
# From release_engineering repo:
cd ~/gt/release_engineering/crew/boden
./scripts/mirror-ci-images.sh --check   # See what's missing
./scripts/mirror-ci-images.sh           # Mirror all
```

**Never use Docker Hub images directly in CI** - they will fail with rate limit errors.

---

## Repository Structure

```
gastown-operator/
├── api/v1alpha1/          # CRD types (Polecat, Convoy)
├── internal/controller/   # Reconciliation logic
├── config/                # Kustomize manifests
├── charts/                # Helm chart
└── hack/                  # Build scripts
```

---

## Key Commands

```bash
# Development
make generate              # Generate CRD manifests
make manifests            # Generate RBAC, webhooks
make test                 # Run unit tests

# Build
make docker-build         # Build container image
make docker-push          # Push to registry

# Deploy
make deploy               # Deploy to current cluster
make undeploy             # Remove from cluster
```

---

## CRD Types

| CRD | Purpose |
|-----|---------|
| **Polecat** | AI worker pod (runs Claude in container) |
| **Convoy** | Batch coordination (multiple polecats) |

---

## Two Editions

| Edition | Target | Base Image | Security |
|---------|--------|------------|----------|
| **Community** | Vanilla K8s | golang:alpine | Standard PSS |
| **Enterprise** | OpenShift | Red Hat UBI9 | FIPS + Restricted SCC |

---

## Issue Tracking

Uses [Beads](https://github.com/steveyegge/beads) for git-based issue tracking.

```bash
bd ready                  # Unblocked issues
bd show <id>             # Full context
bd sync && git push      # ALWAYS before stopping
```

---

## Integration with Gas Town

This operator extends the local `gt` CLI to Kubernetes:

```
LOCAL (gt CLI)              CLUSTER (operator)
┌──────────────┐            ┌──────────────────┐
│ gt sling     │ ─────────► │ Creates Polecat  │
│              │            │ CR → Pod runs    │
│ gt convoy    │ ─────────► │ Creates Convoy   │
│              │            │ CR → Coordinates │
└──────────────┘            └──────────────────┘
```

The webhook in this operator receives work assignments from local `gt` CLI and creates the corresponding Kubernetes resources.
