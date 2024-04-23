# use the official Bun image
# see all versions at https://hub.docker.com/r/oven/bun/tags
FROM golang:latest as builder
WORKDIR /usr/src/app

COPY . .
RUN go mod tidy
RUN go build  -o tinybird-proxy ./cmd/main.go

# run the app


FROM golang:latest
WORKDIR /usr/src/app
COPY --from=builder /usr/src/app/tinybird-proxy ./tinybird-proxy

ARG PORT
EXPOSE ${PORT:-8080}

ENTRYPOINT [ "./tinybird-proxy" ]