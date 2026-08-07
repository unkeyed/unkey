# syntax=docker/dockerfile:1.7

FROM golang:1.26.5@sha256:2005724102f45917a63e9d092fc0e4ea56ea575048ce147caad5f5f61502c365 AS builder

WORKDIR /src
ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download
COPY . .

RUN go build -o /out/unkey .

FROM gcr.io/distroless/static-debian13:nonroot@sha256:d29e660cc75a5b6b1334e03c5c81ccf9bc0884a002c6000dbf0fb96034814478

COPY --from=builder /out/unkey /unkey
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
ENTRYPOINT ["/unkey"]
