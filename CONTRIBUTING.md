# Contributing to gastown-operator

Thanks for your interest in contributing! This operator extends Gas Town to Kubernetes, and we welcome contributions.

## Getting Started

1. Fork the repository
2. Clone your fork
3. Install prerequisites:
   - Go 1.25+ (see go.mod for exact version)
   - kubectl
   - kind or minikube (for local testing)
4. Build and test: `make build && make test`

## Development Workflow

1. Create a feature branch from `main`
2. Make your changes
3. Ensure tests pass: `make test`
4. Run validation: `make lint`
5. Submit a pull request

## Code Style

- Follow standard Go conventions (`gofmt`, `go vet`)
- Run `make lint` before committing
- Add tests for new controllers
- Keep functions focused and small

## What to Contribute

Good first contributions:
- Bug fixes with clear reproduction steps
- Documentation improvements
- Test coverage for untested code paths
- New CRD fields with tests

For larger changes (new CRDs, architectural changes), please open an issue first.

## Commit Messages

Format: `type(scope): description`

Types: `feat`, `fix`, `docs`, `test`, `chore`, `refactor`

Examples:
- `feat(polecat): add resource limits configuration`
- `fix(convoy): correct status aggregation`
- `docs: update quickstart guide`

## Testing

```bash
# Unit tests
make test

# E2E tests (requires kind cluster)
make test-e2e

# Validation (lint + vet)
make lint
```

## Building

```bash
# Community edition
make docker-build

# Enterprise/FIPS edition
make docker-build-fips
```

## Questions?

Open an issue for questions about contributing.
