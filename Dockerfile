# gastown-operator Dockerfile (Community Edition)
# Multi-stage build for vanilla Kubernetes environments
#
# Lightweight build using Alpine and distroless images.
# For FIPS/OpenShift environments, use Dockerfile.fips instead.
#
# Build args allow image override in CI:
#   --build-arg GO_IMAGE=${REGISTRY}/golang:1.24-alpine

ARG GO_IMAGE=golang:1.24-alpine
ARG RUNTIME_IMAGE=gcr.io/distroless/static:nonroot

# ------------------------------------------------------------------------------
# Stage 1: Build gt CLI from daedalus source
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS gt-builder

ARG GIT_TOKEN=""

RUN apk add --no-cache git ca-certificates
WORKDIR /src

# Clone and build gt CLI (use token if provided for private repos)
RUN if [ -n "${GIT_TOKEN}" ]; then \
      git clone https://oauth2:${GIT_TOKEN}@git.deepskylab.io/olympus/daedalus.git . ; \
    else \
      git clone https://git.deepskylab.io/olympus/daedalus.git . ; \
    fi && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gt ./cmd/gt

# ------------------------------------------------------------------------------
# Stage 2: Build operator manager
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

RUN apk add --no-cache git ca-certificates
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build with standard Go crypto
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/manager cmd/main.go

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
