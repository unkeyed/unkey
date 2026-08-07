# syntax=docker/dockerfile:1.7

FROM golang:1.26.5-alpine@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2 AS builder

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

FROM gcr.io/distroless/static-debian13:nonroot@sha256:d29e660cc75a5b6b1334e03c5c81ccf9bc0884a002c6000dbf0fb96034814478 AS certs

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
