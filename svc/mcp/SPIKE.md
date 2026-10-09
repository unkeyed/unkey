# WorkOS AuthKit MCP spike

This process checks what WorkOS staging puts on an access token for two MCP
resources. It verifies the token and returns those claims from `whoami`. It
does not call `svc/api`, and it does not mint an internal API JWT.

Run it only in the WorkOS staging environment. Leave production Connect, the
production JWT template, and production DNS alone.

## What the process serves

`PUBLIC_BASE_URL` is an origin with no path and no trailing slash. For
`https://mcp.example.test` the process serves:

| Path | Resource identifier | Metadata |
| --- | --- | --- |
| `/api` | `https://mcp.example.test/api` | `/.well-known/oauth-protected-resource/api` |
| `/compute` | `https://mcp.example.test/compute` | `/.well-known/oauth-protected-resource/compute` |

Each metadata document sets `resource` to that identifier,
`authorization_servers` to `https://<AUTHKIT_DOMAIN>`, and
`bearer_methods_supported` to `header`. A missing or invalid bearer token is
a 401 with `WWW-Authenticate: Bearer resource_metadata="..."`.

A token is accepted only when the RS256 signature matches the AuthKit JWKS
at `https://<AUTHKIT_DOMAIN>/oauth2/jwks`, `iss` is that origin, `aud` equals
that path's resource identifier, `exp` is still in the future, and `org_id`
is present. `aud` may be a JSON string or an array. One array entry must
equal the resource identifier exactly.

`whoami` returns `sub`, `org_id`, `client_id`, `sid`, `scope`, `aud`, `iss`,
`exp`, `iat`, and `exp_minus_iat`. The bearer token is not returned.

Set `MCP_SPIKE_LOG_CLAIMS=true` to log claim names and values after the
signature check, including when `aud` does not match. The log does not
include the token or the signature. Claims are not logged when the signature
or `alg` check fails. On startup and on each JWKS fetch the process logs
`kid`, `kty`, `alg`, and `use` for the published keys. If that list has no
RS256 key, record the algorithms and stop. The spike cannot read claims from
a token it cannot verify.

## Configure the process

```bash
export AUTHKIT_DOMAIN="your-staging-authkit-host"
export PUBLIC_BASE_URL="https://your-public-origin"
export LISTEN_ADDR="127.0.0.1:8787"
export MCP_SPIKE_LOG_CLAIMS=true
mise exec -- go run ./svc/mcp/cmd/mcp-spike
```

`AUTHKIT_DOMAIN` is the staging AuthKit host, such as the host in
`https://<host>/.well-known/oauth-authorization-server`. You can pass the
host or the `https` origin. `LISTEN_ADDR` is plain HTTP. Terminate TLS at
the tunnel, and set `PUBLIC_BASE_URL` to the tunnel's `https` origin.

Confirm metadata before connecting a client:

```bash
curl -sS "$PUBLIC_BASE_URL/.well-known/oauth-protected-resource/api" | jq .
curl -sS "$PUBLIC_BASE_URL/.well-known/oauth-protected-resource/compute" | jq .
curl -sS -D - -o /dev/null "$PUBLIC_BASE_URL/api"
```

The last command must be a 401, and `WWW-Authenticate` must contain
`resource_metadata` for the `/api` document.

## Staging steps

Do these in the WorkOS staging environment. Record the environment name in
your notes so a production change is obvious if it happens.

1. Open **Connect**, then **Configuration**. Turn on Client ID Metadata
   Document (CIMD). Turn on Dynamic Client Registration (DCR). They are
   separate toggles, and both are off in production. Leave production off.
2. Add two Resource Indicators. The values must match the `resource` fields
   from the metadata documents, character for character:
   - `$PUBLIC_BASE_URL/api`
   - `$PUBLIC_BASE_URL/compute`
3. Open the menu on the `/compute` indicator and choose **Set as default**.
   Clients that omit `resource` then receive `aud` equal to the Compute URL.
   Clients that send `resource` still receive that requested value. The
   default applies to CIMD and DCR clients, not to a Connect application you
   create by hand. WorkOS has no update API for an existing indicator, so
   change the default in the dashboard.
4. Publish the spike on a public `https` origin. A tunnel is enough:

   ```bash
   cloudflared tunnel --url http://127.0.0.1:8787
   ```

   Copy the printed `https` origin into `PUBLIC_BASE_URL` and restart the
   spike. Register the Resource Indicators only after the origin is stable.
   If the tunnel hostname changes, update `PUBLIC_BASE_URL`, restart, and
   replace the indicators. Do not create a production DNS name.
