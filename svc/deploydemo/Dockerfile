FROM gcr.io/distroless/static-debian13:nonroot@sha256:d29e660cc75a5b6b1334e03c5c81ccf9bc0884a002c6000dbf0fb96034814478
ARG TARGETARCH
COPY ${TARGETARCH}/deploy-demo /deploy-demo
ENV PORT=8080
EXPOSE 8080
LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="A small Zen API for Unkey Deploy and gateway API key authentication"
ENTRYPOINT ["/deploy-demo"]
