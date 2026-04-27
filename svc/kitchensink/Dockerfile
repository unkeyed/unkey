# Build context is the repo root, not svc/kitchensink. The binary is
# part of the main module so `go build` needs go.mod there. Build with:
#
#   docker build -f svc/kitchensink/Dockerfile -t kitchensink .

FROM golang:1.25-alpine AS builder

# UNKEY_SECRETS_ID is supplied by the Unkey builder as a hash of the
# project's variables.
ARG UNKEY_SECRETS_ID

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY svc/kitchensink ./svc/kitchensink

# VERSION is sourced from the mounted .env file and baked into
# buildinfo.Version via -ldflags -X, then served at GET /buildinfo. The
# -f guard keeps local builds working when no secret is mounted.
RUN --mount=type=secret,id=${UNKEY_SECRETS_ID},target=/run/secrets/.env \
    if [ -f /run/secrets/.env ]; then set -a && . /run/secrets/.env && set +a; fi && \
    CGO_ENABLED=0 GOOS=linux go build \
      -ldflags "-X 'github.com/unkeyed/unkey/svc/kitchensink/buildinfo.Version=${VERSION:-unset}'" \
      -o /kitchensink ./svc/kitchensink

FROM gcr.io/distroless/static-debian12

COPY --from=builder /kitchensink /kitchensink

ENV PORT=8080
EXPOSE 8080

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Kitchensink — stdlib HTTP server for testing platform features"

ENTRYPOINT ["/kitchensink"]
