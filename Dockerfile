# syntax=docker/dockerfile:1.7

FROM ubuntu:25.10@sha256:7cc5e35f6567ee8c66d2abb4aab0fd866669e6207c237c3a8f0947a5c7f17092 AS builder

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

COPY go.mod go.sum ./
RUN --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    mise exec -- go mod download
COPY . .

RUN --mount=type=cache,id=unkey-go-mod,target=/go/pkg/mod \
    --mount=type=cache,id=unkey-go-build,target=/root/.cache/go-build \
    mise exec -- go build -o /out/unkey ./build/cli

FROM gcr.io/distroless/static-debian13:nonroot@sha256:516757106de35205efb70b91ba03f6f1298424a0678c603255cf2852a7000087

COPY --from=builder /out/unkey /unkey
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
ENTRYPOINT ["/unkey"]
