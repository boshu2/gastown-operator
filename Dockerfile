# gastown-operator Dockerfile
# Multi-stage build for the Gas Town Kubernetes operator
#
# FIPS-compliant build using UBI9 go-toolset with boringcrypto
#
# Build args allow DPR image override in CI:
#   --build-arg GO_IMAGE=${DPR_REGISTRY}/ci-images/go-toolset:1.22-ubi9

ARG GO_IMAGE=registry.access.redhat.com/ubi9/go-toolset:1.22
ARG RUNTIME_IMAGE=registry.access.redhat.com/ubi9/ubi-micro:9.3

# ------------------------------------------------------------------------------
# Stage 1: Build gt CLI from daedalus source
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS gt-builder

USER 0
RUN dnf install -y git && dnf clean all
WORKDIR /src

# Clone and build gt CLI with FIPS-compliant crypto
RUN git clone https://git.deepskylab.io/olympus/daedalus.git . && \
    CGO_ENABLED=1 GOEXPERIMENT=boringcrypto GOOS=linux GOARCH=amd64 \
    go build -trimpath -ldflags="-s -w" -o /out/gt ./cmd/gt

# ------------------------------------------------------------------------------
# Stage 2: Build operator manager
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS builder

ARG TARGETOS=linux
ARG TARGETARCH=amd64

USER 0
RUN dnf install -y git && dnf clean all
WORKDIR /src

# Cache deps
COPY go.mod go.sum ./
RUN go mod download

# Build with FIPS-compliant crypto (boringcrypto)
COPY . .
RUN CGO_ENABLED=1 GOEXPERIMENT=boringcrypto GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/manager cmd/main.go

# ------------------------------------------------------------------------------
# Stage 3: Minimal UBI runtime image
# ------------------------------------------------------------------------------
FROM ${RUNTIME_IMAGE}

WORKDIR /

# Copy binaries
COPY --from=gt-builder /out/gt /usr/local/bin/gt
COPY --from=builder /out/manager .

USER 65532:65532
ENTRYPOINT ["/manager"]
