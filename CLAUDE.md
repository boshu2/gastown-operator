# Gastown Operator - Agent Instructions

> **Recovery**: Run `gt prime` after compaction, clear, or new session

---

## Tech Stack

| Component | Version | Purpose |
|-----------|---------|---------|
| Go | 1.25 | Primary language |
| controller-runtime | 0.23.x | Kubernetes operator framework |
| Kubernetes | 1.35.x | Target platform |
| Ginkgo/Gomega | 2.27.x | Testing framework |
| golangci-lint | 2.5.0 | Linter |
| Kind | latest | Local K8s for E2E |
| Helm | 3.x | Chart packaging |
| Docker/Buildx | latest | Multi-arch builds |

### Key Dependencies

```
k8s.io/api, k8s.io/apimachinery, k8s.io/client-go  # Kubernetes APIs
sigs.k8s.io/controller-runtime                      # Operator framework
github.com/prometheus/client_golang                 # Metrics
github.com/spf13/cobra                              # CLI framework
golang.org/x/time                                   # Rate limiting
```

---

## Build Commands

```bash
# Binaries
make build           # Build operator binary → bin/manager
make kubectl-gt      # Build kubectl plugin → bin/kubectl-gt
make build-gt        # Build gt CLI (clones gastown repo)

# Container Images
make docker-build    # Build image (single arch)

# Manifests
make manifests       # Regenerate CRDs from Go types
make generate        # Regenerate DeepCopy methods
make sync-helm       # Sync CRDs to Helm chart

# Install manifests
make build-installer # Generate dist/install.yaml
```

---

## Testing

### Unit Tests (Fast - Use These)

```bash
make test            # Run unit tests with envtest (~12s)
```

- Uses fake client for controller tests (0.6s for 37 tests)
- Uses envtest for integration tests
- Coverage output: `cover.out`

### E2E Tests (Slow - Kind Cluster)

```bash
make test-e2e        # Full E2E in Kind (~5 min)
make demo            # Deploy to Kind for manual testing
```

### Test Strategy

| Test Type | When to Use | Speed |
|-----------|-------------|-------|
| Unit (fake client) | Controller logic, reconcile flow | Fast (0.6s) |
| Unit (envtest) | CRD validation, API behavior | Medium (12s) |
| E2E (Kind) | Full operator lifecycle, Helm install | Slow (5 min) |

---

## Validation

### Local Validation (Before Push)

```bash
make validate        # go vet + golangci-lint
make validate-helm   # Check Helm chart in sync
make validate-all    # Both of the above
```

### CI Pipeline Jobs

| Job | What It Checks |
|-----|----------------|
| validate | go.mod tidy, go vet, generated code, manifests |
| lint | golangci-lint |
| test | Unit tests with envtest |
| prescan | Complexity, code quality (vibe) |
| security | govulncheck, gosec, gitleaks |
| build | Binaries + container image |
| e2e | Kind cluster tests (main branch only) |

### Pre-Commit Hook

```bash
make setup-hooks     # Install git hooks
```

Runs on commit: gofmt, goimports, go vet, go mod tidy

---

## RPI Workflow

**Research → Plan → Implement → Validate**

### Research Phase

```bash
/research <topic>    # Deep codebase exploration
ao inject            # Load prior knowledge
```

### Plan Phase

```bash
/plan <goal>         # Decompose into beads issues
/pre-mortem <spec>   # Simulate failures before implementing
bd create "title"    # Create issue manually
```

### Implement Phase

```bash
/implement <issue>   # Execute single issue
/crank <epic>        # Autonomous epic execution
/swarm               # Parallel agent execution
```

### Validate Phase

```bash
/vibe [target]       # Code validation
/post-mortem         # Full validation + learning extraction
/retro               # Quick retrospective
```

---

## Release Workflow

**MANDATORY: Follow this checklist. Skipping steps causes CI failures.**

### Pipeline Architecture

```
Tag push (vX.Y.Z) triggers release.yml:
  Pre-flight → Validation → [GoReleaser binaries | Container images] → Helm → SBOM → GitHub Release → Verify
```

