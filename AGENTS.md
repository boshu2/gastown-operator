# Agent Instructions

See **[CLAUDE.md](CLAUDE.md)** for full technical documentation.

---

## Quick Start

```bash
# Install operator
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.4.3 \
  --namespace gastown-system \
  --create-namespace

# Install kubectl-gt plugin
curl -LO https://github.com/boshu2/gastown-operator/releases/download/v0.4.3/kubectl-gt-darwin-arm64
chmod +x kubectl-gt-darwin-arm64 && sudo mv kubectl-gt-darwin-arm64 /usr/local/bin/kubectl-gt

# Set up credentials
kubectl create secret generic git-creds -n gastown-system \
  --from-file=ssh-privatekey=$HOME/.ssh/id_ed25519
kubectl gt auth sync -n gastown-system
```

---

## Tech Stack

| Component | Version |
|-----------|---------|
| Go | 1.25 |
| controller-runtime | 0.23.x |
| Kubernetes | 1.35.x |
| golangci-lint | 2.5.0 |

---

## Build

```bash
make build           # Operator binary
make kubectl-gt      # kubectl plugin
make docker-build    # Container image
make manifests       # Regenerate CRDs
```

---

## Test

```bash
make test            # Unit tests (fast, ~12s)
make test-e2e        # E2E in Kind (slow, ~5min)
```

---

## Validate

```bash
make validate        # go vet + lint
make validate-all    # + Helm sync check
```

### CI Jobs

validate → lint → test → prescan → security → build → e2e (main only)

---

## RPI Workflow

```
Research → Plan → Implement → Validate
```

| Phase | Commands |
|-------|----------|
| Research | `/research`, `ao inject` |
| Plan | `/plan`, `/pre-mortem`, `bd create` |
| Implement | `/implement`, `/crank`, `/swarm` |
| Validate | `/vibe`, `/post-mortem`, `/retro` |

---

## Release Workflow

**MANDATORY CHECKLIST - Follow every step.**

### 1. Pre-Release

```bash
go mod tidy
git diff --exit-code go.mod go.sum || git commit -am "chore: go mod tidy"
make validate-all
make test
git push && gh run watch  # Wait for CI
```

### 2. Version & Changelog

```bash
echo "X.Y.Z" > VERSION
# Edit CHANGELOG.md
git add VERSION CHANGELOG.md
git commit -m "docs: add X.Y.Z changelog entry"
git push && gh run watch  # Wait for CI
```

### 3. Tag

```bash
git tag vX.Y.Z
git push --tags
gh run watch  # Watch Release workflow
```

### 4. Verify

```bash
gh release view vX.Y.Z
helm show chart oci://ghcr.io/boshu2/charts/gastown-operator --version X.Y.Z
```

### v0.4.3 Lesson

**Always `go mod tidy` before tagging.** New imports can promote dependencies from indirect to direct, causing CI to fail on "go.mod not tidy" check.

---

## CLI Commands

| Command | Description |
|---------|-------------|
| `kubectl gt rig list` | List rigs |
| `kubectl gt rig create <name>` | Create rig |
| `kubectl gt polecat list` | List polecats |
| `kubectl gt polecat logs <rig>/<name>` | Stream logs |
| `kubectl gt sling <bead> <rig>` | Dispatch work |
| `kubectl gt convoy list` | List convoys |
| `kubectl gt auth sync` | Sync Claude creds |

---

## Landing the Plane

**When ending a session**, complete ALL steps:

1. **File issues** for remaining work
2. **Run quality gates** - `make validate-all && make test`
3. **Update issue status** - Close finished, update in-progress
4. **PUSH TO REMOTE** (MANDATORY):
   ```bash
   git pull --rebase && bd sync && git push
   git status  # MUST show "up to date"
   ```
5. **Verify** - All changes pushed
6. **Hand off** - Context for next session

**RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing
- If push fails, resolve and retry
