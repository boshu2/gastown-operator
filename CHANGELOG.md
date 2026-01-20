# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.1.1] - 2026-01-20

### Fixed

- **Security**: Fixed SSH StrictHostKeyChecking disabled in internal/git/git.go (MITM vulnerability)
- **Security**: Added TestCommand validation in merge.go to prevent command injection
- **Security**: Consolidated SSH security approach between git.go and pod/builder.go
- **Quality**: Fixed error wrapping using %w instead of %v in rig_webhook.go
- **Quality**: Fixed shell script quoting in devcontainer post-install.sh
- **Quality**: Added nolint:errcheck comments to intentional error ignores
- **Quality**: Documented acceptable cyclomatic complexity with nolint:gocyclo

### Added

- GHCR OCI publishing for Helm chart (oci://ghcr.io/boshu2/gastown-operator)
- GitHub Actions workflow for automated Helm chart releases

## [0.1.0] - 2026-01-17

### Added

- Initial release of Kubernetes Operator for Gas Town
- **Custom Resource Definitions**:
  - `Rig` - Project workspace (cluster-scoped)
  - `Polecat` - Autonomous worker agent pod
  - `Convoy` - Batch tracking for parallel execution
  - `Refinery` - Merge queue processor
  - `Witness` - Worker lifecycle monitor
  - `BeadStore` - Issue tracking backend
- **Two Build Editions**:
  - Community: Vanilla Kubernetes with distroless images
  - Enterprise: OpenShift + FIPS with UBI9 images
- **Security**:
  - Restricted PSS/SCC compliant pod security contexts
  - FIPS-validated crypto (Enterprise edition)
  - Non-root containers with read-only filesystems
- **Deployment**:
  - Helm chart
  - Kustomize overlays for community and FIPS
  - Install manifests for both editions
- **Documentation**:
  - Quick start guide
  - CRD reference
  - Architecture overview
  - Development guide

### Security

- All pods run as non-root (UID 65532)
- Read-only root filesystems with emptyDir for writable paths
- All capabilities dropped
- Seccomp profile enforced

[0.1.1]: https://github.com/boshu2/gastown-operator/releases/tag/v0.1.1
[0.1.0]: https://github.com/boshu2/gastown-operator/releases/tag/v0.1.0