| Component | Owner | What It Does |
|-----------|-------|--------------|
| kubectl-gt binaries | GoReleaser (`.goreleaser.yaml`) | Cross-platform builds, checksums, binary SBOM |
| Container images | release.yml + buildx | Multi-arch (amd64/arm64), Cosign signing |
| Helm chart | release.yml | Package + push to GHCR OCI |
| Image SBOM | release.yml + Syft | SPDX + CycloneDX from image digest |
| Dependencies | Renovate (GitHub App) | Auto-PRs for Go, Actions, Dockerfiles, Makefile tools |

### 1. Pre-Release Validation

```bash
# Ensure go.mod is tidy (CRITICAL - v0.4.3 lesson)
go mod tidy
git diff --exit-code go.mod go.sum || {
  git add go.mod go.sum
  git commit -m "chore: go mod tidy"
}

# Run full validation
make validate-all
make test

# Push and wait for CI
git push
gh run watch
```

### 2. Update Version

```bash
# Update VERSION file
echo "X.Y.Z" > VERSION

# Update CHANGELOG.md with release notes
# Follow Keep a Changelog format

git add VERSION CHANGELOG.md
git commit -m "docs: add X.Y.Z changelog entry"
git push

# Wait for CI again
gh run watch
```

### 3. Tag and Release

```bash
# Only after CI passes
git tag vX.Y.Z
git push --tags

# Monitor release workflow
gh run watch
```

### 4. Post-Release Verification

```bash
# Verify artifacts
gh release view vX.Y.Z
helm show chart oci://ghcr.io/boshu2/charts/gastown-operator --version X.Y.Z
docker pull ghcr.io/boshu2/gastown-operator:X.Y.Z
```

### Release Artifacts

| Artifact | Location |
|----------|----------|
| Container image | `ghcr.io/boshu2/gastown-operator:X.Y.Z` |
| Helm chart | `oci://ghcr.io/boshu2/charts/gastown-operator:X.Y.Z` |
| Install manifest | GitHub Release `install.yaml` |
| kubectl-gt binaries | GitHub Release (darwin/linux/windows, amd64/arm64) |
| SBOM (container) | GitHub Release (SPDX + CycloneDX) |
| SBOM (binaries) | GitHub Release (SPDX per binary) |
| Checksums | GitHub Release `checksums.txt` |

### Dependency Updates (Renovate)

Renovate GitHub App creates PRs automatically for:

| Category | Source | Automerge |
|----------|--------|-----------|
| Go modules | `go.mod` | Minor/patch yes, K8s ecosystem no |
| GitHub Actions | `.github/workflows/` | Yes |
| Dockerfiles | `Dockerfile*` | No |
| Makefile tools | `GOLANGCI_LINT_VERSION`, etc. | No |

### Why go mod tidy Matters

**v0.4.3 Lesson:** Tagged before running `go mod tidy`. New imports can promote dependencies from indirect to direct. CI failed on "go.mod not tidy" check.

**Rule:** Always run `go mod tidy` after adding new imports.

---

## Architecture

### Custom Resources

| CRD | Purpose |
|-----|---------|
| **Rig** | Workspace definition (git repo + settings) |
| **Polecat** | Worker pod running Claude Code |
| **Witness** | Monitors Polecat activity, escalates issues |
| **Refinery** | Merges Polecat branches, resolves conflicts |
| **BeadStore** | Issue tracking backend (git-native) |
| **Convoy** | Batch operations across multiple Polecats |

### Controller Patterns (Elite Operator)

- **Watch predicates**: GenerationChangedPredicate on all 6 controllers
- **Rate limiting**: Exponential backoff (5ms-5min) + bucket (10/s, burst 100)
- **Metrics**: Per-operation histograms (child creation, pod ops, git ops)
- **CEL validation**: CRD rules without webhooks

---

## Common Issues

### go.mod not tidy
```bash
go mod tidy && git add go.mod go.sum && git commit --amend --no-edit
```

### Manifests out of date
```bash
make manifests generate && git add config/ api/ && git commit -m "chore: regenerate manifests"
```

### Helm chart out of sync
```bash
make sync-helm && git add helm/ && git commit -m "chore: sync helm chart"
```

### CEL validation errors
Guard optional fields: `!has(self.field) || self.field.value <= 50`
