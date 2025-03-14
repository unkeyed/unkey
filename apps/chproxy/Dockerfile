FROM golang:1.24-alpine AS builder

WORKDIR /go/src/github.com/unkeyed/unkey/apps/chproxy
COPY go.mod ./

COPY . .
RUN go build -o bin/chproxy

FROM golang:1.24-alpine
RUN apk add --update curl

WORKDIR  /usr/local/bin
COPY --from=builder /go/src/github.com/unkeyed/unkey/apps/chproxy/bin/chproxy .

CMD ["/usr/local/bin/chproxy"]
