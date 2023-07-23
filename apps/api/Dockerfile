FROM golang:1.20 AS builder

WORKDIR /go/src/github.com/unkeyed/unkey/apps/api

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

ARG VERSION

RUN go build -o unkey -ldflags "-X 'github.com/unkeyed/unkey/apps/api/pkg/version.Version=${VERSION}'" ./cmd/api

FROM golang:1.20

COPY --from=builder /go/src/github.com/unkeyed/unkey/apps/api/unkey /usr/local/bin/unkey


CMD ["/usr/local/bin/unkey"]
