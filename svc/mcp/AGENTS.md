# MCP

svc/mcp serves the Unkey API as MCP tools at mcp.unkey.com. It runs on Unkey
Deploy from [Dockerfile](Dockerfile) with the repository root as build context,
and reads its config from environment variables ([config.go](config.go)).

- Tools come only from the API's embedded OpenAPI spec. Every operation has an
  `x-unkey-mcp` extension with `toolsets` or `exclude`; never add a tool, a
  per-tool override, or a second list of operations in Go. After a spec change,
  run `mise run generate` and regenerate the golden file with
  `mise exec -- go test ./svc/mcp/internal/tools -run TestToolsetsGolden -update`.
- Tool names are a public contract: users allowlist them. They derive from the
  `operationId` by the Speakeasy rule in [name.go](internal/tools/name.go).
  Renaming an `operationId` renames a tool; treat it as a breaking change.
- Toolsets, their canonical and legacy paths, and everything derived from them
  (routes, OAuth resources, install pages) come from the one table in
  [toolset.go](internal/tools/toolset.go). The Speakeasy-era paths `/mcp/v2` and
  `/mcp/compute` and the `Mcp-Unkey-V2-*` headers must keep working.
- svc/mcp holds no secret, database, or vault client. It never verifies a
  credential: the API does, on every forwarded call. It reads unverified token
  claims only to answer 401 so OAuth clients refresh, see
  [caller.go](internal/caller/caller.go). Do not grant anything on those claims.
  After the API answers 401 to a session token, each replica keeps its SHA-256,
  never the token, so the next request gets that 401 too.
- Tool arguments are forwarded as the request body without validation; the
  API's 400 is the error the model should see. API failures are tool results
  with `isError`, never Go errors.
- Every tools/call writes one internal ClickHouse row,
  `mcp_tool_calls_raw_v1` ([analytics.go](internal/gateway/analytics.go)).
  Never store arguments or results: they can hold plaintext keys. Record a
  claimed workspace only after the API answered 2xx to that token.
