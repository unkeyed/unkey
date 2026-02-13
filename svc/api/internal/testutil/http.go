package testutil

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/internal/services/analytics"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
	vaulttestutil "github.com/unkeyed/unkey/svc/vault/testutil"
)

// Harness provides a complete integration test environment with real dependencies.
// It manages Docker containers for MySQL, Redis, ClickHouse, and S3, seeds baseline
// test data, and exposes all services needed to test API endpoints.
//
// The exported fields provide direct access to services when tests need to verify
// side effects or set up complex scenarios beyond what the helper methods offer.
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
	Vault                      vault.Client
	AnalyticsConnectionManager analytics.ConnectionManager
	seeder                     *seed.Seeder
}

// NewHarness creates a fully initialized test harness with all dependencies started.
// Container startup is parallelized, and the database is seeded with baseline data
// including a root workspace and key space. The harness is tied to the test lifecycle
// and containers are cleaned up when the test completes.
func NewHarness(t *testing.T) *Harness {
	clk := clock.NewTestClock()

	// Start all services in parallel first
	containers.StartAllServices(t)

	mysqlCfg := containers.MySQL(t)
	mysqlDSN := mysqlCfg.FormatDSN()

	redisUrl := dockertest.Redis(t)

	db, err := db.New(db.Config{
		PrimaryDSN:  mysqlDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	caches, err := caches.New(caches.Config{
		CacheInvalidationTopic: nil,
		NodeID:                 "",
		Clock:                  clk,
	})
	require.NoError(t, err)

	srv, err := zen.New(zen.Config{
		MaxRequestBodySize: 0,
		Flags: &zen.Flags{
			TestMode: true,
		},
		TLS:          nil,
		EnableH2C:    false,
		ReadTimeout:  0,
		WriteTimeout: 0,
	})
	require.NoError(t, err)

	// Get ClickHouse connection string
	chDSN := containers.ClickHouse(t)

	// Create real ClickHouse client
	ch, err := clickhouse.New(clickhouse.Config{
		URL: chDSN,
	})
	require.NoError(t, err)

	validator, err := validation.New()
	require.NoError(t, err)

	ctr, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: redisUrl,
	})
	require.NoError(t, err)

	ratelimitService, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: ctr,
	})
	require.NoError(t, err)

	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		DB:      db,
		Counter: ctr,
		TTL:     60 * time.Second,
	})
	require.NoError(t, err)

	keyService, err := keys.New(keys.Config{
		DB:           db,
		KeyCache:     caches.VerificationKeyByHash,
		RateLimiter:  ratelimitService,
		RBAC:         rbac.New(),
		Clickhouse:   ch,
		Region:       "test",
		UsageLimiter: ulSvc,
	})
	require.NoError(t, err)

	testVault := vaulttestutil.StartTestVaultWithMemory(t)
	v := vault.NewConnectClient(testVault.Client)

	// Create analytics connection manager
	analyticsConnManager, err := analytics.NewConnectionManager(analytics.ConnectionManagerConfig{
		SettingsCache: caches.ClickhouseSetting,
		Database:      db,
		Clock:         clk,
		BaseURL:       chDSN,
		Vault:         v,
	})
	require.NoError(t, err)

	// Create seeder
	seeder := seed.New(t, db, v)

	seeder.Seed(context.Background())

	audit, err := auditlogs.New(auditlogs.Config{
		DB: db,
	})
	require.NoError(t, err)

	h := Harness{
		t:                          t,
		srv:                        srv,
		validator:                  validator,
		Keys:                       keyService,
		UsageLimiter:               ulSvc,
		Ratelimit:                  ratelimitService,
		Vault:                      v,
		ClickHouse:                 ch,
		DB:                         db,
		seeder:                     seeder,
		Clock:                      clk,
		AnalyticsConnectionManager: analyticsConnManager,
		Auditlogs:                  audit,
		Caches:                     caches,
		middleware: []zen.Middleware{
			zen.WithObservability(),
			zen.WithLogging(),
			middleware.WithErrorHandling(),
			zen.WithValidation(validator),
		},
	}

	return &h
}

// Register adds a route to the test server with the standard middleware stack.
// Pass custom middleware to override the defaults, which include observability,
// logging, error handling, and validation. Passing no middleware uses the defaults.
func (h *Harness) Register(route zen.Route, middleware ...zen.Middleware) {
	if len(middleware) == 0 {
		middleware = h.middleware
	}

	h.srv.RegisterRoute(
		middleware,
		route,
	)
}

