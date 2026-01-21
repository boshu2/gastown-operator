# Gastown Operator - CI/CD Deployment Guide

This document describes the complete CI/CD pipeline from local development to production release.

## Architecture Overview

```mermaid
flowchart TB
    subgraph LOCAL["🖥️ Local Development"]
        DEV[Developer Machine]
        HOOKS[Git Hooks]
        DEV -->|git commit| HOOKS
        HOOKS -->|pre-commit| PC[Format + Vet]
        HOOKS -->|pre-push| PP[Lint + Test + Build]
    end

    subgraph GITLAB["🦊 GitLab (Behind VPN)"]
        MR[Merge Request]
        MAIN[main branch]
        VALIDATE[Validate Stage]
        VERSION[semantic-release]

        PP -->|push| MR
        MR -->|merge| MAIN
        MAIN --> VALIDATE
        VALIDATE --> VERSION
    end

    subgraph TEKTON["⚙️ Tekton Pipeline (olympus-ci)"]
        CLONE[Clone]
        SCAN[7 Security Scans]
        BUILD[Build Image]
        SIGN[Sign + SBOM]
        INTERNAL[Push to Internal Registry]

        VERSION -->|trigger| CLONE
        CLONE --> SCAN
        SCAN --> BUILD
        BUILD --> SIGN
        SIGN --> INTERNAL
    end

    subgraph TEST["🧪 E2E Testing"]
        UPGRADE[Helm Upgrade Test<br/>gastown-test-upgrade]
        FRESH[Fresh Install Test<br/>gastown-test-fresh]
        E2E[Smoke Tests]

        INTERNAL --> UPGRADE
        INTERNAL --> FRESH
        UPGRADE --> E2E
        FRESH --> E2E
    end

    subgraph PUBLISH["📦 Publish (Only After Tests Pass)"]
        GHCR[Push to GHCR]
        HELM[Push Helm Chart]
        RELEASE[GitHub Release]

        E2E -->|pass| GHCR
        GHCR --> HELM
        HELM --> RELEASE
    end

    subgraph GITHUB["🐙 GitHub (Public Storefront)"]
        GH_CODE[Code Mirror]
        GH_RELEASE[Release Page]
        GH_GHCR[ghcr.io Images]

        RELEASE --> GH_CODE
        RELEASE --> GH_RELEASE
        GHCR --> GH_GHCR
    end

    subgraph MANUAL["👤 Manual Deployment"]
        DEPLOY_INT[deploy:internal<br/>gastown-system]
    end

    E2E -->|manual trigger| DEPLOY_INT
```

## Pipeline Stages in Detail

### Stage 1: Local Development

```mermaid
flowchart LR
    subgraph PRECOMMIT["pre-commit hook"]
        F1[gofmt check]
        F2[goimports check]
        F3[go vet]
        F4[TODO/debug scan]
        F1 --> F2 --> F3 --> F4
    end

    subgraph PREPUSH["pre-push hook"]
        P1[go vet]
        P2[golangci-lint]
        P3[make manifests generate]
        P4[gofmt verify]
        P5[go build]
        P6[go test ./internal/...]
        P1 --> P2 --> P3 --> P4 --> P5 --> P6
    end

    COMMIT[git commit] --> PRECOMMIT
    PRECOMMIT -->|pass| STAGED[Committed]
    PUSH[git push] --> PREPUSH
    PREPUSH -->|pass| REMOTE[Pushed to GitLab]
```

**Local hooks ensure:**
- Code is properly formatted
- No obvious bugs (go vet)
- Linting passes
- Generated code is up-to-date
- Code compiles
- Unit tests pass

**Bypass (emergency only):**
```bash
SKIP_TESTS=1 git push  # Skip unit tests
git push --no-verify   # Skip all hooks (not recommended)
```

### Stage 2: GitLab CI Validation

```mermaid
flowchart TB
    subgraph VALIDATE["validate stage (parallel)"]
        LINT_GO[lint:go<br/>golangci-lint]
        LINT_YAML[lint:yaml<br/>yamllint]
        VAL_MAN[validate:manifests<br/>make manifests]
        VAL_GEN[validate:generate<br/>make generate]
    end

    MR[MR or main push] --> VALIDATE
    VALIDATE --> NEXT{All pass?}
    NEXT -->|yes| VERSION[version stage]
    NEXT -->|no| FAIL[Pipeline Failed]
```

