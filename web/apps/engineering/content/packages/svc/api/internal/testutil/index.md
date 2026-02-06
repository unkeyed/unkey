---
title: testutil
description: "provides integration test infrastructure for the API service"
---

Package testutil provides integration test infrastructure for the API service.

This package creates a complete, isolated test environment with real dependencies (MySQL, Redis, ClickHouse, S3, control plane) running in Docker containers. Tests using this package verify end-to-end behavior rather than mocking service boundaries.

### Key Types

The main entry point is \[Harness], which orchestrates container startup, database seeding, and provides access to all services. Use \[NewHarness] to create one. \[TestResponse] wraps HTTP responses with typed body parsing for assertions.

### Usage

Create a harness at the start of your test. The harness handles container lifecycle and provides methods to create test data and make HTTP requests:

	func TestMyEndpoint(t *testing.T) {
	    h := testutil.NewHarness(t)
	    h.Register(myRoute)

	    ws := h.CreateWorkspace()
	    rootKey := h.CreateRootKey(ws.ID, "api.keys.create")

	    resp := testutil.CallRoute[RequestType, ResponseType](h, myRoute, headers, req)
	    require.Equal(t, 200, resp.Status)
	}

For deployment-related tests, use \[Harness.CreateTestDeploymentSetup] to create a workspace, project, environment, and root key in one call.

### Container Dependencies

The harness starts MySQL, Redis, ClickHouse, and MinIO (S3-compatible) containers. These are shared across tests within a package for speed, but each test gets fresh database state through the seeder. Container startup is parallelized to minimize test latency.

## Variables

```go
var _ ctrlv1connect.DeploymentServiceClient = (*MockDeploymentClient)(nil)
```


## Functions

### func UnmarshalBody

```go
func UnmarshalBody[Body any](t *testing.T, r *httptest.ResponseRecorder, body *Body)
```

UnmarshalBody decodes a JSON response body into the provided pointer. This is useful when working directly with httptest.ResponseRecorder rather than using \[CallRoute] or \[CallRaw].


## Types

### type CreateTestDeploymentSetupOptions

```go
type CreateTestDeploymentSetupOptions struct {
	ProjectName     string
	ProjectSlug     string
	EnvironmentSlug string
	SkipEnvironment bool
	Permissions     []string
}
```

CreateTestDeploymentSetupOptions configures the resources created by \[Harness.CreateTestDeploymentSetup].

### type DeploymentTestSetup

```go
type DeploymentTestSetup struct {
	Workspace   db.Workspace
	RootKey     string
	Project     db.Project
	Environment db.Environment
}
```

DeploymentTestSetup contains all resources needed for deployment tests.

### type Harness

```go
type Harness struct {
	t *testing.T

	// Clock is a controllable clock for time-dependent tests. Advancing the clock
	// affects rate limiting windows, token expiration, and other time-based behavior.
	Clock *clock.TestClock

	srv       *zen.Server
	validator *validation.Validator

	middleware []zen.Middleware

	// DB provides direct database access for verifying side effects or setting up
	// test data that the seeder methods don't cover.
	DB                         db.Database
	Caches                     caches.Caches
	Keys                       keys.KeyService
	UsageLimiter               usagelimiter.Service
	Auditlogs                  auditlogs.AuditLogService
	ClickHouse                 clickhouse.ClickHouse
	Ratelimit                  ratelimit.Service
	Vault                      *vault.Service
	AnalyticsConnectionManager analytics.ConnectionManager
	seeder                     *seed.Seeder
}
```

Harness provides a complete integration test environment with real dependencies. It manages Docker containers for MySQL, Redis, ClickHouse, and S3, seeds baseline test data, and exposes all services needed to test API endpoints.

The exported fields provide direct access to services when tests need to verify side effects or set up complex scenarios beyond what the helper methods offer.

#### func NewHarness

```go
func NewHarness(t *testing.T) *Harness
```

NewHarness creates a fully initialized test harness with all dependencies started. Container startup is parallelized, and the database is seeded with baseline data including a root workspace and key space. The harness is tied to the test lifecycle and containers are cleaned up when the test completes.

#### func (Harness) CreateApi