// CreateRootKey creates a root key that authorizes operations on the given workspace.
// The returned string is the raw key value for use in Authorization headers. Pass
// permission names to restrict what the key can do; omitting permissions grants no
// permissions (the key can authenticate but not authorize any operations).
func (h *Harness) CreateRootKey(workspaceID string, permissions ...string) string {
	return h.seeder.CreateRootKey(context.Background(), workspaceID, permissions...)
}

// CreateWorkspace creates a new workspace with auto-generated IDs and names.
func (h *Harness) CreateWorkspace() db.Workspace {
	return h.seeder.CreateWorkspace(context.Background())
}

// CreateApi creates an API with the specified configuration. The API's key space
// is created automatically. See [seed.CreateApiRequest] for available options.
func (h *Harness) CreateApi(req seed.CreateApiRequest) db.Api {
	return h.seeder.CreateAPI(context.Background(), req)
}

// CreateKey creates a key in the specified key space with optional permissions,
// roles, rate limits, and other configuration. Returns both the key ID (for database
// lookups) and the raw key value (for authentication). See [seed.CreateKeyRequest].
func (h *Harness) CreateKey(req seed.CreateKeyRequest) seed.CreateKeyResponse {
	return h.seeder.CreateKey(context.Background(), req)
}

// CreateIdentity creates an identity with optional rate limits attached.
func (h *Harness) CreateIdentity(req seed.CreateIdentityRequest) db.Identity {
	return h.seeder.CreateIdentity(context.Background(), req)
}

// CreateRatelimit creates a rate limit configuration attached to either a key or identity.
func (h *Harness) CreateRatelimit(req seed.CreateRatelimitRequest) db.Ratelimit {
	return h.seeder.CreateRatelimit(context.Background(), req)
}

// CreateRole creates a role with optional permissions attached.
func (h *Harness) CreateRole(req seed.CreateRoleRequest) db.Role {
	return h.seeder.CreateRole(context.Background(), req)
}

// CreatePermission creates a permission that can be attached to keys or roles.
func (h *Harness) CreatePermission(req seed.CreatePermissionRequest) db.Permission {
	return h.seeder.CreatePermission(context.Background(), req)
}

// CreateProject creates a project within a workspace.
func (h *Harness) CreateProject(req seed.CreateProjectRequest) db.Project {
	return h.seeder.CreateProject(context.Background(), req)
}

// CreateEnvironment creates an environment within a project.
func (h *Harness) CreateEnvironment(req seed.CreateEnvironmentRequest) db.Environment {
	return h.seeder.CreateEnvironment(h.t.Context(), req)
}

// CreateDeployment creates a deployment within a project and environment.
func (h *Harness) CreateDeployment(req seed.CreateDeploymentRequest) db.Deployment {
	return h.seeder.CreateDeployment(context.Background(), req)
}

// DeploymentTestSetup contains all resources needed for deployment tests.
type DeploymentTestSetup struct {
	Workspace   db.Workspace
	RootKey     string
	Project     db.Project
	Environment db.Environment
}

// CreateTestDeploymentSetupOptions configures the resources created by
// [Harness.CreateTestDeploymentSetup].
type CreateTestDeploymentSetupOptions struct {
	ProjectName     string
	ProjectSlug     string
	EnvironmentSlug string
	SkipEnvironment bool
	Permissions     []string
}