**Runs on:** MRs and main branch pushes
**Purpose:** Catch issues that slipped past local hooks

### Stage 3: Semantic Release

```mermaid
flowchart LR
    COMMITS[Analyze Commits] --> DECIDE{Release needed?}
    DECIDE -->|feat:/fix:| BUMP[Version Bump]
    DECIDE -->|chore:/docs:| SKIP[Skip Release]
    BUMP --> TAG[Create Git Tag]
    TAG --> ENV[Export VERSION env]
```

**Commit types:**
| Prefix | Version Bump | Example |
|--------|--------------|---------|
| `feat:` | Minor (0.1.0 → 0.2.0) | New feature |
| `fix:` | Patch (0.1.0 → 0.1.1) | Bug fix |
| `feat!:` | Major (0.1.0 → 1.0.0) | Breaking change |
| `chore:` | None | Maintenance |

### Stage 4: Tekton Pipeline

```mermaid
flowchart TB
    subgraph PARALLEL["Stage 2: Parallel Scans"]
        TEST[go-test]
        GOVULN[govulncheck]
        GOSEC[gosec]
        GITLEAKS[gitleaks]
        TRIVY_FS[trivy-fs]
        HADOLINT[hadolint]
        VIBE[vibe-prescan]
    end

    CLONE[1. Clone] --> PARALLEL
    PARALLEL --> GT[3. Build gt CLI]
    GT --> BUILD[4. Build Image]
    BUILD --> SCAN_IMG[5. Trivy Image Scan]
    SCAN_IMG --> SBOM[6. Generate SBOM]
    SBOM --> SIGN[7. Cosign Sign]
    SIGN --> INTERNAL[8. Push to Internal Registry]
```

**Key points:**
- All security scans run in parallel
- Image is built with Kaniko
- SBOM generated in SPDX format
- Image signed with Cosign (keyless)
- **Image stays in internal registry until e2e passes**

### Stage 5: E2E Testing

```mermaid
flowchart TB
    subgraph UPGRADE["Upgrade Test (gastown-test-upgrade)"]
        U1[helm upgrade --install]
        U2[Wait for ready]
        U3[Verify CRDs]
        U4[Test CR reconciliation]
    end

    subgraph FRESH["Fresh Install Test (gastown-test-fresh)"]
        F0[Delete namespace if exists]
        F1[helm install]
        F2[Wait for ready]
        F3[Verify CRDs]
        F4[Test CR reconciliation]
    end

    subgraph SMOKE["Smoke Tests"]
        S1[Operator running?]
        S2[CRDs installed?]
        S3[Rig reconciles?]
        S4[No errors in logs?]
    end

    TEKTON[Tekton Complete] --> UPGRADE
    TEKTON --> FRESH
    UPGRADE --> SMOKE
    FRESH --> SMOKE
    SMOKE -->|all pass| PUBLISH[Publish Stage]
    SMOKE -->|fail| STOP[Pipeline Failed<br/>No GitHub release]
```

**Two test scenarios:**
1. **Upgrade path:** Existing installation upgraded to new version
2. **Fresh install:** Clean namespace, new installation

### Stage 6: Publish to GitHub

```mermaid
flowchart LR
    subgraph PUBLISH["publish stage (sequential)"]
        GHCR[Push Image to GHCR]
        HELM[Push Helm to GHCR]
        SYNC[Sync Code to GitHub]
        RELEASE[Create GitHub Release]
    end

    E2E[E2E Tests Pass] --> GHCR
    GHCR --> HELM
    HELM --> SYNC
    SYNC --> RELEASE
```

**Critical:** This stage ONLY runs after e2e tests pass.
**Artifacts published:**
- `ghcr.io/boshu2/gastown-operator:X.Y.Z`
- `oci://ghcr.io/boshu2/charts/gastown-operator:X.Y.Z` (Helm)

## Complete Pipeline Timeline

