# Build context is the repo root, not svc/kitchensink. The binary is
# part of the main module so `go build` needs go.mod there. Build with:
#
#   docker build -f svc/kitchensink/Dockerfile -t kitchensink .

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY svc/kitchensink ./svc/kitchensink

RUN CGO_ENABLED=0 GOOS=linux go build -o /kitchensink ./svc/kitchensink

FROM gcr.io/distroless/static-debian12

COPY --from=builder /kitchensink /kitchensink

ENV PORT=8080
EXPOSE 8080

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Kitchensink — stdlib HTTP server for testing platform features"

ENTRYPOINT ["/kitchensink"]
