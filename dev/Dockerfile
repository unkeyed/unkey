# syntax=docker/dockerfile:1.7

FROM ubuntu:25.10@sha256:7cc5e35f6567ee8c66d2abb4aab0fd866669e6207c237c3a8f0947a5c7f17092 AS toolchain

RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates curl git \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /src
ENV CGO_ENABLED=0 \
    GOPATH=/go \
    MISE_DATA_DIR=/opt/mise \
    MISE_INSTALL_PATH=/usr/local/bin/mise

COPY dev/install-mise ./dev/install-mise
RUN ./dev/install-mise
COPY .mise/config.toml .mise/mise.lock ./.mise/
RUN mise trust .mise/config.toml && mise install --locked --yes go

FROM toolchain AS builder

COPY go.mod go.sum ./
RUN --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    mise exec -- go mod download

# Dashboard services only depend on Go packages in these module directories.
# Keeping web and docs out prevents frontend edits from invalidating the build.
COPY build ./build
COPY gen ./gen
COPY internal ./internal
COPY pkg ./pkg
COPY svc ./svc

RUN --mount=type=cache,id=unkey-go-build,target=/root/.cache/go-build \
    --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    mise exec -- go build -o /out/ \
      ./build/api \
      ./build/control-api \
      ./build/control-worker \
      ./build/vault

FROM gcr.io/distroless/static-debian13:nonroot@sha256:516757106de35205efb70b91ba03f6f1298424a0678c603255cf2852a7000087 AS certs

FROM busybox:1.36@sha256:73aaf090f3d85aa34ee199857f03fa3a95c8ede2ffd4cc2cdb5b94e566b11662 AS runtime

COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
ENTRYPOINT ["/unkey"]

FROM runtime AS api
COPY --from=builder /out/api /unkey

FROM runtime AS control-api
COPY --from=builder /out/control-api /unkey

FROM runtime AS control-worker
COPY --from=builder /out/control-worker /unkey

FROM runtime AS vault
COPY --from=builder /out/vault /unkey
