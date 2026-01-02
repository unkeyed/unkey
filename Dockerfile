FROM golang:1.25 AS builder

WORKDIR /go/src/github.com/unkeyed/unkey

COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION
ENV CGO_ENABLED=0
RUN go build -o bin/unkey -ldflags="-X 'github.com/unkeyed/unkey/pkg/version.Version=${VERSION}'" ./main.go

FROM gcr.io/distroless/static-debian12
COPY --from=builder /go/src/github.com/unkeyed/unkey/bin/unkey /

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Unkey API"

ENTRYPOINT  ["/unkey"]
