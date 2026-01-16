package testutil

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
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
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rpc/interceptor"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
	masterKeys "github.com/unkeyed/unkey/pkg/vault/keys"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/pkg/zen/validation"
	"github.com/unkeyed/unkey/svc/api/internal/middleware"
	"github.com/unkeyed/unkey/svc/api/internal/testutil/seed"
)

type Harness struct {
	t *testing.T

	Clock *clock.TestClock

	srv       *zen.Server
	validator *validation.Validator

	middleware []zen.Middleware

	DB                         db.Database
	Caches                     caches.Caches
	Logger                     logging.Logger
	Keys                       keys.KeyService
	UsageLimiter               usagelimiter.Service
	Auditlogs                  auditlogs.AuditLogService
	ClickHouse                 clickhouse.ClickHouse
	Ratelimit                  ratelimit.Service
	Vault                      *vault.Service
	AnalyticsConnectionManager analytics.ConnectionManager
	CtrlDeploymentClient       ctrlv1connect.DeploymentServiceClient
	CtrlBuildClient            ctrlv1connect.BuildServiceClient
	seeder                     *seed.Seeder
}

func NewHarness(t *testing.T) *Harness {
	clk := clock.NewTestClock()
	logger := logging.New()

	// Start all services in parallel first
	containers.StartAllServices(t)

	mysqlCfg := containers.MySQL(t)
	mysqlDSN := mysqlCfg.FormatDSN()

	redisUrl := dockertest.Redis(t)

	db, err := db.New(db.Config{
		Logger:      logger,
		PrimaryDSN:  mysqlDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	caches, err := caches.New(caches.Config{
		CacheInvalidationTopic: nil,
		NodeID:                 "",
		Logger:                 logger,
		Clock:                  clk,
	})
	require.NoError(t, err)

	srv, err := zen.New(zen.Config{
		MaxRequestBodySize: 0,
		Logger:             logger,
		Flags: &zen.Flags{
			TestMode: true,
		},
		TLS:          nil,
		ReadTimeout:  0,
		WriteTimeout: 0,
	})
	require.NoError(t, err)

	// Get ClickHouse connection string
	chDSN := containers.ClickHouse(t)

	// Create real ClickHouse client
	ch, err := clickhouse.New(clickhouse.Config{
		URL:    chDSN,
		Logger: logger,
	})
	require.NoError(t, err)

	validator, err := validation.New()
	require.NoError(t, err)

	ctr, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: redisUrl,
		Logger:   logger,
	})
	require.NoError(t, err)

	ratelimitService, err := ratelimit.New(ratelimit.Config{
		Logger:  logger,
		Clock:   clk,
		Counter: ctr,
	})
	require.NoError(t, err)

	ulSvc, err := usagelimiter.NewRedisWithCounter(usagelimiter.RedisConfig{
		Logger:  logger,
		DB:      db,
		Counter: ctr,
		TTL:     60 * time.Second,
	})
	require.NoError(t, err)

	keyService, err := keys.New(keys.Config{
		Logger:       logger,
		DB:           db,
		KeyCache:     caches.VerificationKeyByHash,
		RateLimiter:  ratelimitService,
		RBAC:         rbac.New(),
		Clickhouse:   ch,
		Region:       "test",
		UsageLimiter: ulSvc,
	})
	require.NoError(t, err)

	s3 := containers.S3(t)

	vaultStorage, err := storage.NewS3(storage.S3Config{
		S3URL:             s3.HostURL,
		S3Bucket:          "test",
		S3AccessKeyID:     s3.AccessKeyID,
		S3AccessKeySecret: s3.AccessKeySecret,
		Logger:            logger,
	})
	require.NoError(t, err)

	_, masterKey, err := masterKeys.GenerateMasterKey()
	require.NoError(t, err)
	v, err := vault.New(vault.Config{
		Logger:     logger,
		Storage:    vaultStorage,
		MasterKeys: []string{masterKey},
	})
	require.NoError(t, err)

	// Create analytics connection manager
	analyticsConnManager, err := analytics.NewConnectionManager(analytics.ConnectionManagerConfig{
		SettingsCache: caches.ClickhouseSetting,
		Database:      db,
		Logger:        logger,
		Clock:         clk,
		BaseURL:       chDSN,
		Vault:         v,
	})
	require.NoError(t, err)

	// Create seeder
	seeder := seed.New(t, db, v)

	seeder.Seed(context.Background())

	// Get CTRL service URL and token
	ctrlURL, ctrlToken := containers.ControlPlane(t)

	// Create CTRL clients
	ctrlDeploymentClient := ctrlv1connect.NewDeploymentServiceClient(
		http.DefaultClient,
		ctrlURL,
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", ctrlToken),
		})),
	)

	ctrlBuildClient := ctrlv1connect.NewBuildServiceClient(
		http.DefaultClient,
		ctrlURL,
		connect.WithInterceptors(interceptor.NewHeaderInjector(map[string]string{
			"Authorization": fmt.Sprintf("Bearer %s", ctrlToken),
		})),
	)

	audit, err := auditlogs.New(auditlogs.Config{
		DB:     db,
		Logger: logger,
	})
	require.NoError(t, err)

	h := Harness{
		t:                          t,
		Logger:                     logger,
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
		CtrlDeploymentClient:       ctrlDeploymentClient,
		CtrlBuildClient:            ctrlBuildClient,
		Auditlogs:                  audit,
		Caches:                     caches,
		middleware: []zen.Middleware{
			zen.WithObservability(),
			zen.WithLogging(logger),
			middleware.WithErrorHandling(logger),
			zen.WithValidation(validator),
		},
	}

	return &h
}

