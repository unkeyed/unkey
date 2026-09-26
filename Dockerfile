# syntax=docker/dockerfile:1.7

FROM golang:1.27.1@sha256:512690a5660563b57d37ecc31129e7f136e831db2aed24a1dbeb8ad7380dc0fa AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /out/unkey ./build/cli

FROM gcr.io/distroless/static-debian13:nonroot@sha256:e2e927ec666bae08560abb3c55d0659eceabb657f56b6782ab500a9fc7f555e3

COPY --from=builder /out/unkey /unkey
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
ENTRYPOINT ["/unkey"]
