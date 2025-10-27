FROM golang:1.25 AS builder

WORKDIR /go/src/github.com/unkeyed/unkey/go

COPY go.mod go.sum ./
RUN go mod download
COPY . .
ARG VERSION
ENV CGO_ENABLED=0
RUN go build -o bin/unkey -ldflags="-X 'github.com/unkeyed/unkey/go/pkg/version.Version=${VERSION}'" ./main.go

FROM golang:1.25
COPY --from=builder /go/src/github.com/unkeyed/unkey/go/bin/unkey /

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey/go
LABEL org.opencontainers.image.description="Unkey API"

ENTRYPOINT  ["/unkey"]