// Register registers a route with the harness.
// You can override the middleware by passing a list of middleware.
func (h *Harness) Register(route zen.Route, middleware ...zen.Middleware) {
	if len(middleware) == 0 {
		middleware = h.middleware
	}

	h.srv.RegisterRoute(
		middleware,
		route,
	)
}

// CreateRootKey creates a root key with the specified permissions
func (h *Harness) CreateRootKey(workspaceID string, permissions ...string) string {
	return h.seeder.CreateRootKey(context.Background(), workspaceID, permissions...)
}

func (h *Harness) CreateWorkspace() db.Workspace {
	return h.seeder.CreateWorkspace(context.Background())
}

func (h *Harness) CreateApi(req seed.CreateApiRequest) db.Api {
	return h.seeder.CreateAPI(context.Background(), req)
}

func (h *Harness) CreateKey(req seed.CreateKeyRequest) seed.CreateKeyResponse {
	return h.seeder.CreateKey(context.Background(), req)
}

func (h *Harness) CreateIdentity(req seed.CreateIdentityRequest) db.Identity {
	return h.seeder.CreateIdentity(context.Background(), req)
}

func (h *Harness) CreateRatelimit(req seed.CreateRatelimitRequest) db.Ratelimit {
	return h.seeder.CreateRatelimit(context.Background(), req)
}

func (h *Harness) CreateRole(req seed.CreateRoleRequest) db.Role {
	return h.seeder.CreateRole(context.Background(), req)
}

func (h *Harness) CreatePermission(req seed.CreatePermissionRequest) db.Permission {
	return h.seeder.CreatePermission(context.Background(), req)
}

func (h *Harness) CreateProject(req seed.CreateProjectRequest) db.Project {
	return h.seeder.CreateProject(context.Background(), req)
}

func (h *Harness) CreateEnvironment(req seed.CreateEnvironmentRequest) db.Environment {
	return h.seeder.CreateEnvironment(h.t.Context(), req)
}

// DeploymentTestSetup contains all resources needed for deployment tests
type DeploymentTestSetup struct {
	Workspace   db.Workspace
	RootKey     string
	Project     db.Project
	Environment db.Environment
}

// CreateTestDeploymentSetupOptions allows customization of the test setup
type CreateTestDeploymentSetupOptions struct {
	ProjectName     string
	ProjectSlug     string
	EnvironmentSlug string
	SkipEnvironment bool
}

// CreateTestDeploymentSetup creates workspace, root key, project, and environment with sensible defaults
func (h *Harness) CreateTestDeploymentSetup(opts ...CreateTestDeploymentSetupOptions) DeploymentTestSetup {
	h.t.Helper()

	config := CreateTestDeploymentSetupOptions{
		ProjectName:     "test-project",
		ProjectSlug:     "production",
		EnvironmentSlug: "production",
		SkipEnvironment: false,
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
	}

	workspace := h.CreateWorkspace()
	rootKey := h.CreateRootKey(workspace.ID)

	project := h.CreateProject(seed.CreateProjectRequest{
		WorkspaceID:      workspace.ID,
		Name:             config.ProjectName,
		ID:               uid.New(uid.ProjectPrefix),
		Slug:             config.ProjectSlug,
		GitRepositoryURL: "",
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

func WithMaxQueryResultRows(rows int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxQueryResultRows = rows
	}
}

func WithMaxQueryMemoryBytes(bytes int64) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxQueryMemoryBytes = bytes
	}
}

func WithMaxQueriesPerWindow(queries int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxQueriesPerWindow = queries
	}
}

func WithMaxExecutionTimePerWindow(seconds int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.MaxExecutionTimePerWindow = seconds
	}
}

func WithRetentionDays(days int32) SetupAnalyticsOption {
	return func(c *setupAnalyticsConfig) {
		c.RetentionDays = days
	}
}

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

func (h *Harness) Resources() seed.Resources {
	return h.seeder.Resources
}

type TestResponse[TBody any] struct {
	Status  int
	Headers http.Header
	Body    *TBody
	RawBody string
}

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

// UnmarshalBody is a helper function to unmarshal the response body
func UnmarshalBody[Body any](t *testing.T, r *httptest.ResponseRecorder, body *Body) {
	err := json.Unmarshal(r.Body.Bytes(), &body)
	require.NoError(t, err)
}
