# gastown-operator Dockerfile
# Multi-stage build for the Gas Town Kubernetes operator
#
# Build args allow DPR image override in CI:
#   --build-arg GO_IMAGE=${DPR_REGISTRY}/ci-images/golang:1.24-alpine

ARG GO_IMAGE=golang:1.24-alpine
ARG DISTROLESS_IMAGE=gcr.io/distroless/static:nonroot

# ------------------------------------------------------------------------------
# Stage 1: Build gt CLI from daedalus source
# ------------------------------------------------------------------------------
FROM ${GO_IMAGE} AS gt-builder

RUN apk add --no-cache git ca-certificates
WORKDIR /src

# Clone and build gt CLI
RUN git clone https://git.deepskylab.io/olympus/daedalus.git . && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags="-s -w" -o /out/gt ./cmd/gt

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

# Build
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags="-s -w" -o /out/manager cmd/main.go

# ------------------------------------------------------------------------------
# Stage 3: Minimal runtime image
# ------------------------------------------------------------------------------
FROM ${DISTROLESS_IMAGE}

WORKDIR /

# Copy binaries
COPY --from=gt-builder /out/gt /usr/local/bin/gt
COPY --from=builder /out/manager .

USER 65532:65532
ENTRYPOINT ["/manager"]
