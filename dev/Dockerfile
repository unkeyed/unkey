# syntax=docker/dockerfile:1.7

FROM golang:1.27.1-alpine@sha256:cf6fca6641884b8433441b2b0652976f975e1d0fdd26d177eaaf8596087f3125 AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    go mod download

# Dashboard services only depend on Go packages in these module directories.
# Keeping web and docs out prevents frontend edits from invalidating the build.
COPY build ./build
COPY gen ./gen
COPY internal ./internal
COPY pkg ./pkg
COPY svc ./svc

RUN --mount=type=cache,id=unkey-go-build,target=/root/.cache/go-build \
    --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    go build -o /out/ \
      ./build/api \
      ./build/control-api \
      ./build/control-worker \
      ./build/vault

FROM gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3 AS certs

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
