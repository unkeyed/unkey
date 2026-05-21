FROM golang:1.25@sha256:cd05a378aaf011e8056745363e5c40f4f2bef0fa4d9bf19b9c38316079c332ff AS builder

# Install Bazelisk (which will download the correct Bazel version)
RUN go install github.com/bazelbuild/bazelisk@5a7ff3892fdbd05672e6b8e60f139e05b2b76940 # v1.29.0

WORKDIR /src

COPY . .

RUN bazelisk build //:unkey

# Extract the binary path and copy it to a known location
RUN cp $(bazelisk cquery //:unkey --output=files 2>/dev/null) /unkey

FROM gcr.io/distroless/static-debian12@sha256:9c346e4be81b5ca7ff31a0d89eaeade58b0f95cfd3baed1f36083ddb47ca3160
COPY --from=builder /unkey /unkey

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Unkey API"

ENTRYPOINT  ["/unkey"]
