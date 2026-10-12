# Frontline

- Strip client `X-Unkey-*` headers and trailers before trusting policy identity.
  Retain peer metadata only for verification, then remove it before forwarding
  to customer workloads. Trailers can appear after the body is read, so the
  proxy repeats the check. See [sanitize_headers.go](middleware/sanitize_headers.go).
- Peer hops and client IP are trusted only after metadata verification. Preserve
  signature, expiry, size, and single-header checks, plus the signed-hop limit
  that stops cross-region forwarding loops. See
  [applyPeerMetadata](routes/proxy/handler.go) and [meta/codec.go](internal/meta/codec.go).
- Policies execute in order, and later policies can consume the principal from
  earlier authentication. Reordering or parallelizing them changes authorization
  behavior. See [policies/engine.go](internal/policies/engine.go).
- Automatic proxy retries are limited to dial failures before the request body
  is consumed. A timeout or mid-stream failure can follow a successful upstream
  side effect; replaying it risks duplicate execution. See
  [retry_test.go](routes/proxy/retry_test.go).

## Configuration documentation

- The [Notion configuration reference](https://app.notion.com/p/3f2512d643f38171828bd190d0ca67ff)
  can be generated from `Config` and its nested structs with
  `mise exec -- go run ./tools/configdocs --file svc/frontline/config.go --struct Config`
  from the repo root.
- Keep field comments and tags accurate when changing configuration. The tool
  publishes only struct data and comments; it does not infer runtime behavior.
  Keep deployment notes and design rationale on the human-owned service page.