```mermaid
gantt
    title CI/CD Pipeline Timeline
    dateFormat X
    axisFormat %s

    section Local
    pre-commit hooks    :a1, 0, 5s
    pre-push hooks      :a2, after a1, 30s

    section GitLab Validate
    lint:go (parallel)      :b1, after a2, 60s
    lint:yaml (parallel)    :b2, after a2, 30s
    validate:manifests      :b3, after a2, 45s
    validate:generate       :b4, after a2, 45s

    section Version
    semantic-release    :c1, after b1, 30s

    section Tekton
    Clone               :d1, after c1, 30s
    Security Scans (7x parallel) :d2, after d1, 180s
    Build Image         :d3, after d2, 300s
    Sign + SBOM         :d4, after d3, 60s
    Push Internal       :d5, after d4, 60s

    section E2E Tests
    Deploy Upgrade      :e1, after d5, 120s
    Deploy Fresh        :e2, after d5, 120s
    Smoke Tests         :e3, after e2, 60s

    section Publish
    Push GHCR           :f1, after e3, 60s
    Push Helm           :f2, after f1, 30s
    GitHub Release      :f3, after f2, 30s
```

## Environment Matrix

| Environment | Namespace | Registry | Trigger | Purpose |
|-------------|-----------|----------|---------|---------|
| **Test Upgrade** | `gastown-test-upgrade` | Internal (DPR) | Auto | Upgrade path validation |
| **Test Fresh** | `gastown-test-fresh` | Internal (DPR) | Auto | Fresh install validation |
| **Internal** | `gastown-system` | Internal (DPR) | Manual | Production-like internal |
| **Dev** | varies | GHCR | Manual | External/community |

## Required GitLab CI/CD Variables

| Variable | Type | Description |
|----------|------|-------------|
| `KUBE_CONFIG` | Variable | Base64-encoded kubeconfig |
| `REGISTRY_URL` | Variable | Internal registry URL |
| `GITHUB_TOKEN` | Variable | PAT with `repo` scope |
| `GITHUB_DEPLOY_KEY` | Variable | SSH key (base64) |
| `values-internal.yaml` | Secure File | Internal Helm values |

## Failure Scenarios

```mermaid
flowchart TB
    subgraph FAIL_POINTS["Pipeline Failure Points"]
        F1[Local hooks fail]
        F2[Validate stage fails]
        F3[Tekton security scan fails]
        F4[Tekton build fails]
        F5[E2E upgrade test fails]
        F6[E2E fresh install fails]
        F7[E2E smoke tests fail]
    end

    F1 -->|fix locally| RETRY1[Retry commit/push]
    F2 -->|fix & push| RETRY2[New pipeline]
    F3 -->|security issue| BLOCK[Release blocked]
    F4 -->|build error| FIX[Fix code]
    F5 -->|upgrade broken| ROLLBACK[Check backward compat]
    F6 -->|install broken| DEBUG[Debug helm chart]
    F7 -->|runtime error| LOGS[Check operator logs]

    BLOCK -.->|nothing published| SAFE[GitHub unchanged]
    F5 -.->|nothing published| SAFE
    F6 -.->|nothing published| SAFE
    F7 -.->|nothing published| SAFE
```

**Key safety:** If ANY test fails, nothing is published to GitHub/GHCR.

## Manual Operations

### Trigger Manual Deploy
```bash
# In GitLab UI: CI/CD → Pipelines → [pipeline] → deploy:internal → Play
```

### Emergency Rollback
```bash
# Rollback Helm release
helm rollback gastown-operator -n gastown-system

# Or deploy specific version
helm upgrade gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  -n gastown-system
```

### Skip Semantic Release
```bash
# Commit with [skip release] in message
git commit -m "chore: update docs [skip release]"
```

## Monitoring

### Check Pipeline Status
```bash
# GitLab
glab ci status

# Tekton
tkn pipelinerun list -n olympus-ci
tkn pipelinerun logs -f --last -n olympus-ci
```

### Check Deployments
```bash
# Test environments
kubectl get pods -n gastown-test-upgrade
kubectl get pods -n gastown-test-fresh

# Production
kubectl get pods -n gastown-system
```
