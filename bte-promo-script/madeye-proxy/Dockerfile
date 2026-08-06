FROM golang:1.25-alpine AS builder
WORKDIR /src
RUN apk add --no-cache ca-certificates
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/madeye-proxy ./bte-promo-script/madeye-proxy

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /out/madeye-proxy /madeye-proxy
USER 65532:65532
EXPOSE 4000
ENTRYPOINT ["/madeye-proxy"]
