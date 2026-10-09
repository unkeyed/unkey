# Phase 0 findings

This note is the repo-side map for the WorkOS staging spike. It does not
change `svc/api`. The later design is unchanged: the MCP server verifies the
WorkOS token, then mints its own short-lived internal JWT for `svc/api`.
Token passthrough stays forbidden.

## WorkOS audience and the auth chain

`api.unkey.com` is a constant in this repo, not a value read from WorkOS.

- `pkg/auth/jwt/claims.go` line 8 defines `Audience` as `api.unkey.com` for
  dashboard-minted API bearer JWTs.
- The dashboard proxy mints the fallback token in
  `web/apps/dashboard/app/proxy/[...path]/route.ts`. Lines 150 to 168 set
  `iss` to `app.unkey.com`, `aud` to the one-element array
  `["api.unkey.com"]`, a nested `org.id`, `roles`, `sub`, and a 120 second
  lifetime (`now + 120` on line 167). Lines 160 to 162 say this audience
  mirrors the WorkOS JWT template so `svc/api` can verify the fallback the
  same way it verifies a forwarded WorkOS access token. The template JSON
  itself is not in the repo. The only statement of what the template emits
  is that comment.
- Checked-in `[[auth]]` chains exist in two places, and they match. Both are
  local development configs. `dev/k8s/manifests/api.yaml` lines 20 to 31 and
  `web/apps/dashboard/dev/api.toml` lines 5 to 16 are, in order:
  1. `type = "jwt"`, `issuer = "app.unkey.com"`, `audience = "api.unkey.com"`,
     HS256 `secrets`, `provider = "workos"`.
  2. `type = "portal_session"`.
  3. `type = "root_key"`.
- There is no `deploy/` tree and no other checked-in API config with
  `[[auth]]`. Staging and production auth entries are not in this
  repository, so their issuer, audience, and key source are unverified here.
- The JWT entry shape is `svc/api/config.go` lines 84 to 108. `issuer` is
  required. `audience` is optional. Exactly one of `secrets` (HS256) or
  `jwks_url` (RS256) is allowed (`config.go` lines 441 to 471,
  `run.go` lines 399 to 403). `provider = "workos"` wraps the resolver with
  role mapping (`run.go` lines 408 to 410). No checked-in config sets
  `jwks_url`. The dev chain verifies the dashboard's HS256 tokens. It does
  not fetch `https://<authkit-domain>/oauth2/jwks`.
- `svc/api/run.go` lines 386 to 420 builds one resolver per `[[auth]]` entry,
  in file order. `pkg/auth/service.go` lines 37 to 43 keeps going after a
  resolver error, so a JWT rejection does not by itself skip the root-key
  resolver. A Connect token is not a root key, so the JWT error is still
  what the client sees when nothing else claims it.
- `pkg/jwt/doc.go` line 125 says an `aud` claim serialized as a bare JSON
  string fails to parse. `pkg/jwt/claims.go` line 69 types `aud` as
  `[]string`. The dashboard mints an array. WorkOS Connect tokens use a
  string. The spike verifier accepts both. `svc/api` does not.

## Connect tokens versus `pkg/auth/jwt`

`pkg/auth/jwt/resolver.go` lines 139 to 146 require `sub` (or a nested
`user.id`), a nested `org.id`, `exp`, and `iat`. The claims struct is
`pkg/auth/jwt/claims.go` lines 11 to 25: nested `org`, optional nested
`user`, `name`, and `roles`. On success, lines 194 to 210 build a user
principal whose `JWTSource` keeps the decoded header, payload, `roles`, and
the raw signature.

A WorkOS Connect token, as described for this project, misses that shape in
three ways:

- Organization is a flat `org_id`. `claims.Org.ID` stays empty, so line 141
  rejects the token even when the signature, issuer, and audience are valid.
- There is no `roles` claim. `pkg/auth/workos/resolver.go` lines 23 to 38
  sets `Permissions` only from `JWTSource.Roles` through
  `permissionsForRoles`. An empty role list grants nothing.
  `pkg/auth/workos/permissions.go` lines 121 to 130 map `admin` to `**`,
  `developer` and `basic_member` to the developer set, and `viewer` to the
  read set. Unknown roles add nothing.
- `aud` is the requested resource, or the environment client id when no
  Resource Indicator is registered. The dev JWT entry requires
  `api.unkey.com`. A string `aud` also fails to decode before that
  comparison, because of the `pkg/jwt` limitation above.

Smallest later `svc/api` change, not implemented here. The MCP server mints
an internal HS256 JWT the existing entry can already verify: `iss`
`app.unkey.com`, `aud` `api.unkey.com`, nested `org.id`, `sub`, `exp`,
`iat`, and `roles`. Connect tokens have no roles, so that minted token is
what carries the role slugs. Add two optional claims on
`pkg/auth/jwt.Claims`: a `max_permissions` ceiling, and the MCP `client_id`
plus `sid`. In `resolverWithRoles.Resolve`, after `permissionsForRoles`,
keep a role permission only when the ceiling authorizes it. Use the existing
permission check rather than string equality. `admin` expands to one
wildcard permission, and a raw string intersection would drop a narrower
ceiling or keep only an identical wildcard. Absent `max_permissions` leaves
today's dashboard behavior. Present and empty grants nothing. Copy
`client_id` and `sid` onto the principal. Do not copy the WorkOS signature
or the bearer token into the principal.

## Where a principal shows up in audit logs

