# Agent Instructions

See **[AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md)** for complete agent context.

This file exists for compatibility with tools that look for AGENTS.md.

## Quick Reference (CLI-First)

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

# Create rig and dispatch work
kubectl gt rig create my-project --git-url git@github.com:org/repo.git --prefix mp
kubectl gt sling issue-123 my-project --name furiosa
kubectl gt polecat logs my-project/furiosa -f
```

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

## Full Documentation

- [README.md](README.md) - Main docs with CLI reference
- [AGENT_INSTRUCTIONS.md](AGENT_INSTRUCTIONS.md) - Agent context
- [docs/USER_GUIDE.md](docs/USER_GUIDE.md) - Complete walkthrough

## Key Point

**Use the kubectl-gt CLI** for normal workflows. YAML templates are available in [templates/](templates/) for advanced use cases or GitOps.

---

## Release Workflow

**MANDATORY: Follow this checklist for every release. Skipping steps causes CI failures.**

### Pre-Release Checklist

Run these commands BEFORE tagging:

```bash
# 1. Ensure go.mod is tidy (CRITICAL - v0.4.3 lesson)
go mod tidy
git diff --exit-code go.mod go.sum || {
  echo "go.mod changed - commit it first"
  git add go.mod go.sum
  git commit -m "chore: go mod tidy"
}

# 2. Run full validation
make lint
make test

# 3. Push and wait for CI
git push
gh run watch  # Wait for CI to pass

# 4. Verify CI passed
gh run list --limit 1  # Should show "success"
```

### Create Release

Only after CI passes:

```bash
# 1. Update CHANGELOG.md with release notes
# 2. Commit changelog
git add CHANGELOG.md
git commit -m "docs: add X.Y.Z changelog entry"
git push

# 3. Tag and push (triggers release workflow)
git tag vX.Y.Z
git push --tags

# 4. Monitor release
gh run watch  # Watch the Release workflow
```

### Post-Release Verification

```bash
# Verify artifacts published
gh release view vX.Y.Z

# Verify helm chart
helm show chart oci://ghcr.io/boshu2/charts/gastown-operator --version X.Y.Z

# Verify container image
docker pull ghcr.io/boshu2/gastown-operator:X.Y.Z
```

### Why This Matters

**v0.4.3 Lesson:** Tagged before running `go mod tidy`. The `pkg/workqueue` rate limiter imports `golang.org/x/time/rate`, which promoted the dependency from indirect to direct. CI failed on "go.mod not tidy" check. Release succeeded (separate workflow) but CI showed failure.

**Root cause:** New imports can change dependency classification. Always run `go mod tidy` before committing Go changes.

---

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
