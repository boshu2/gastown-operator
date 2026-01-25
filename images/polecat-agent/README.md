# Polecat Agent Container Image

Pre-built, security-scanned container image for Gas Town polecat workers with Claude Code pre-installed.

## Why Pre-built?

| Without Pre-built | With Pre-built |
|-------------------|----------------|
| Install Claude Code at runtime | Already installed |
| ~30-60s startup delay | Instant startup |
| Requires network access | No external deps at runtime |
| Unknown component versions | Pinned, documented versions |
| No security scan | Trivy-scanned, SBOM included |

## Quick Start

```bash
docker pull ghcr.io/boshu2/polecat-agent:0.4.0
```

## Security

Every release includes:

| Artifact | Description |
|----------|-------------|
| **SBOM** | Software Bill of Materials attached to image |
| **Trivy Scan** | Vulnerability scan for HIGH/CRITICAL CVEs |
| **Version Manifest** | JSON with all component versions |
| **Provenance** | Build attestation (who built, when, how) |

### Pinned Versions (0.4.0)

| Component | Version | Source |
|-----------|---------|--------|
| Claude Code | 2.0.22 | [anthropics/claude-code](https://github.com/anthropics/claude-code) |
| gt CLI | main | [steveyegge/gastown](https://github.com/steveyegge/gastown) |
| Base Image | debian:bookworm-slim | Official Debian |
| git | system | Debian package |

### Viewing Security Artifacts

```bash
# View SBOM (attached to image)
docker buildx imagetools inspect ghcr.io/boshu2/polecat-agent:0.4.0 --format '{{json .SBOM}}'

# Run Trivy scan
trivy image ghcr.io/boshu2/polecat-agent:0.4.0

# View provenance
docker buildx imagetools inspect ghcr.io/boshu2/polecat-agent:0.4.0 --format '{{json .Provenance}}'
```

## What's Included

| Component | Purpose |
|-----------|---------|
| **Claude Code** | AI coding assistant (native binary) |
| **gt CLI** | Gas Town integration |
| **git** | Version control |
| **openssh-client** | SSH for git operations |
| **jq** | JSON processing |
| **curl** | HTTP client |

**Not included** (minimal base - extend as needed):
- Python, Go, Node.js, Rust
- Build tools (make, gcc)
- Language package managers

Need more tools? See [CUSTOMIZING.md](CUSTOMIZING.md) for how to extend the image.

## Usage with Operator

The operator uses this image by default. To override globally:

```yaml
# In operator deployment (values.yaml)
env:
  GASTOWN_CLAUDE_IMAGE: "ghcr.io/boshu2/polecat-agent:0.4.0"
```

Or per-Polecat:

```yaml
apiVersion: gastown.gastown.io/v1alpha1
kind: Polecat
spec:
  execution:
    kubernetes:
      image: "ghcr.io/boshu2/polecat-agent:0.4.0"
```

## Building Locally

```bash
# Single arch (for testing)
docker build -t polecat-agent:local images/polecat-agent/

# Multi-arch with SBOM (for release)
docker buildx build --platform linux/amd64,linux/arm64 \
  --sbom=true --provenance=true \
  -t ghcr.io/boshu2/polecat-agent:0.4.0 \
  --push images/polecat-agent/

# Full release with Trivy scan
./scripts/release-polecat-agent.sh 0.4.0
```

## Release Process

```bash
# 1. Update version pins in Dockerfile
vim images/polecat-agent/Dockerfile

# 2. Build, scan, and push
./scripts/release-polecat-agent.sh 0.4.0

# 3. Artifacts created in dist/
ls dist/
# trivy-polecat-agent-0.4.0.json   - Vulnerability scan
# sbom-polecat-agent-0.4.0.json    - Software bill of materials
# manifest-polecat-agent-0.4.0.json - Version manifest
```

## Enterprise/FIPS

For FIPS-compliant environments, we provide a UBI-based variant:

```bash
docker pull ghcr.io/boshu2/polecat-agent:0.4.0-fips
```

Build your own:
```bash
docker build \
  --build-arg DEBIAN_VERSION=ubi9-minimal \
  -t polecat-agent:fips \
  images/polecat-agent/
```

## Container Security

- Runs as non-root user (`polecat`, UID 999)
- Minimal attack surface (slim base image, only required packages)
- No secrets baked in (provided at runtime via K8s secrets)
- SSH keys mounted at runtime, not built into image
- Read-only root filesystem compatible
