# API Route Handler Guide

All public API endpoints live in `svc/api/routes/`. Each endpoint is a self-contained package following strict conventions for structure, testing, and authorization.

## Package Naming

```
svc/api/routes/v2_{domain}_{action}/
```

Examples:
- `v2_projects_create_project/`
- `v2_keys_verify_key/`
- `v2_ratelimit_set_override/`

## Handler Structure

Every handler package contains:

```
v2_projects_create_project/
├── handler.go          # Route handler implementation
├── 200_test.go         # Success case integration tests
├── 400_test.go         # Validation error tests
├── 401_test.go         # Authentication error tests
├── 403_test.go         # Authorization error tests
├── 404_test.go         # Not found tests
├── 409_test.go         # Conflict tests (if applicable)
├── 412_test.go         # Precondition failed tests (if applicable)
└── BUILD.bazel         # Bazel build file
```

Test files are named by HTTP status code. Each file tests scenarios producing that status.

## handler.go Template

```go
package handler

import (
    "context"
    "net/http"

    "github.com/unkeyed/unkey/pkg/codes"
    "github.com/unkeyed/unkey/pkg/fault"
    "github.com/unkeyed/unkey/pkg/rbac"
    "github.com/unkeyed/unkey/pkg/zen"
    "github.com/unkeyed/unkey/svc/api/openapi"
)

// Type aliases for generated OpenAPI types. Tests and other consumers
// reference these instead of the full generated path.
type (
    Request  = openapi.V2ProjectsCreateProjectRequestBody
    Response = openapi.V2ProjectsCreateProjectResponseBody
)

// Handler holds dependencies injected during route registration.
type Handler struct {
    DB         db.Database
    Auditlogs  auditlogs.AuditLogService
    // ... only what this handler needs
}

func (h *Handler) Method() string { return "POST" }
func (h *Handler) Path() string   { return "/v2/projects.createProject" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
    // 1. Get authenticated principal
    principal, err := s.GetPrincipal()
    if err != nil {
        return err
    }

    // 2. Bind and validate request body
    req, err := zen.BindBody[Request](s)
    if err != nil {
        return err
    }

    // 3. Authorize with resource permissions
    err = principal.Authorize(rbac.T(rbac.Tuple{
        ResourceType: rbac.Project,
        ResourceID:   "*",
        Action:       rbac.CreateProject,
    }))
    if err != nil {
        return err
    }

    // 4. Business logic (DB queries, service calls)
    // Always scope queries to principal.WorkspaceID
    // ...

    // 5. Return response with meta
    return s.JSON(http.StatusOK, Response{
        Meta: openapi.Meta{
            RequestId: s.RequestID(),
        },
        Data: ...,
    })
}
```

## Key Conventions

### 1. Type Aliases at Top

Always define `Request` and `Response` type aliases referencing the generated OpenAPI types:

```go
type (
    Request  = openapi.V2ProjectsCreateProjectRequestBody
    Response = openapi.V2ProjectsCreateProjectResponseBody
)
```

### 2. Dependency Injection via Struct Fields

Handlers receive only the dependencies they need. No global state.

```go
type Handler struct {
    DB        db.Database
    Auditlogs auditlogs.AuditLogService
    KeyCache  caches.VerificationKeyByHash
}
```

### 3. Error Handling with fault

Use `fault` for all errors with proper codes:

```go
return fault.New("project not found",
    fault.Code(codes.Data.Project.NotFound.URN()),
    fault.Internal("project_id=%s workspace_id=%s", projectID, wsID),
    fault.Public("The project does not exist or you don't have access."),
)
```

- `fault.Code()`: Maps to HTTP status via the error handling middleware
- `fault.Internal()`: Logged server-side, never exposed to client
- `fault.Public()`: Returned in the API response body

### 4. Workspace Scoping

Every DB query must include `principal.WorkspaceID`:

