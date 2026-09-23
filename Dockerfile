# syntax=docker/dockerfile:1.7

FROM golang:1.25@sha256:699337d620559a59b4a2bb298ad59611e535d2ee755a34cf2d2a98f37578dc80 AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /out/unkey ./build/cli

FROM gcr.io/distroless/static-debian13:nonroot@sha256:d29e660cc75a5b6b1334e03c5c81ccf9bc0884a002c6000dbf0fb96034814478

COPY --from=builder /out/unkey /unkey
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
ENTRYPOINT ["/unkey"]
