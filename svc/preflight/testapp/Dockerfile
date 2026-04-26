# testapp: the synthetic HTTP server preflight probes exercise.
#
# Build context is the repo root. The binary is part of the main
# module so `go build` needs go.mod there. Build with:
#
#   docker build -f cmd/preflight/testapp/Dockerfile -t preflight-testapp .
#
# The resulting image is deployed through the normal unkey deploy
# pipeline by preflight itself; probes then talk to its endpoints.

FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd/preflight/testapp ./cmd/preflight/testapp

RUN CGO_ENABLED=0 GOOS=linux go build -o /testapp ./cmd/preflight/testapp

FROM gcr.io/distroless/static-debian12

COPY --from=builder /testapp /testapp

# PORT is injected by Krane at deploy time; default to 8080 for local
# docker run / manual testing outside the preflight workspace.
ENV PORT=8080
EXPOSE 8080

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Preflight test app — synthetic workload deployed through the preflight pipeline."

USER 65532:65532

ENTRYPOINT ["/testapp"]