```go
project, err := h.DB.FindProjectByWorkspaceAndID(ctx, db.FindProjectByWorkspaceAndIDParams{
    WorkspaceID: principal.WorkspaceID,
    ProjectID:   req.ProjectId,
})
```

### 5. Response Envelope

All responses use the standard envelope:

```go
return s.JSON(http.StatusOK, Response{
    Meta: openapi.Meta{
        RequestId: s.RequestID(),
    },
    Data: ...,
})
```

## Route Registration

Routes are wired in `svc/api/routes/register.go`:

```go
srv.RegisterRoute(
    protectedMiddlewares,
    &v2ProjectsCreateProject.Handler{
        CtrlClient: svc.CtrlProjectClient,
    },
)
```

Two middleware stacks:
- `protectedMiddlewares`: Auth required (panic recovery, observability, metrics, logging, error handling, timeout, validation, authentication)
- `publicMiddlewares`: No auth (liveness, portal exchange, OpenAPI spec)

## OpenAPI Spec

The API spec lives in `svc/api/openapi/`:

```
svc/api/openapi/
├── openapi-split.yaml              # Root file listing all endpoint refs
├── openapi-generated.yaml          # Merged output (generated, do not edit)
├── gen.go                          # Generated Go types
└── spec/paths/v2/{domain}/{action}/
    ├── index.yaml                  # Endpoint definition (method, path, responses)
    ├── V2{Domain}{Action}RequestBody.yaml
    └── V2{Domain}{Action}ResponseBody.yaml
```

When adding a new endpoint:
1. Create the spec files under `spec/paths/v2/`
2. Add the reference to `openapi-split.yaml`
3. Run `mise run generate` to regenerate `gen.go` and the merged YAML

## Integration Tests

Tests use the harness in `svc/api/internal/testutil/`:

```go
func Test200(t *testing.T) {
    t.Parallel()
    h := testutil.NewHarness(t)

    // Seed test data
    ws := h.Seed.Workspace(t)
    rootKey := h.Seed.RootKey(t, ws, rbac.CreateProject)

    // Make request
    resp := h.Post(t, "/v2/projects.createProject", handler.Request{
        Name: "my-project",
        Slug: "my-project",
    }, testutil.WithRootKey(rootKey))

    // Assert response
    require.Equal(t, http.StatusOK, resp.StatusCode)

    var body handler.Response
    testutil.DecodeJSON(t, resp, &body)
    require.NotEmpty(t, body.Data.Id)
}
```

Test conventions:
- Use `t.Parallel()` in all tests
- Use the test harness for HTTP calls (handles base URL, headers)
- Seed data explicitly per test (no shared fixtures)
- Test files named by status code they exercise
- Use `require` (not `assert`) for fail-fast behavior

## Services Struct

`svc/api/routes/services.go` defines the dependency injection container:

```go
type Services struct {
    Database             db.Database
    Keys                 keys.KeyService
    Auth                 auth.Authenticator
    Ratelimit            ratelimit.Service
    Auditlogs            auditlogs.AuditLogService
    Caches               caches.Caches
    Vault                vault.VaultServiceClient
    CtrlDeploymentClient ctrl.DeployServiceClient
    CtrlProjectClient    ctrl.ProjectServiceClient
    // ...
}
```

Constructed in `svc/api/run.go` and passed to `Register()`.

## Adding a New Endpoint

1. Create the OpenAPI spec in `svc/api/openapi/spec/paths/v2/{domain}/{action}/`
2. Add the ref to `openapi-split.yaml`
3. Run `mise run generate` (creates types in `gen.go`)
4. Create the handler package: `svc/api/routes/v2_{domain}_{action}/`
5. Implement `handler.go` with Method/Path/Handle
6. Add test files (200, 400, 401, 403, 404 at minimum)
7. Wire in `register.go` with appropriate middleware stack
8. Add handler deps to `Services` if needed (update `services.go`)
9. Run `mise run bazel` to sync BUILD files
10. Run `mise run test` to verify
