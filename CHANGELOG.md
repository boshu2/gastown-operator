# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.2](https://github.com/boshu2/gastown-operator/compare/v0.3.1...v0.3.2) (2026-01-20) - First Stable Release

This is the first stable release of the Gas Town Kubernetes Operator.

### Highlights

- **Helm chart published to GHCR** with working defaults (no internal registry references)
- **Two editions**: Community (vanilla K8s) and Enterprise (OpenShift + FIPS)
- **Separated registries**: Container images at `ghcr.io/boshu2/gastown-operator`, Helm charts at `oci://ghcr.io/boshu2/charts/gastown-operator`
- **Helm-first documentation**: README prioritizes simple helm install

### Installation

```bash
helm install gastown-operator oci://ghcr.io/boshu2/charts/gastown-operator \
  --version 0.3.2 \
  --namespace gastown-system \
  --create-namespace
```

### Bug Fixes

* **helm:** use GHCR as default image registry (no internal registry leak)
* **helm:** separate chart and image registries to avoid tag collision
* **docs:** remove v prefix from image tags, update to 0.3.2
* **ci:** merge release workflows to fix GITHUB_TOKEN trigger limitation

## [0.3.1](https://github.com/boshu2/gastown-operator/compare/v0.3.0...v0.3.1) (2026-01-20)

### Bug Fixes

* **e2e:** correctly capture prescan exit code
* **prescan:** correctly extract file path from gocyclo output
* **prescan:** prevent early exit from arithmetic evaluation
* **prescan:** use here-string to avoid subshell variable scoping

## [0.3.0](https://github.com/boshu2/gastown-operator/compare/v0.2.0...v0.3.0) (2026-01-20)

### Features

* **ci:** add comprehensive E2E release validation script

## [0.2.0](https://github.com/boshu2/gastown-operator/compare/v0.1.2...v0.2.0) (2026-01-20)

### Features

* **ci:** add full automated CI/CD pipeline with Vibe + Athena

## [0.1.2](https://github.com/boshu2/gastown-operator/compare/v0.1.1...v0.1.2) (2026-01-19)

### Bug Fixes

* **helm:** correct NOTES.txt template syntax

## [0.1.1](https://github.com/boshu2/gastown-operator/compare/v0.1.0...v0.1.1) (2026-01-18)

### Bug Fixes

* **security:** SSH host key verification (MITM protection)
* **security:** command injection validation
* **helm:** GHCR OCI helm chart publishing
* **ci:** GitHub Actions CI with E2E tests

## [0.1.0](https://github.com/boshu2/gastown-operator/releases/tag/v0.1.0) (2026-01-16)

### Features

* Initial release
* Rig, Polecat, Convoy, Witness, Refinery, BeadStore CRDs
* Polecat controller creates pods for kubernetes execution mode
* Community and FIPS editions
