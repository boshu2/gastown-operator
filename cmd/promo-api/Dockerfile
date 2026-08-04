# Promo API - thin HTTP front door for Gastown promo script jobs
FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache git ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/promo-api ./cmd/promo-api

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/promo-api /promo-api
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/promo-api"]
