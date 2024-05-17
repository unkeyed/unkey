FROM golang:1.22-alpine as builder


RUN go install github.com/go-task/task/v3/cmd/task@latest


WORKDIR /go/src/github.com/unkeyed/unkey/apps/vault
COPY go.sum go.mod ./
RUN go mod download

COPY . .
RUN go build -o bin/vault .

FROM golang:1.22-alpine
WORKDIR  /usr/local/bin
COPY --from=builder /go/src/github.com/unkeyed/unkey/apps/vault/bin/vault .

CMD ["vault"]