```go
func (h *Harness) CreateApi(req seed.CreateApiRequest) db.Api
```

CreateApi creates an API with the specified configuration. The API's key space is created automatically. See \[seed.CreateApiRequest] for available options.

#### func (Harness) CreateDeployment

```go
func (h *Harness) CreateDeployment(req seed.CreateDeploymentRequest) db.Deployment
```

CreateDeployment creates a deployment within a project and environment.

#### func (Harness) CreateEnvironment

```go
func (h *Harness) CreateEnvironment(req seed.CreateEnvironmentRequest) db.Environment
```

CreateEnvironment creates an environment within a project.

#### func (Harness) CreateIdentity

```go
func (h *Harness) CreateIdentity(req seed.CreateIdentityRequest) db.Identity
```

CreateIdentity creates an identity with optional rate limits attached.

#### func (Harness) CreateKey

```go
func (h *Harness) CreateKey(req seed.CreateKeyRequest) seed.CreateKeyResponse
```

CreateKey creates a key in the specified key space with optional permissions, roles, rate limits, and other configuration. Returns both the key ID (for database lookups) and the raw key value (for authentication). See \[seed.CreateKeyRequest].

#### func (Harness) CreatePermission

```go
func (h *Harness) CreatePermission(req seed.CreatePermissionRequest) db.Permission
```

CreatePermission creates a permission that can be attached to keys or roles.

#### func (Harness) CreateProject

```go
func (h *Harness) CreateProject(req seed.CreateProjectRequest) db.Project
```

CreateProject creates a project within a workspace.

#### func (Harness) CreateRatelimit

```go
func (h *Harness) CreateRatelimit(req seed.CreateRatelimitRequest) db.Ratelimit
```

CreateRatelimit creates a rate limit configuration attached to either a key or identity.

#### func (Harness) CreateRole

```go
func (h *Harness) CreateRole(req seed.CreateRoleRequest) db.Role
```

CreateRole creates a role with optional permissions attached.

#### func (Harness) CreateRootKey

```go
func (h *Harness) CreateRootKey(workspaceID string, permissions ...string) string
```

CreateRootKey creates a root key that authorizes operations on the given workspace. The returned string is the raw key value for use in Authorization headers. Pass permission names to restrict what the key can do; omitting permissions grants no permissions (the key can authenticate but not authorize any operations).

#### func (Harness) CreateTestDeploymentSetup

```go
func (h *Harness) CreateTestDeploymentSetup(opts ...CreateTestDeploymentSetupOptions) DeploymentTestSetup
```

CreateTestDeploymentSetup creates a complete deployment test environment with a workspace, root key, project, and environment. This is a convenience method for tests that need all these resources together. Pass \[CreateTestDeploymentSetupOptions] to customize names, slugs, or skip environment creation. Defaults to project name "test-project", slugs "production", and full permissions unless specified.

#### func (Harness) CreateWorkspace

```go
func (h *Harness) CreateWorkspace() db.Workspace
```

CreateWorkspace creates a new workspace with auto-generated IDs and names.

#### func (Harness) Register

```go
func (h *Harness) Register(route zen.Route, middleware ...zen.Middleware)
```

Register adds a route to the test server with the standard middleware stack. Pass custom middleware to override the defaults, which include observability, logging, error handling, and validation. Passing no middleware uses the defaults.

#### func (Harness) Resources

```go
func (h *Harness) Resources() seed.Resources
```

Resources returns the baseline seed data created during harness initialization. This includes the root workspace, root key space, and user workspace that exist before any test-specific data is created.

#### func (Harness) SetupAnalytics

```go
func (h *Harness) SetupAnalytics(workspaceID string, opts ...SetupAnalyticsOption)
```

SetupAnalytics configures a ClickHouse user and analytics settings for a workspace. This creates the user in ClickHouse, encrypts the password in the vault, and stores the settings in the database. Use the With\* options to customize query limits and retention. Tests that query analytics data must call this before making requests.

### type MockDeploymentClient

