FROM golang:1.23-alpine AS builder


WORKDIR /go/src/github.com/unkeyed/unkey/apps/chproxy
COPY go.mod ./
# COPY go.sum ./
# RUN go mod download

COPY . .
RUN go build -o bin/chproxy ./main.go


FROM golang:1.23-alpine
RUN apk add --update curl

WORKDIR  /usr/local/bin
COPY --from=builder /go/src/github.com/unkeyed/unkey/apps/chproxy/bin/chproxy .

CMD [ "/usr/local/bin/chproxy"]
