# gastown-operator Dockerfile (Community Edition)
# Multi-stage build for vanilla Kubernetes environments
#
# Supports multi-arch builds (amd64, arm64) via docker buildx.
# The gt CLI is built from source for each target platform.
#
# For Tekton CI: Uses pre-built gt binary from context if available.
# For local builds: Run `make build-gt` first OR let the build stage build it.
#
# Build args allow image override in CI:
#   --build-arg GO_IMAGE=${REGISTRY}/golang:1.25-alpine

ARG GO_IMAGE=golang:1.25-alpine
ARG RUNTIME_IMAGE=gcr.io/distroless/static:nonroot

# ------------------------------------------------------------------------------
# Stage 1: Build gt CLI from gastown (for multi-arch support)
# If a pre-built gt binary exists in context, this stage is still run but
# the binary from context takes precedence in the final stage.
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS gt-builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install git and ca-certificates
RUN if command -v apk > /dev/null; then \
        apk add --no-cache git ca-certificates; \
    elif command -v apt-get > /dev/null; then \
        apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*; \
    elif command -v dnf > /dev/null; then \
        dnf install -y git ca-certificates && dnf clean all; \
    elif command -v yum > /dev/null; then \
        yum install -y git ca-certificates && yum clean all; \
    else \
        echo "No package manager found" && exit 1; \
    fi

WORKDIR /build

# Clone and build gt for target platform
RUN git clone --depth 1 https://github.com/steveyegge/gastown.git . && \
    CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/gt ./cmd/gt

# ------------------------------------------------------------------------------
# Stage 2: Build operator manager
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install git and ca-certificates (Alpine, Debian, or RHEL)
RUN if command -v apk > /dev/null; then \
        apk add --no-cache git ca-certificates; \
    elif command -v apt-get > /dev/null; then \
        apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*; \
    elif command -v dnf > /dev/null; then \
        dnf install -y git ca-certificates && dnf clean all; \
    elif command -v yum > /dev/null; then \
        yum install -y git ca-certificates && yum clean all; \
    else \
        echo "No package manager found" && exit 1; \
    fi
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build with standard Go crypto
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/manager ./cmd/main.go

# ------------------------------------------------------------------------------
# Stage 3: Minimal distroless runtime image
# ------------------------------------------------------------------------------
FROM ${RUNTIME_IMAGE}

WORKDIR /

# Copy gt CLI built for target platform
COPY --from=gt-builder /out/gt /usr/local/bin/gt

# Copy operator manager
COPY --from=builder /out/manager .

USER 65532:65532
ENTRYPOINT ["/manager"]
