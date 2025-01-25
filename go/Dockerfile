FROM golang:1.23-alpine AS builder



WORKDIR /go/src/github.com/unkeyed/unkey/go
COPY go.sum go.mod ./
RUN go mod download

COPY . .
ARG VERSION
RUN go build -o bin/unkey -ldflags "-X 'github.com/unkeyed/unkey/go/pkg/version.Version=${VERSION}'"  ./cmd/main.go

FROM golang:1.23-alpine
WORKDIR  /usr/local/bin
COPY --from=builder /go/src/github.com/unkeyed/unkey/go/bin/unkey .
COPY --from=builder /go/src/github.com/unkeyed/unkey/go/config.docker.json .

ENTRYPOINT [ "/usr/local/bin/unkey"]
