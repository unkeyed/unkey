FROM golang:1.25 AS builder

# Install Bazelisk (which will download the correct Bazel version)
RUN go install github.com/bazelbuild/bazelisk@latest

WORKDIR /src

COPY . .

RUN bazelisk build //:unkey

# Extract the binary path and copy it to a known location
RUN cp $(bazelisk cquery //:unkey --output=files 2>/dev/null) /unkey

FROM gcr.io/distroless/static-debian12
COPY --from=builder /unkey /unkey

LABEL org.opencontainers.image.source=https://github.com/unkeyed/unkey
LABEL org.opencontainers.image.description="Unkey API"

ENTRYPOINT  ["/unkey"]
