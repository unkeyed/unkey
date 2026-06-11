# Build from the repository root:
#   docker build -t unkey/frontline:dev -f svc/frontline-rs/Dockerfile .
FROM rust:1.90-bookworm AS builder
WORKDIR /build
COPY Cargo.toml Cargo.lock ./
COPY svc/frontline-rs ./svc/frontline-rs
RUN cargo build --release -p unkey-frontline

FROM debian:bookworm-slim
COPY --from=builder /build/target/release/unkey-frontline /usr/local/bin/unkey-frontline
ENTRYPOINT ["/usr/local/bin/unkey-frontline"]