// CreateTestDeploymentSetup creates a complete deployment test environment with a
// workspace, root key, project, and environment. This is a convenience method for
// tests that need all these resources together. Pass [CreateTestDeploymentSetupOptions]
// to customize names, slugs, or skip environment creation. Defaults to project name
// "test-project", slugs "production", and full permissions unless specified.
func (h *Harness) CreateTestDeploymentSetup(opts ...CreateTestDeploymentSetupOptions) DeploymentTestSetup {
	h.t.Helper()

	config := CreateTestDeploymentSetupOptions{
		ProjectName:     "test-project",
		ProjectSlug:     "production",
		EnvironmentSlug: "production",
		SkipEnvironment: false,
		Permissions:     nil,
	}

	if len(opts) > 0 {
		if opts[0].ProjectName != "" {
			config.ProjectName = opts[0].ProjectName
		}
		if opts[0].ProjectSlug != "" {
			config.ProjectSlug = opts[0].ProjectSlug
		}
		if opts[0].EnvironmentSlug != "" {
			config.EnvironmentSlug = opts[0].EnvironmentSlug
		}
		config.SkipEnvironment = opts[0].SkipEnvironment
		if opts[0].Permissions != nil {
			config.Permissions = opts[0].Permissions
		}
	}

	workspace := h.CreateWorkspace()

	var rootKey string
	if config.Permissions != nil {
		rootKey = h.CreateRootKey(workspace.ID, config.Permissions...)
	} else {
		rootKey = h.CreateRootKey(workspace.ID)
	}

	project := h.CreateProject(seed.CreateProjectRequest{
		WorkspaceID:      workspace.ID,
		Name:             config.ProjectName,
		ID:               uid.New(uid.ProjectPrefix),
		Slug:             config.ProjectSlug,
		DefaultBranch:    "",
		DeleteProtection: false,
	})

	var environment db.Environment
	if !config.SkipEnvironment {
		environment = h.CreateEnvironment(seed.CreateEnvironmentRequest{
			ID:               uid.New(uid.EnvironmentPrefix),
			WorkspaceID:      workspace.ID,
			ProjectID:        project.ID,
			Slug:             config.EnvironmentSlug,
			Description:      config.EnvironmentSlug + " environment",
			DeleteProtection: false,
			SentinelConfig:   nil,
		})
	}

	return DeploymentTestSetup{
		Workspace:   workspace,
		RootKey:     rootKey,
		Project:     project,
		Environment: environment,
	}
}

// SetupAnalyticsOption configures analytics settings for [Harness.SetupAnalytics].
type SetupAnalyticsOption func(*setupAnalyticsConfig)

type setupAnalyticsConfig struct {
	MaxQueryResultRows        int32
	MaxQueryMemoryBytes       int64
	MaxQueriesPerWindow       int32
	MaxExecutionTimePerWindow int32
	QuotaDurationSeconds      int32
	MaxQueryExecutionTime     int32
	RetentionDays             int32
}

// WithMaxQueryResultRows sets the maximum number of rows a query can return.
// Default is 10,000,000.
func WithMaxQueryResultRows(rows int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxQueryResultRows = rows
	}
}

// WithMaxQueryMemoryBytes sets the maximum memory a query can use.
// Default is 1,000,000,000 (1GB).
func WithMaxQueryMemoryBytes(bytes int64) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxQueryMemoryBytes = bytes
	}
}

// WithMaxQueriesPerWindow sets the maximum queries allowed per quota window.
// Default is 1,000.
func WithMaxQueriesPerWindow(queries int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxQueriesPerWindow = queries
	}
}

// WithMaxExecutionTimePerWindow sets the maximum total execution time per quota window.
// Default is 1,800 seconds (30 minutes).
func WithMaxExecutionTimePerWindow(seconds int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxExecutionTimePerWindow = seconds
	}
}

// WithRetentionDays sets how long analytics data is retained. Default is 30 days.
func WithRetentionDays(days int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.RetentionDays = days
	}
}

