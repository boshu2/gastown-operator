# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).


## [0.3.2](https://github.com/boshu2/gastown-operator/compare/v0.3.1...v0.3.2) (2026-01-20)

### Bug Fixes

* **ci:** merge release workflows to fix GITHUB_TOKEN trigger limitation ([9bc65fc](https://github.com/boshu2/gastown-operator/commit/9bc65fcfc4aa044dd8b4f9a732077f4fca4865ad))

## [0.3.2](https://github.com/boshu2/gastown-operator/compare/v0.3.1...v0.3.2) (2026-01-20)

### Bug Fixes

* **ci:** merge release workflows to fix GITHUB_TOKEN trigger limitation ([9bc65fc](https://github.com/boshu2/gastown-operator/commit/9bc65fcfc4aa044dd8b4f9a732077f4fca4865ad))

## [0.3.2](https://github.com/boshu2/gastown-operator/compare/v0.3.1...v0.3.2) (2026-01-20)

### Bug Fixes

* **ci:** merge release workflows to fix GITHUB_TOKEN trigger limitation ([9bc65fc](https://github.com/boshu2/gastown-operator/commit/9bc65fcfc4aa044dd8b4f9a732077f4fca4865ad))

## [0.3.1](https://github.com/boshu2/gastown-operator/compare/v0.3.0...v0.3.1) (2026-01-20)

### Bug Fixes

* **e2e:** correctly capture prescan exit code ([6768217](https://github.com/boshu2/gastown-operator/commit/6768217e95d52718e118d742e49c3ca1e91644a2))
* **prescan:** correctly extract file path from gocyclo output ([ba73bf1](https://github.com/boshu2/gastown-operator/commit/ba73bf1d6577f08ce49b31498937f1cbe495242c))
* **prescan:** prevent early exit from arithmetic evaluation ([e2abe2f](https://github.com/boshu2/gastown-operator/commit/e2abe2f43e5e096d8452734a6e6c39085d4349b4))
* **prescan:** use here-string to avoid subshell variable scoping ([aed81a4](https://github.com/boshu2/gastown-operator/commit/aed81a4883bdc542d4a52a8f68c0e95e70562e1d))

## [0.3.0](https://github.com/boshu2/gastown-operator/compare/v0.2.0...v0.3.0) (2026-01-20)

### Features

* **ci:** add comprehensive E2E release validation script ([45b82a5](https://github.com/boshu2/gastown-operator/commit/45b82a5e1cecbd455dbdbea7ec5a31b86121341d))

## [0.2.0](https://github.com/boshu2/gastown-operator/compare/v0.1.2...v0.2.0) (2026-01-20)

### Features

* **ci:** add comprehensive GitHub Actions E2E testing ([2f38427](https://github.com/boshu2/gastown-operator/commit/2f3842756878f0bad1ece9d5ee56b69d3e973829))
* **ci:** add full automated CI/CD pipeline with Vibe + Athena ([5dda90d](https://github.com/boshu2/gastown-operator/commit/5dda90d19344bfd915e85ddba5ef7fed1f8165af))
* **controllers:** remediate vibe assessment findings (P1 & P2) ([1a16a23](https://github.com/boshu2/gastown-operator/commit/1a16a231ec29c3b96975da836cc7f84165673964))
* **docs:** remediate P3 vibe assessment findings ([0cf8939](https://github.com/boshu2/gastown-operator/commit/0cf8939b99df5abf9ee5f4f34dc03bc255bad571))
* **go-cwl:** add /version endpoint to webhook server ([f492b49](https://github.com/boshu2/gastown-operator/commit/f492b491cc2837fc860ff386ff2ebe8954205b1d))
* **helm:** add GHCR OCI publishing for helm chart ([753a81a](https://github.com/boshu2/gastown-operator/commit/753a81af803aeec89489aa84a7eff1a5cc723380))
* **pod:** add API key auth as alternative to OAuth ([e44e02d](https://github.com/boshu2/gastown-operator/commit/e44e02d96f3635c96942ef22bfddc8512d38256e))
* **release:** add comprehensive E2E validation gate for releases ([2e094fb](https://github.com/boshu2/gastown-operator/commit/2e094fbeb8b03a804014884d10c47a1f2fe9801a))
* **tekton:** add comprehensive security scanning and supply chain tasks ([8700670](https://github.com/boshu2/gastown-operator/commit/87006709b8101756d20172f1778608fd0c10164e))
* **tekton:** add dockerfile parameter for FIPS builds ([9d5a376](https://github.com/boshu2/gastown-operator/commit/9d5a3767f0d7a4afa6a703921eff02cf88086587))

### Bug Fixes

* **api:** use error wrapping in rig_webhook.go ([bf66fde](https://github.com/boshu2/gastown-operator/commit/bf66fdeea51e2b7f5988693f21d0821f5a0e362f))
* change default agent from opencode to claude-code ([4727f8e](https://github.com/boshu2/gastown-operator/commit/4727f8e3ae3092379d106d8a302401fa0f986898))
* **devcontainer:** quote command substitutions in post-install.sh ([a027dd5](https://github.com/boshu2/gastown-operator/commit/a027dd5c52472ac81cc43b3df34c7a2c67d1b034))
* **git:** use pre-verified SSH known_hosts to prevent MITM attacks ([21d69d8](https://github.com/boshu2/gastown-operator/commit/21d69d805908851c4aab1c75ddfb46f7eae8a396))
* **pod:** enable git push and improve prompt handling ([b5d7a65](https://github.com/boshu2/gastown-operator/commit/b5d7a658f128ae6e6edcc0ddf35be5383004257f))
* **security:** add TestCommand validation to prevent command injection ([20a56f1](https://github.com/boshu2/gastown-operator/commit/20a56f197213c323364c333e3796cd75f0963241))

### Refactoring

* **release:** simplify GitHub Actions, keep real E2E in Tekton ([46cf6f9](https://github.com/boshu2/gastown-operator/commit/46cf6f9069d5fb4db890fd001a66411457603256))

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
