FROM golang:1.23 AS builder



WORKDIR /go/src/github.com/unkeyed/unkey/go
COPY go.sum go.mod ./
RUN go mod download

COPY . .
ARG VERSION
ENV CGO_ENABLED=0
RUN go build -o bin/unkey -ldflags "-X 'github.com/unkeyed/unkey/go/pkg/version.Version=${VERSION}'"  ./main.go

FROM gcr.io/distroless/static-debian12
# FROM scratch
COPY --from=builder /go/src/github.com/unkeyed/unkey/go/bin/unkey /
COPY --from=builder /go/src/github.com/unkeyed/unkey/go/config.docker.json /
COPY --from=builder /go/src/github.com/unkeyed/unkey/go/config.ecs.json /


ENTRYPOINT  [ "/unkey"]