// SetupAnalytics configures a ClickHouse user and analytics settings for a workspace.
// This creates the user in ClickHouse, encrypts the password in the vault, and stores
// the settings in the database. Use the With* options to customize query limits and
// retention. Tests that query analytics data must call this before making requests.
func (h *Harness) SetupAnalytics(workspaceID string, opts ...SetupAnalyticsOption) {
	ctx := context.Background()

	// Defaults
	config := setupAnalyticsConfig{
		MaxQueryResultRows:        10_000_000,
		MaxQueryMemoryBytes:       1_000_000_000,
		MaxQueriesPerWindow:       1_000,
		MaxExecutionTimePerWindow: 1_800,
		QuotaDurationSeconds:      3_600,
		MaxQueryExecutionTime:     30,
		RetentionDays:             30, // Default 30-day retention
	}

	// Apply options
	for _, opt := range opts {
		opt(&config)
	}

	password := "test_password"
	username := workspaceID

	// Encrypt the password using the vault service
	encryptRes, err := h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
		Keyring: workspaceID,
		Data:    password,
	})
	require.NoError(h.t, err)

	// Ensure quota exists with retention days
	err = db.Query.UpsertQuota(ctx, h.DB.RW(), db.UpsertQuotaParams{
		WorkspaceID:            workspaceID,
		LogsRetentionDays:      config.RetentionDays,
		AuditLogsRetentionDays: config.RetentionDays,
		RequestsPerMonth:       1_000_000,
		Team:                   false,
	})
	require.NoError(h.t, err)

	// Configure ClickHouse user with permissions, quotas, and settings
	err = h.ClickHouse.ConfigureUser(ctx, clickhouse.UserConfig{
		WorkspaceID:               workspaceID,
		Username:                  username,
		Password:                  password,
		AllowedTables:             clickhouse.DefaultAllowedTables(),
		QuotaDurationSeconds:      config.QuotaDurationSeconds,
		MaxQueriesPerWindow:       config.MaxQueriesPerWindow,
		MaxExecutionTimePerWindow: config.MaxExecutionTimePerWindow,
		MaxQueryExecutionTime:     config.MaxQueryExecutionTime,
		MaxQueryMemoryBytes:       config.MaxQueryMemoryBytes,
		MaxQueryResultRows:        config.MaxQueryResultRows,
		RetentionDays:             config.RetentionDays,
	})
	require.NoError(h.t, err)

	// Store the encrypted credentials in the database
	now := h.Clock.Now().UnixMilli()
	err = db.Query.InsertClickhouseWorkspaceSettings(ctx, h.DB.RW(), db.InsertClickhouseWorkspaceSettingsParams{
		WorkspaceID:               workspaceID,
		Username:                  username,
		PasswordEncrypted:         encryptRes.GetEncrypted(),
		QuotaDurationSeconds:      config.QuotaDurationSeconds,
		MaxQueriesPerWindow:       config.MaxQueriesPerWindow,
		MaxExecutionTimePerWindow: config.MaxExecutionTimePerWindow,
		MaxQueryExecutionTime:     config.MaxQueryExecutionTime,
		MaxQueryMemoryBytes:       config.MaxQueryMemoryBytes,
		MaxQueryResultRows:        config.MaxQueryResultRows,
		CreatedAt:                 now,
		UpdatedAt:                 sql.NullInt64{Valid: true, Int64: now},
	})
	require.NoError(h.t, err)
}

// Resources returns the baseline seed data created during harness initialization.
// This includes the root workspace, root key space, and user workspace that exist
// before any test-specific data is created.
func (h *Harness) Resources() seed.Resources {
	return h.seeder.Resources
}

// TestResponse wraps an HTTP response with typed body parsing for test assertions.
type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    *TBody
	RawBody string
}

// CallRaw executes an HTTP request against the test server and returns the parsed
// response. Use this when you need full control over the request, such as setting
// path parameters or custom headers. The response body is JSON-unmarshaled into the
// type parameter.
func CallRaw[Res any](h *Harness, req *http.Request) TestResponse[Res] {
	rr := httptest.NewRecorder()

	h.srv.Mux().ServeHTTP(rr, req)
	rawBody := rr.Body.Bytes()

	res := TestResponse[Res]{
		Status:  rr.Code,
		Headers: rr.Header(),
		RawBody: string(rawBody),
		Body:    nil,
	}

	var responseBody Res
	err := json.Unmarshal(rawBody, &responseBody)
	require.NoError(h.t, err)

	res.Body = &responseBody

	return res
}

// CallRoute executes a request against a registered route and returns the typed
// response. The request body is JSON-encoded from req, and the response is unmarshaled
// into Res. This is the primary way to test API endpoints. Pass nil headers to use
// an empty header set.
func CallRoute[Req any, Res any](h *Harness, route zen.Route, headers http.Header, req Req) TestResponse[Res] {
	h.t.Helper()

	rr := httptest.NewRecorder()

	body := new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(req)
	require.NoError(h.t, err)

	httpReq := httptest.NewRequest(route.Method(), route.Path(), body)
	httpReq.Header = headers
	if httpReq.Header == nil {
		httpReq.Header = http.Header{}
	}

	h.srv.Mux().ServeHTTP(rr, httpReq)
	require.NoError(h.t, err)

	rawBody := rr.Body.Bytes()

	res := TestResponse[Res]{
		Status:  rr.Code,
		Headers: rr.Header(),
		RawBody: string(rawBody),
		Body:    nil,
	}

	var responseBody Res
	err = json.Unmarshal(rawBody, &responseBody)
	require.NoError(h.t, err)

	res.Body = &responseBody

	return res
}

// UnmarshalBody decodes a JSON response body into the provided pointer. This is
// useful when working directly with httptest.ResponseRecorder rather than using
// [CallRoute] or [CallRaw].
func UnmarshalBody[Body any](t *testing.T, r *httptest.ResponseRecorder, body *Body) {
	err := json.Unmarshal(r.Body.Bytes(), &body)
	require.NoError(t, err)
}
