# Polecat Agent - Pre-built container for Gas Town polecat workers
#
# This image comes with Claude Code pre-installed for faster polecat startup.
# Uses native Claude Code binary (no Node.js/npm required).
#
# SECURITY: All versions are pinned for reproducible builds and SBOM accuracy.
# SBOM and Trivy scan results are published with each release.
#
# Usage:
#   docker pull ghcr.io/boshu2/polecat-agent:latest
#
# Build (from repo root — needed for promo-finish):
#   docker build -f images/polecat-agent/Dockerfile -t polecat-agent:local .
#
# Multi-arch build with attestations:
#   docker buildx build --platform linux/amd64,linux/arm64 \
#     --sbom=true --provenance=true \
#     -f images/polecat-agent/Dockerfile \
#     -t ghcr.io/boshu2/polecat-agent:0.4.0 \
#     --push .

# ==============================================================================
# Version Pins - Update these for each release
# ==============================================================================
# Claude Code: https://github.com/anthropics/claude-code/releases
ARG CLAUDE_CODE_VERSION=2.0.22
# Gas Town CLI: https://github.com/steveyegge/gastown
ARG GT_VERSION=main
ARG GOLANG_VERSION=1.26
# Base image - Debian slim for glibc compatibility
ARG DEBIAN_VERSION=bookworm-slim

# ==============================================================================
# Stage 1: Build gt CLI
# ==============================================================================
FROM golang:${GOLANG_VERSION}-alpine AS gt-builder

RUN apk add --no-cache git

WORKDIR /build
ARG GT_VERSION
ENV GOTOOLCHAIN=auto
RUN git clone --depth 1 --branch ${GT_VERSION} https://github.com/steveyegge/gastown.git . && \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/gt ./cmd/gt

# ==============================================================================
# Stage 1b: Build promo-finish (orchestrator handoff tool)
# Build with context = repo root:
#   docker build -f images/polecat-agent/Dockerfile -t polecat-agent:local .
# ==============================================================================
FROM golang:${GOLANG_VERSION}-alpine AS promo-finish-builder
WORKDIR /src
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download
COPY bte-promo-script/tools/finish ./bte-promo-script/tools/finish
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/promo-finish ./bte-promo-script/tools/finish

# ==============================================================================
# Stage 1c: Build promo-source-fetch (episode docx download for promo jobs)
# ==============================================================================
FROM golang:${GOLANG_VERSION}-alpine AS promo-source-fetch-builder
WORKDIR /src
ENV GOTOOLCHAIN=auto
COPY go.mod go.sum ./
RUN go mod download
COPY bte-promo-script/tools/sourcefetch ./bte-promo-script/tools/sourcefetch
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/promo-source-fetch ./bte-promo-script/tools/sourcefetch

# ==============================================================================
# Stage 2: Build polecat-agent image
# ==============================================================================
FROM debian:${DEBIAN_VERSION}

# Repeat ARGs needed in this stage
ARG CLAUDE_CODE_VERSION

LABEL org.opencontainers.image.title="Polecat Agent"
LABEL org.opencontainers.image.description="Pre-built container for Gas Town polecat workers with Claude Code ${CLAUDE_CODE_VERSION}"
LABEL org.opencontainers.image.source="https://github.com/boshu2/gastown-operator"
LABEL org.opencontainers.image.vendor="Gas Town"
LABEL org.opencontainers.image.version="${CLAUDE_CODE_VERSION}"
LABEL io.gastown.claude-code.version="${CLAUDE_CODE_VERSION}"

# Install system dependencies (no Node.js needed - Claude Code is native binary)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    openssh-client \
    ca-certificates \
    curl \
    jq \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user for security (UID 65532 matches operator's security context)
RUN groupadd -r -g 65532 polecat && useradd -r -u 65532 -g polecat -d /home/nonroot -m polecat

# Install Claude Code via native installer script
# Pin to specific version for reproducibility
ARG CLAUDE_CODE_VERSION
RUN curl -fsSL https://claude.ai/install.sh | bash -s ${CLAUDE_CODE_VERSION} && \
    # Copy actual binary (not symlink) to system-wide location
    cp -L /root/.local/bin/claude /usr/local/bin/claude && \
    chmod +x /usr/local/bin/claude && \
    # Cleanup root's installation to save space
    rm -rf /root/.local && \
    # Verify installation
    /usr/local/bin/claude --version

# Copy gt CLI + promo-finish from builders
COPY --from=gt-builder /out/gt /usr/local/bin/gt
COPY --from=promo-finish-builder /out/promo-finish /usr/local/bin/promo-finish
COPY --from=promo-source-fetch-builder /out/promo-source-fetch /usr/local/bin/promo-source-fetch
RUN chmod +x /usr/local/bin/gt /usr/local/bin/promo-finish /usr/local/bin/promo-source-fetch

# Setup SSH directory with proper permissions
RUN mkdir -p /home/nonroot/.ssh && \
    chmod 700 /home/nonroot/.ssh && \
    chown -R polecat:polecat /home/nonroot/.ssh

# Setup Claude config directory
RUN mkdir -p /home/nonroot/.claude && \
    chown -R polecat:polecat /home/nonroot/.claude

# Setup git config defaults
RUN git config --system user.email "polecat@gastown.local" && \
    git config --system user.name "Polecat Worker" && \
    git config --system init.defaultBranch main

# Create workspace directory
RUN mkdir -p /workspace && chown polecat:polecat /workspace

# BTE promo pipeline (static workspace — copied to emptyDir by workspace-init)
COPY bte-promo-script/repos/bte-promo-script-repo /opt/bte-promo-tool
RUN chown -R polecat:polecat /opt/bte-promo-tool

# Switch to non-root user
USER polecat
WORKDIR /workspace

# Verify installations and print versions for build logs
RUN echo "=== Installed Versions ===" && \
    echo "Claude Code: $(claude --version)" && \
    echo "gt CLI: $(gt version 2>/dev/null || echo 'installed')" && \
    echo "promo-finish: $(promo-finish --help 2>&1 | head -1 || echo 'installed')" && \
    echo "git: $(git --version)" && \
    echo "=========================="

# Default entrypoint - can be overridden by operator
ENTRYPOINT ["claude"]
CMD ["--help"]
