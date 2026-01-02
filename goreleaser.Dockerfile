FROM alpine:latest as certs
RUN apk --no-cache add ca-certificates

# see https://goreleaser.com/errors/docker-build/#do

FROM scratch

COPY unkey /unkey
COPY --from=certs /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Unkey API"

ENTRYPOINT  ["/unkey"]
