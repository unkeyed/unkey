FROM golang:1.22-alpine as builder



WORKDIR /go/src/github.com/unkeyed/unkey/apps/agent
COPY go.sum go.mod ./
RUN go mod download

COPY . .
RUN go build -o bin/agent .

FROM golang:1.22-alpine
WORKDIR  /usr/local/bin
COPY --from=builder /go/src/github.com/unkeyed/unkey/apps/agent/bin/agent .

CMD ["agent", "agent", "--enable-ratelimit"]


