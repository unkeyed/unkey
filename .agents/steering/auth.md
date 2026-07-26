# API Authentication & Authorization Guide

The `pkg/auth` package provides unified request authentication for the API service. It chains multiple credential resolvers and produces a normalized `Principal` used by all route handlers for authorization.

## Architecture

```
Request → Middleware → Auth Chain → Principal → Handler → principal.Authorize(rbac.T(...))
                         │
                         ├── Root Key Resolver (Bearer → key hash → DB lookup)
                         ├── JWT/JWKS Resolver (Bearer → JWKS verification → claims)
                         └── Portal Session Resolver (Cookie → DB lookup)
```

## Core Types

### Principal (`pkg/auth/principal/`)

The normalized authenticated subject. Every protected handler receives this.

```go
type Principal struct {
    Version     string         // Schema version ("v1")
    Subject     Subject        // Who (ID, Name, SubjectType)
    Type        Type           // How authenticated (TypeAPIKey, TypeJWT, TypePortalSession)
    Source      Source         // Method-specific details (KeySource, JWTSource, PortalSessionSource)
    WorkspaceID string         // Scopes all DB queries
    Permissions []string       // Flat RBAC permission set (URN format)
}
```

Principal types:
- `TypeAPIKey` / `SubjectTypeRootKey`: Authenticated via root key hash lookup
- `TypeJWT` / `SubjectTypeUser`: Authenticated via JWKS-verified JWT (dashboard users)
- `TypePortalSession`: Authenticated via portal browser session cookie

### Resolver Interface (`pkg/auth/resolver.go`)

```go
type Resolver interface {
    Resolve(ctx context.Context, sess *zen.Session) (*principal.Principal, error)
}
```

Contract:
- Return `(nil, nil)` when the request has no credential this resolver understands
- Return `(nil, error)` when the credential matches but verification fails
- Return `(principal, nil)` when fully authenticated

### Auth Service (`pkg/auth/service.go`)

```go
svc := auth.New(rootKeyResolver, jwtResolver, portalResolver)
principal, err := svc.Authenticate(ctx, session)
```

The chain tries resolvers in order. First non-nil principal wins. On total failure, reports the most authoritative error (infrastructure > authorization > authentication).

## Resolvers

### Root Key (`pkg/auth/root_key/`)

- Extracts Bearer token from Authorization header
- Computes SHA256 hash of the token
- Looks up the key in the database by hash
- Populates permissions from key's RBAC grants
- Returns `(nil, nil)` if no Authorization header present (yields to next resolver)

### JWT/JWKS (`pkg/auth/jwt/`)

- Extracts Bearer token from Authorization header
- Verifies signature against configured JWKS endpoint (with key rotation)
- Validates audience and expiry claims
- Resolves workspace from JWT claims (org_id → workspace lookup)
- Translates provider-specific permissions (WorkOS slugs → Unkey URN permissions)
- JWKS fetcher: cold-start retries without backoff penalty, periodic refresh

Key config fields in API TOML:
```toml
[auth]
jwks_url = "https://api.workos.com/sso/jwks/..."  # Must be HTTPS
audience = "..."                                     # Required for audience binding
provider = "workos"                                  # Enables permission translation
```

### Portal Session (`pkg/auth/portal_session/`)

- Reads session cookie from the request
- Looks up browser session in MySQL
- Validates session expiry (24hr)
- Populates permissions from the session's RBAC grants
- Scopes to the portal config's workspace

## WorkOS Permission Translation (`pkg/auth/workos/`)

WorkOS uses permission slugs (e.g., `keys:create`, `admin:*`). Unkey uses URN-based permissions. The translation layer maps:

```
keys:create  → unkey:v1:<workspace_id>:keyspaces/*#create_key
admin:*      → unkey:v1:<workspace_id>:**#*
```

The mapping table lives in `pkg/auth/workos/permissions.go`. Only slugs with handler coverage are added. Unknown slugs are logged and skipped.

To sync definitions to WorkOS: `tools/upsert-workos-permissions`

## Authorization (RBAC)

### URN Format (`pkg/urn/`)

```
unkey:v1:<workspace_id>:<resource_path>
```

Resource paths use typed builders:
```go
urn.V1{WorkspaceID: "ws_123", Resource: "keyspaces/ks_456/keys/*"}
```

### Permission Format (`pkg/rbac/`)

```
<urn>#<action>
```

Example: `unkey:v1:ws_123:keyspaces/ks_456#create_key`

### Checking Permissions in Handlers

```go
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
    principal, err := s.GetPrincipal()
    if err != nil {
        return err
    }

    err = principal.Authorize(rbac.T(rbac.Tuple{
        ResourceType: rbac.Project,
        ResourceID:   "*",
        Action:       rbac.CreateProject,
    }))
    if err != nil {
        return err  // Returns 403
    }
    // ...
}
```

`rbac.T()` constructs a `PermissionQuery` from tuples. The query checks whether the principal's permission set satisfies the required tuple(s).

### Typed Permission Definitions (`pkg/rbac/permissions/`)

Each resource has a file with typed actions:
- `keys.go`: `create_key`, `read_key`, `update_key`, `delete_key`, `encrypt_key`, `decrypt_key`
- `keyspaces.go`: `create_keyspace`, `read_keyspace`, `delete_keyspace`
- `projects.go`: `create_project`, `read_project`, `update_project`, `delete_project`
- `ratelimit_namespaces.go`: actions for rate limit namespace management
- `ratelimit_overrides.go`: actions for override CRUD
- `environments.go`: actions scoped to deployment environments
- `apps.go`: actions for app management

## Middleware Integration (`svc/api/internal/middleware/`)

The `WithAuthentication` middleware:
1. Calls `auth.Authenticate(ctx, session)`
2. On success, the principal is set on the session (`sess.SetPrincipal`)
3. Checks workspace quota and rate limits
4. Handlers access principal via `s.GetPrincipal()`

Protected vs public routes are determined by middleware stack in `register.go`:
- `protectedMiddlewares`: includes `withAuthentication`
- `publicMiddlewares`: no auth (used for liveness, portal exchange, OpenAPI spec)

## Adding a New Resolver

1. Create a subpackage under `pkg/auth/` (e.g., `pkg/auth/my_source/`)
2. Implement the `Resolver` interface
3. Return `(nil, nil)` when the credential type isn't present
4. Use `fault` package for errors with proper codes (`codes.Auth.Authentication.*` or `codes.Auth.Authorization.*`)
5. Wire into the chain in `svc/api/run.go`

## Adding New Resource Permissions

1. Add the action constant to the appropriate file in `pkg/rbac/permissions/`
2. Add the `rbac.Tuple` check in the handler
3. If it should be available via WorkOS, add the slug mapping to `pkg/auth/workos/permissions.go`
4. Run `tools/upsert-workos-permissions` to sync to WorkOS
5. Update the dashboard permissions UI in `web/apps/dashboard/.../permissions.ts`
