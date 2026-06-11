# unkey-frontline (Rust)

Rust port of `svc/frontline`: the multi-tenant ingress that terminates TLS
for customer domains, resolves hostnames to deployments, and forwards
requests to a running deployment instance in the local region — or to a peer
frontline in another region when no local instance exists.

The request hot path runs on tokio + hyper — an earlier hand-rolled
thread-per-connection HTTP stack was benchmarked against the Go service and
lost on both throughput and correctness under load (see BENCHMARKS.md; the
hyper gateway beats Go ~1.6-2.2× at half the memory). Everything off the hot
path stays hand-rolled: metrics registry with Prometheus text exposition,
structured JSON logging, SWR caches, encodings, uid generation, Go-style
duration parsing, and the small sync HTTP client used for vault/ctrl/
ClickHouse.

| Crate | Why |
| --- | --- |
| `tokio`, `hyper`, `hyper-util`, `http`, `http-body-util`, `bytes` | async request hot path |
| `rustls`, `tokio-rustls`, `hyper-rustls`, `rustls-pemfile`, `webpki-roots` | TLS termination and outbound TLS |
| `mysql` | MySQL wire protocol and authentication |
| `serde`, `serde_json` | Connect RPC payloads, ClickHouse rows, JSON error bodies |
| `toml` | config files, format-compatible with the Go service |

## Scope

Ported from the Go service:

- Hostname routing with SWR caches (`frontline_routes` + `instances` +
  region proximity table, shuffled local instances, standby peer region)
- Dial-failure retry across local instances: a request is only replayed
  when no connection was established (hyper connect errors with the body
  unpolled), so the body is untouched; mid-stream failures surface to the
  client unchanged
- Cross-region forwarding to `frontline.{region}.{apex_domain}` with hop
  counting / loop prevention and parent-tracking headers
- Dynamic TLS via SNI: certificates from the database, private keys
  decrypted through vault, SWR-cached with exact + wildcard candidates;
  static cert files for development; explicit disable
- ACME HTTP-01 challenge relay to the control plane (Connect RPC)
- HTTP -> HTTPS 308 redirects
- `X-Unkey-*` reserved header stripping at the edge
- Error rendering (HTML error page or JSON by Accept header) with stable
  URN codes and the same status mapping as the Go service
- Prometheus metrics (same names, labels, buckets) on a dedicated port
- ClickHouse request analytics (`sentinel_requests_raw_v1`) with buffered,
  drop-on-overflow inserts and Authorization redaction
- WebSocket / 101 upgrade tunneling
- Per-request timing headers (`X-Unkey-Timing`)

Every response additionally carries `X-Unkey-Frontline-Language: rust` so
traffic can be attributed during the Go -> Rust cutover.

The dev cluster (`mise run dev`) builds this crate via
`svc/frontline-rs/Dockerfile` (wired in dev/Tiltfile) and deploys it with the
unchanged `dev/k8s/manifests/frontline.yaml`: the ConfigMap parses verbatim —
`[redis]` and `[pprof]` are accepted and ignored with startup warnings — and
the `/_unkey/internal/health/{ready,live}` probe paths are served.

Out of scope (per project decision): the policy engine (keyauth, ratelimit,
firewall, openapi validation) — this port routes and proxies only.

## Known divergences from the Go service

- HTTP/1.1 only. Deployments configured with `upstream_protocol = h2c` are
  forwarded over HTTP/1.1 (the Go TransportRegistry already falls back to
  http1 for unknown protocols; this port extends that fallback to h2c).
- ClickHouse uses the HTTP interface (`clickhouse.url` should be an
  `http(s)://user:pass@host:8123` endpoint), not the native protocol.
- No OTLP tracing / tail sampling; `[observability]` config is accepted but
  inert. Logs are JSON lines on stdout (`UNKEY_LOG_LEVEL=debug|info|warn|error`).
- No graceful shutdown sequence; the process runs until killed.

## Running

```bash
cargo run -p unkey-frontline -- --config /etc/unkey/frontline.toml
```

Config is TOML-compatible with the Go service (Go-style durations like
`"15m"` are supported, and both `mysql://` URLs and Go DSNs
`user:pass@tcp(host:3306)/db` work):

```toml
platform = "local"
region = "dev"
http_port = 7070
https_port = 7443
prometheus_port = 9090
ctrl_addr = "localhost:8080"

[tls]
# disabled = true                  # or cert_file/key_file, or vault below
cert_file = "/path/cert.pem"
key_file = "/path/key.pem"

[database]
primary = "mysql://unkey:password@localhost:3306/unkey"
# readonly_replica = "..."

[vault]
url = "https://vault.internal"
token = "..."

[clickhouse]
url = "http://default:password@localhost:8123"
```

## Testing

```bash
cargo test -p unkey-frontline
```

Unit tests cover routing decisions, HTTP parsing/framing, SWR cache
semantics, error mapping, encodings, and the metrics registry. The proxy
path (TLS termination, dial-retry recovery, header sanitization, redirects,
error pages, metrics) was verified end-to-end against a seeded MySQL and a
live upstream.
