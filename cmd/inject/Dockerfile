# Build stage
FROM golang:1.25 AS builder

RUN apt-get update && apt-get install -y upx-ucl

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary with all optimizations
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /inject ./cmd/inject \
    && upx --best --lzma /inject

# Final stage - minimal image with CA certs for HTTPS calls
FROM alpine:3.19

RUN apk --no-cache add ca-certificates

COPY --from=builder /inject /inject

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Unkey Secrets Injector"

ENTRYPOINT ["/inject"]