5. Fetch AuthKit metadata and save it next to your notes:

   ```bash
   curl -sS "https://$AUTHKIT_DOMAIN/.well-known/oauth-authorization-server" | jq .
   ```

   Record `issuer`, `jwks_uri`, `scopes_supported`,
   `registration_endpoint`, and `grant_types_supported`.
6. Connect each client to both URLs. For each URL, try once while the
   client sends `resource`, and once while it omits `resource` if the client
   lets you control that. Use an account that belongs to two organizations
   so the org picker has a choice.
   - Claude: add a custom connector for `$PUBLIC_BASE_URL/api`, then repeat
     for `$PUBLIC_BASE_URL/compute`.
   - Cursor: add a remote MCP server for each URL.
   - ChatGPT developer mode: add a custom MCP connector for each URL.
   - MCP Inspector: `npx @modelcontextprotocol/inspector`, then connect to
     each URL.
7. Call `whoami` on each connection that authenticates. Copy the tool result
   into the table. When the call returns 401, copy the claim log instead.
   The log line is `mcp spike decoded jwt claims`.
8. Refresh once on a client that received a refresh token. Record whether
   the previous refresh token still works, whether the new access token has
   the same `aud`, and the new `exp_minus_iat`.
9. JWT template experiment, staging only. The template is environment-wide
   and also applies to AuthKit session tokens, so a dashboard login in that
   environment uses the same template. Read the template, mint one MCP token,
   then add one staging-only claim such as `spike` set to `1`. Mint a second
   MCP token and a dashboard session. Record whether the new claim appears,
   and whether `aud` changed, on both tokens. Remove the claim before you
   stop. Do not edit the production template.

Keep bearer tokens, refresh tokens, and client secrets out of git, chat, and
screenshots. The table stores claims, not tokens.

## Token table

Fill one row per client, URL, and resource behavior. Put the claim log's
JSON object in the claims cell, or the `whoami` object when the call
succeeds. `exp_minus_iat` is a number of seconds.

| Client | URL | Resource parameter | HTTP result | aud | Claims | exp minus iat | Refresh rotation |
| --- | --- | --- | --- | --- | --- | --- | --- |
| Claude | `/api` | sent | | | | | |
| Claude | `/api` | omitted | | | | | |
| Claude | `/compute` | sent | | | | | |
| Claude | `/compute` | omitted | | | | | |
| Cursor | `/api` | sent | | | | | |
| Cursor | `/api` | omitted | | | | | |
| Cursor | `/compute` | sent | | | | | |
| Cursor | `/compute` | omitted | | | | | |
| ChatGPT dev mode | `/api` | sent | | | | | |
| ChatGPT dev mode | `/api` | omitted | | | | | |
| ChatGPT dev mode | `/compute` | sent | | | | | |
| ChatGPT dev mode | `/compute` | omitted | | | | | |
| MCP Inspector | `/api` | sent | | | | | |
| MCP Inspector | `/api` | omitted | | | | | |
| MCP Inspector | `/compute` | sent | | | | | |
| MCP Inspector | `/compute` | omitted | | | | | |

When `resource` is omitted, the expected `aud` under current WorkOS docs is
the `/compute` indicator, because that indicator is the default. `/api` must
then return 401 `audience mismatch`, and the claim log must still show the
real `aud`. If `aud` is `api.unkey.com` or the environment client id, the
template or the missing Resource Indicator list is overriding the resource.
That result is the point of the spike. Record it. Do not "fix" the spike to
accept it.

## UX and template table

| Client | Org picker | Consent screen text | Template claim appeared on MCP token | Template changed MCP aud | Template changed dashboard session token |
| --- | --- | --- | --- | --- | --- |
| Claude | | | | | |
| Cursor | | | | | |
| ChatGPT dev mode | | | | | |
| MCP Inspector | | | | | |

For the org picker, write whether AuthKit asked you to choose an
organization, which org landed in `org_id`, and whether a second
organization produced a second `org_id`. For consent, copy the words on the
screen, including the client name and the scopes.

## Questions for WorkOS

Send these from the staging notes. CIMD and DCR clients currently receive
only `openid`, `profile`, `email`, and `offline_access`.

- Can an environment grant custom scopes to every CIMD and DCR client, or
  only to a Connect application created in the dashboard?
- Is there an API to list or revoke the consents a user has granted to MCP
  clients?
- Where is the access-token TTL configured, and does it differ for refresh
  grants? Can it be shorter than the dashboard session?
- Can Standalone Connect run in the same environment as hosted AuthKit, or
  does Standalone replace the hosted login?
- Do users who only authenticate through an MCP client count as MAU?

WorkOS documents the MCP setup at
<https://workos.com/docs/authkit/mcp>. Resource Indicators are also exposed
at `POST /user_management/authkit_oauth_resources`. Creating an indicator
with `"default": true` works once. Changing the default later is a dashboard
action.