```go
type MockDeploymentClient struct {
	mu                    sync.Mutex
	CreateDeploymentFunc  func(context.Context, *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error)
	GetDeploymentFunc     func(context.Context, *connect.Request[ctrlv1.GetDeploymentRequest]) (*connect.Response[ctrlv1.GetDeploymentResponse], error)
	RollbackFunc          func(context.Context, *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error)
	PromoteFunc           func(context.Context, *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error)
	CreateDeploymentCalls []*ctrlv1.CreateDeploymentRequest
	GetDeploymentCalls    []*ctrlv1.GetDeploymentRequest
	RollbackCalls         []*ctrlv1.RollbackRequest
	PromoteCalls          []*ctrlv1.PromoteRequest
}
```

MockDeploymentClient is a test double for the control plane's deployment service.

Each method has an optional function field that tests can set to customize behavior. If the function is nil, the method returns a sensible default. The mock also records calls so tests can verify the correct requests were made.

This mock is safe for concurrent use. All call recording is protected by a mutex.

#### func (MockDeploymentClient) CreateDeployment

```go
func (m *MockDeploymentClient) CreateDeployment(ctx context.Context, req *connect.Request[ctrlv1.CreateDeploymentRequest]) (*connect.Response[ctrlv1.CreateDeploymentResponse], error)
```

#### func (MockDeploymentClient) GetDeployment

```go
func (m *MockDeploymentClient) GetDeployment(ctx context.Context, req *connect.Request[ctrlv1.GetDeploymentRequest]) (*connect.Response[ctrlv1.GetDeploymentResponse], error)
```

#### func (MockDeploymentClient) Promote

```go
func (m *MockDeploymentClient) Promote(ctx context.Context, req *connect.Request[ctrlv1.PromoteRequest]) (*connect.Response[ctrlv1.PromoteResponse], error)
```

#### func (MockDeploymentClient) Rollback

```go
func (m *MockDeploymentClient) Rollback(ctx context.Context, req *connect.Request[ctrlv1.RollbackRequest]) (*connect.Response[ctrlv1.RollbackResponse], error)
```

### type SetupAnalyticsOption

```go
type SetupAnalyticsOption func(*setupAnalyticsConfig)
```

SetupAnalyticsOption configures analytics settings for \[Harness.SetupAnalytics].

#### func WithMaxExecutionTimePerWindow

```go
func WithMaxExecutionTimePerWindow(seconds int32) SetupAnalyticsOption
```

WithMaxExecutionTimePerWindow sets the maximum total execution time per quota window. Default is 1,800 seconds (30 minutes).

#### func WithMaxQueriesPerWindow

```go
func WithMaxQueriesPerWindow(queries int32) SetupAnalyticsOption
```

WithMaxQueriesPerWindow sets the maximum queries allowed per quota window. Default is 1,000.

#### func WithMaxQueryMemoryBytes

```go
func WithMaxQueryMemoryBytes(bytes int64) SetupAnalyticsOption
```

WithMaxQueryMemoryBytes sets the maximum memory a query can use. Default is 1,000,000,000 (1GB).

#### func WithMaxQueryResultRows

```go
func WithMaxQueryResultRows(rows int32) SetupAnalyticsOption
```

WithMaxQueryResultRows sets the maximum number of rows a query can return. Default is 10,000,000.

#### func WithRetentionDays

```go
func WithRetentionDays(days int32) SetupAnalyticsOption
```

WithRetentionDays sets how long analytics data is retained. Default is 30 days.

### type TestResponse

```go
type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    *TBody
	RawBody string
}
```

TestResponse wraps an HTTP response with typed body parsing for test assertions.

#### func CallRaw

```go
func CallRaw[Res any](h *Harness, req *http.Request) TestResponse[Res]
```

CallRaw executes an HTTP request against the test server and returns the parsed response. Use this when you need full control over the request, such as setting path parameters or custom headers. The response body is JSON-unmarshaled into the type parameter.

#### func CallRoute

```go
func CallRoute[Req any, Res any](h *Harness, route zen.Route, headers http.Header, req Req) TestResponse[Res]
```

CallRoute executes a request against a registered route and returns the typed response. The request body is JSON-encoded from req, and the response is unmarshaled into Res. This is the primary way to test API endpoints. Pass nil headers to use an empty header set.