`svc/api/internal/auditactor/actor.go` lines 43 to 55 copies
`Subject.Type`, `Subject.ID`, and `Subject.Name` into the audit actor and
sets `Meta` to an empty map. Portal sessions are the only source that
changes the actor type. `JWTSource` payload, roles, and signature are not
copied.

Compute mutations that go through ctrl use `svc/api/internal/ctrlclient/actor.go`
lines 13 to 26. That builds `ActorInfo` from the same subject fields and
sets `Meta` to an empty map. `svc/ctrl/internal/actor/actor.go` lines 12 to
33 turns that into `user`, `rootkey`, `github`, or `system`, and copies
`Meta` through. An empty map stays empty. There is no MCP actor type.

The row stores `actor_type`, `actor_id`, `actor_name`, and `actor_meta`
(`pkg/clickhouse/schema/025_audit_logs_raw_v1.sql` lines 44 to 47). The
dashboard query selects `actor_meta` (`web/internal/clickhouse/src/audit-logs.ts`
line 71) and then drops it. `web/apps/dashboard/lib/trpc/routers/audit/fetch.ts`
lines 135 to 138 keep only `id`, `name`, and `type`. The actor cell
(`web/apps/dashboard/components/audit-logs-table/components/cells/actor-cell.tsx`
lines 8 to 31) and the log footer
(`web/apps/dashboard/app/(app)/[workspaceSlug]/audit/components/table/log-details/components/log-footer.tsx`
lines 13 to 44) show a user profile when `type` is `user`, and otherwise
show `actor.id`. They do not render `actor_meta`.

A JWT principal is `SubjectTypeUser` with `Subject.ID` set to `sub`
(`pkg/auth/jwt/resolver.go` lines 194 to 200). The actor cell therefore
looks up that user and shows the person's name. `client_id` and `sid` would
be invisible there if they live only on `JWTSource` or in `actor_meta`.
Putting them in `Subject.ID` would replace the user. The audit change above
should write them to `actor.meta` from both `auditactor.FromPrincipal` and
`ctrlclient.Actor`. They will be in ClickHouse. The current dashboard will
not show them until a later UI change.

## Compute operations in the OpenAPI bundle

Counts are from `svc/api/openapi/openapi-generated.yaml`, which has 99
operations. The document-level `tags` list at lines 18919 to 18951 omits
`projects` and `rootKeys` even though operations use those tags.

| Tag | Count | Operations |
| --- | --- | --- |
| projects | 5 | `projects.createProject` (16735), `projects.deleteProject` (16860), `projects.getProject` (16976), `projects.listProjects` (17091), `projects.updateProject` (17215) |
| apps | 5 | `apps.createApp` (9241), `apps.deleteApp` (9395), `apps.getApp` (9513), `apps.listApps` (9633), `apps.updateApp` (9781) |
| environments | 6 | `environments.getEnvironment` (12192), `environments.listEnvironmentVariables` (12340), `environments.listEnvironments` (12416), `environments.removeEnvironmentVariables` (12592), `environments.setEnvironmentVariables` (12686), `environments.updateSettings` (12803) |
| deployments | 9 | eight `/v2/deployments.*` operations (10056 through 10892) plus `/v3/deployments.createDeployment` as `deployments.createDeploymentV3` (18853) |
| deploy | 2 | legacy `deploy.createDeployment` (9918) and `deploy.getDeployment` (9985) |
| domains | 5 | `domains.createDomain` (10990), `domains.deleteDomain` (11259), `domains.getDomain` (11441), `domains.listDomains` (11721), `domains.verifyDomain` (12010) |
| gateway | 3 | `gateway.listPolicies` (12926), `gateway.setPolicies` (13014), `gateway.updatePolicy` (13134) |
| github | 1 | `github.installApp` (13241) |
| analytics, named reads | 2 of 4 | `analytics.getGatewayRequests` (8645) and `analytics.getRuntimeLogs` (8795). The other two are `analytics.getRatelimits` (8719) and `analytics.getVerifications` (8870) |

Sensitive operations, by the categories in the brief:

- Root keys, tag `rootKeys`, five operations: `rootKeys.createKey` (18048),
  `rootKeys.deleteKey` (18097), `rootKeys.listKeys` (18164),
  `rootKeys.rerollKey` (18227), `rootKeys.updateKey` (18293). Create and
  reroll return the root key secret once. These are outside the Compute tags
  above. An `/api` server that exposes the whole v2 API includes them. A
  Compute allowlist limited to the tags above does not.
- There is no `keys.decrypt` operation. Plaintext recovery is a `decrypt`
  flag on `apis.listKeys` (operation 9162, warning at line 9149) and
  `keys.getKey` (operation 14153, warning at line 14137). It returns the
  plaintext only for keys created recoverable. `keys.createKey` (14004) and
  `keys.rerollKey` (14454) return the secret once.
- `environments.listEnvironmentVariables` (12340) returns decrypted
  plaintext for recoverable variables (line 12331). `writeonly` variables
  return no value. `environments.setEnvironmentVariables` (12686) writes
  those values. `environments.removeEnvironmentVariables` (12592) deletes
  them.
- Deletes: `apis.deleteApi` (9012), `apps.deleteApp` (9395),
  `domains.deleteDomain` (11259), `identities.deleteIdentity` (13423),
  `keys.deleteKey` (14075), `permissions.deletePermission` (15220),
  `permissions.deleteRole` (15292), `portal.deletePortal` (16065),
  `projects.deleteProject` (16860), `ratelimit.deleteOverride` (17348),
  `rootKeys.deleteKey` (18097).

`github.installApp` is the only GitHub operation. It starts an app install,
so it belongs on the sensitive side of a Compute allowlist even though it is
not a delete or a decrypt.
