# gastown-operator Dockerfile (Community Edition)
# Multi-stage build for vanilla Kubernetes environments
#
# Lightweight build using Alpine and distroless images.
# For FIPS/OpenShift environments, use Dockerfile.fips instead.
#
# Build args allow image override in CI:
#   --build-arg GO_IMAGE=${REGISTRY}/golang:1.25-alpine

ARG GO_IMAGE=golang:1.25-alpine
ARG RUNTIME_IMAGE=gcr.io/distroless/static:nonroot

# ------------------------------------------------------------------------------
# Stage 1: Build gt CLI from daedalus source
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS gt-builder

# Install git and ca-certificates (works on both Alpine and Debian)
RUN if command -v apk > /dev/null; then \
        apk add --no-cache git ca-certificates; \
    else \
        apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*; \
    fi
WORKDIR /src

# Clone and build gt CLI from upstream GitHub (public)
RUN git clone https://github.com/steveyegge/gastown.git . && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gt ./cmd/gt

# ------------------------------------------------------------------------------
# Stage 2: Build operator manager
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

# Install git and ca-certificates (works on both Alpine and Debian)
RUN if command -v apk > /dev/null; then \
        apk add --no-cache git ca-certificates; \
    else \
        apt-get update && apt-get install -y --no-install-recommends git ca-certificates && rm -rf /var/lib/apt/lists/*; \
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

# Copy binaries
COPY --from=gt-builder /out/gt /usr/local/bin/gt
COPY --from=builder /out/manager .

USER 65532:65532
ENTRYPOINT ["/manager"]
