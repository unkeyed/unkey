# see https://goreleaser.com/errors/docker-build/#do

FROM scratch
COPY unkey /unkey
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey/go
LABEL org.opencontainers.image.description="Unkey API"

ENTRYPOINT  ["/unkey"]
