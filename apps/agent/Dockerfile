FROM golang:1.21 AS builder

WORKDIR /go/src/github.com/unkeyed/unkey/apps/agent

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

ARG VERSION

RUN go build -o unkey -ldflags "-X 'github.com/unkeyed/unkey/apps/agent/pkg/version.Version=${VERSION}'" .

FROM golang:1.20

COPY --from=builder /go/src/github.com/unkeyed/unkey/apps/agent/unkey /usr/local/bin/unkey


CMD ["/usr/local/bin/unkey", "agent"]
