# API auth entries for Gram MCP

Gram forwards the user's WorkOS access token unchanged to `api.unkey.com`.
`svc/api` validates it. These entries are not in the local-dev configs. The
dashboard entry stays `issuer = "app.unkey.com"`, `audience = "api.unkey.com"`,
HS256 `secrets`, and `provider = "workos"`, with no `permission_ceiling`.

A `provider = "workos"` entry must set `audience`. Omit `permission_ceiling`
for no cap. An empty list is invalid. Patterns are `resource#action`, using
the same wildcards as role grants, and are bound to the token's workspace
when the request is authenticated. The v2 audience has no ceiling. The
Compute audience uses the developer role's Compute resources and excludes
keyspaces, identities, ratelimits, rbac, portals, limits, and usage.

`projects/*/apps/*/environments/*/variables/*#read` stays in the Compute
permission ceiling below. Viewer, developer, and admin roles can call
`environments.listEnvironmentVariables`, which returns decrypted recoverable
values. That is intentional: secrets stay readable through MCP.

## Staging

```toml
[[auth]]
type = "jwt"
provider = "workos"
issuer = "https://beautiful-day-63-staging.authkit.app"
jwks_url = "https://beautiful-day-63-staging.authkit.app/oauth2/jwks"
audience = "https://mcp.unkey.com/mcp/v2"

[[auth]]
type = "jwt"
provider = "workos"
issuer = "https://beautiful-day-63-staging.authkit.app"
jwks_url = "https://beautiful-day-63-staging.authkit.app/oauth2/jwks"
audience = "https://mcp.unkey.com/mcp/compute"
permission_ceiling = [
  "github/apps/*#read",
  "github/apps/*#write",
  "github/apps/*#delete",
  "projects/*#read",
  "projects/*#write",
  "projects/*#delete",
  "projects/*/apps/*#read",
  "projects/*/apps/*#write",
  "projects/*/apps/*#delete",
  "projects/*/apps/*/environments/*#read",
  "projects/*/apps/*/environments/*#write",
  "projects/*/apps/*/environments/*#delete",
  "projects/*/apps/*/environments/*/deployments/*#read",
  "projects/*/apps/*/environments/*/deployments/*#write",
  "projects/*/apps/*/environments/*/deployments/*#delete",
  "projects/*/apps/*/environments/*/deployments/*/logs#read",
  "projects/*/apps/*/environments/*/deployments/*/buildLogs#read",
  "projects/*/apps/*/environments/*/domains/*#read",
  "projects/*/apps/*/environments/*/domains/*#write",
  "projects/*/apps/*/environments/*/domains/*#delete",
  "projects/*/apps/*/environments/*/variables/*#read",
  "projects/*/apps/*/environments/*/variables/*#write",
  "projects/*/apps/*/environments/*/variables/*#delete",
  "projects/*/apps/*/environments/*/gateway/logs#read",
  "projects/*/apps/*/environments/*/gateway/policies/*#read",
  "projects/*/apps/*/environments/*/gateway/policies/*#write",
  "projects/*/apps/*/environments/*/gateway/policies/*#delete",
]
```

## Production

```toml
[[auth]]
type = "jwt"
provider = "workos"
issuer = "https://login.unkey.com"
jwks_url = "https://login.unkey.com/oauth2/jwks"
audience = "https://mcp.unkey.com/mcp/v2"

[[auth]]
type = "jwt"
provider = "workos"
issuer = "https://login.unkey.com"
jwks_url = "https://login.unkey.com/oauth2/jwks"
audience = "https://mcp.unkey.com/mcp/compute"
permission_ceiling = [
  "github/apps/*#read",
  "github/apps/*#write",
  "github/apps/*#delete",
  "projects/*#read",
  "projects/*#write",
  "projects/*#delete",
  "projects/*/apps/*#read",
  "projects/*/apps/*#write",
  "projects/*/apps/*#delete",
  "projects/*/apps/*/environments/*#read",
  "projects/*/apps/*/environments/*#write",
  "projects/*/apps/*/environments/*#delete",
  "projects/*/apps/*/environments/*/deployments/*#read",
  "projects/*/apps/*/environments/*/deployments/*#write",
  "projects/*/apps/*/environments/*/deployments/*#delete",
  "projects/*/apps/*/environments/*/deployments/*/logs#read",
  "projects/*/apps/*/environments/*/deployments/*/buildLogs#read",
  "projects/*/apps/*/environments/*/domains/*#read",
  "projects/*/apps/*/environments/*/domains/*#write",
  "projects/*/apps/*/environments/*/domains/*#delete",
  "projects/*/apps/*/environments/*/variables/*#read",
  "projects/*/apps/*/environments/*/variables/*#write",
  "projects/*/apps/*/environments/*/variables/*#delete",
  "projects/*/apps/*/environments/*/gateway/logs#read",
  "projects/*/apps/*/environments/*/gateway/policies/*#read",
  "projects/*/apps/*/environments/*/gateway/policies/*#write",
  "projects/*/apps/*/environments/*/gateway/policies/*#delete",
]
```

Place these after the existing dashboard `[[auth]]` entry. An audience or
issuer mismatch on one entry is an authentication failure for that entry.
The chain tries the next entry. A token is accepted only by the entry whose
audience matches.
