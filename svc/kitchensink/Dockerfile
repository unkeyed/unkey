# Build context is the repo root, not svc/kitchensink. The binary is
# part of the main module so `go build` needs go.mod there. Build with:
#
#   docker build -f svc/kitchensink/Dockerfile -t kitchensink .

FROM ubuntu:24.04@sha256:c4a8d5503dfb2a3eb8ab5f807da5bc69a85730fb49b5cfca2330194ebcc41c7b AS builder

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
ENV GOPATH=/go \
    MISE_DATA_DIR=/opt/mise \
    MISE_INSTALL_PATH=/usr/local/bin/mise

COPY dev/install-mise ./dev/install-mise
RUN ./dev/install-mise
COPY .mise/config.toml .mise/mise.lock ./.mise/
RUN mise trust .mise/config.toml && mise install --locked --yes go

COPY go.mod go.sum ./
RUN --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    mise exec -- go mod download

COPY svc/kitchensink ./svc/kitchensink

# UNKEY_SECRETS_ID is supplied by the Unkey builder as a hash of the
# project's variables.
ARG UNKEY_SECRETS_ID

# VERSION is sourced from the mounted .env file and baked into
# buildinfo.Version via -ldflags -X, then served at GET /buildinfo. The
# -f guard keeps local builds working when no secret is mounted.
RUN --mount=type=secret,id=${UNKEY_SECRETS_ID},target=/run/secrets/.env \
    --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=unkey-go-build,target=/root/.cache/go-build \
    if [ -f /run/secrets/.env ]; then set -a && . /run/secrets/.env && set +a; fi && \
    CGO_ENABLED=0 GOOS=linux mise exec -- go build \
      -ldflags "-X 'github.com/unkeyed/unkey/svc/kitchensink/buildinfo.Version=${VERSION:-unset}'" \
      -o /kitchensink ./svc/kitchensink

FROM gcr.io/distroless/static-debian13:nonroot@sha256:d29e660cc75a5b6b1334e03c5c81ccf9bc0884a002c6000dbf0fb96034814478

COPY --from=builder /kitchensink /kitchensink

ENV PORT=8080
EXPOSE 8080

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Kitchensink — stdlib HTTP server for testing platform features"

ENTRYPOINT ["/kitchensink"]
