ARG CGO_ENABLED=0

FROM golang:1.25 AS builder

WORKDIR /go/src/github.com/unkeyed/unkey/go
# Copy everything first because go.mod has replace directives pointing to ./deploy/pkg/ modules
# This ensures all local dependencies are available before go mod download
COPY . .
RUN go mod download
ARG VERSION
ARG CGO_ENABLED
ENV CGO_ENABLED=${CGO_ENABLED}
RUN go build -o bin/unkey -ldflags="-X 'github.com/unkeyed/unkey/go/pkg/version.Version=${VERSION}'" ./main.go

# Use base image that supports CGO binaries when CGO is enabled, static when disabled
FROM gcr.io/distroless/base-debian12:latest AS base-1
FROM gcr.io/distroless/static-debian12:latest AS base-0

FROM base-${CGO_ENABLED} AS final

COPY --from=builder /go/src/github.com/unkeyed/unkey/go/bin/unkey /

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey/go
LABEL org.opencontainers.image.description="Unkey API"

ENTRYPOINT  ["/unkey"]
