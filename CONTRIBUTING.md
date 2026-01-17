# Contributing to gastown-operator

We welcome contributions! This document provides guidelines.

## Getting Started

1. Fork the repository
2. Create a feature branch from `main`
3. Make your changes
4. Run tests: `make test`
5. Submit a pull request

## Development

### Prerequisites

- Go 1.24+
- Docker or Podman
- kubectl with cluster access
- make

### Build and Test

```bash
make build        # Build binary
make test         # Run tests
make lint         # Run linters
make manifests    # Generate CRDs
```

### Local Development

```bash
kind create cluster --name gastown-dev
make install      # Install CRDs
make run          # Run operator locally
```

## Pull Request Guidelines

- Keep PRs focused on a single change
- Add tests for new functionality
- Update documentation as needed
- Ensure CI passes

## Commit Messages

Use conventional commits:

```
feat(scope): add new feature
fix(scope): fix bug
docs: update documentation
```

## License

Contributions are licensed under Apache 2.0.
