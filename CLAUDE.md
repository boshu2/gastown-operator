# Mayor Context (gastown_operator)

> **Recovery**: Run `gt prime` after compaction, clear, or new session

Full context is injected by `gt prime` at session start.

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

## Development Commands

```bash
make lint          # Run golangci-lint
make test          # Run unit tests (fast, uses fake client)
make test-e2e      # Run e2e tests (slow, needs cluster)
make build         # Build operator binary
make docker-build  # Build container image
make manifests     # Regenerate CRDs
```

---

## Architecture Quick Reference

- **Rig**: Workspace definition (git repo + settings)
- **Polecat**: Worker pod running Claude Code
- **Witness**: Monitors Polecat activity, escalates issues
- **Refinery**: Merges Polecat branches, resolves conflicts
- **BeadStore**: Issue tracking backend (git-native)
- **Convoy**: Batch operations across multiple Polecats
