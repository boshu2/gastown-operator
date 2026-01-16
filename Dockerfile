# ==============================================================================
# gastown-operator Dockerfile
# ==============================================================================
# Multi-stage build that includes:
# 1. gt CLI binary (built from daedalus source)
# 2. operator manager binary
#
# Uses DPR-mirrored images (DPR added to cluster allowedRegistries).
# ==============================================================================

ARG DPR_REGISTRY=dprusocplvjmp01.deepsky.lab:5000

# ------------------------------------------------------------------------------
# Stage 1: Build gt CLI from daedalus (gastown) source
# ------------------------------------------------------------------------------
FROM ${DPR_REGISTRY}/ci-images/golang:1.24 AS gt-builder

WORKDIR /gastown
# Clone daedalus (gastown) repo and build gt CLI
# Using HTTPS for CI compatibility (no SSH keys in container)
RUN git clone https://git.deepskylab.io/olympus/daedalus.git . && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o gt ./cmd/gt

# ------------------------------------------------------------------------------
# Stage 2: Build the operator manager binary
# ------------------------------------------------------------------------------
FROM ${DPR_REGISTRY}/ci-images/golang:1.24 AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /workspace

# Copy go modules first for layer caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -a -o manager cmd/main.go

# ------------------------------------------------------------------------------
# Stage 3: Final minimal image
# ------------------------------------------------------------------------------
FROM ${DPR_REGISTRY}/ci-images/distroless-static:nonroot

WORKDIR /

# Copy gt CLI from stage 1
COPY --from=gt-builder /gastown/gt /usr/local/bin/gt

# Copy operator manager from stage 2
COPY --from=builder /workspace/manager .

# Run as non-root (OpenShift compatible)
USER 65532:65532

ENTRYPOINT ["/manager"]
