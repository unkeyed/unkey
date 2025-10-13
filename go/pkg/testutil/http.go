package testutil

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/go/internal/services/analytics"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/ratelimit"
	"github.com/unkeyed/unkey/go/internal/services/usagelimiter"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/counter"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/testutil/containers"
	"github.com/unkeyed/unkey/go/pkg/testutil/seed"
	"github.com/unkeyed/unkey/go/pkg/vault"
	masterKeys "github.com/unkeyed/unkey/go/pkg/vault/keys"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	"github.com/unkeyed/unkey/go/pkg/zen"
	"github.com/unkeyed/unkey/go/pkg/zen/validation"
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
	seeder                     *seed.Seeder
}

func NewHarness(t *testing.T) *Harness {
	clk := clock.NewTestClock()

	logger := logging.New()

	// Start all services in parallel first
	containers.StartAllServices(t)

	mysqlCfg := containers.MySQL(t)
	mysqlCfg.DBName = "unkey"
	mysqlDSN := mysqlCfg.FormatDSN()

	redisUrl := containers.Redis(t)

	db, err := db.New(db.Config{
		Logger:      logger,
		PrimaryDSN:  mysqlDSN,
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	caches, err := caches.New(caches.Config{
		Logger: logger,
		Clock:  clk,
	})
	require.NoError(t, err)

	srv, err := zen.New(zen.Config{
		Logger: logger,
		Flags: &zen.Flags{
			TestMode: true,
		},
		TLS: nil,
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
		BaseURL:       "clickhouse://localhost:9000/default?secure=false&skip_verify=true&dial_timeout=10s",
		Vault:         v,
	})
	require.NoError(t, err)

	// Create seeder
	seeder := seed.New(t, db, v)

	seeder.Seed(context.Background())

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
		Auditlogs: auditlogs.New(auditlogs.Config{
			DB:     db,
			Logger: logger,
		}),
		Caches: caches,
		middleware: []zen.Middleware{
			zen.WithTracing(),
			zen.WithLogging(logger),
			zen.WithErrorHandling(logger),
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

func (h *Harness) CreateIdentity(req seed.CreateIdentityRequest) string {
	return h.seeder.CreateIdentity(context.Background(), req)
}

func (h *Harness) CreateRatelimit(req seed.CreateRatelimitRequest) string {
	return h.seeder.CreateRatelimit(context.Background(), req)
}

func (h *Harness) CreateRole(req seed.CreateRoleRequest) string {
	return h.seeder.CreateRole(context.Background(), req)
}

func (h *Harness) CreatePermission(req seed.CreatePermissionRequest) string {
	return h.seeder.CreatePermission(context.Background(), req)
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
