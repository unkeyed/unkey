package keys

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/counter"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/dockertest"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func setupWorkspaceRatelimitTest(t *testing.T) (*service, db.Database) {
	t.Helper()

	containers.StartAllServices(t)
	mysqlCfg := containers.MySQL(t)
	redisURL := dockertest.Redis(t)
	chDSN := containers.ClickHouse(t)

	database, err := db.New(db.Config{
		PrimaryDSN:  mysqlCfg.FormatDSN(),
		ReadOnlyDSN: "",
	})
	require.NoError(t, err)

	clk := clock.NewTestClock()

	c, err := caches.New(caches.Config{
		Broadcaster: nil,
		NodeID:      "",
		Clock:       clk,
	})
	require.NoError(t, err)

	ctr, err := counter.NewRedis(counter.RedisConfig{
		RedisURL: redisURL,
	})
	require.NoError(t, err)

	rl, err := ratelimit.New(ratelimit.Config{
		Clock:   clk,
		Counter: ctr,
	})
	require.NoError(t, err)

	ch, err := clickhouse.New(clickhouse.Config{
		URL: chDSN,
	})
	require.NoError(t, err)

	audit, err := auditlogs.New(auditlogs.Config{DB: database})
	require.NoError(t, err)

	namespaceSvc, err := namespace.New(namespace.Config{
		DB:        database,
		Cache:     c.RatelimitNamespace,
		Auditlogs: audit,
	})
	require.NoError(t, err)

	s := &service{
		db:                        database,
		quotaCache:                c.WorkspaceQuota,
		ratelimitNamespaceService: namespaceSvc,
		rateLimiter:               rl,
		clickhouse:                ch,
		region:                    "test",
	}

	return s, database
}

func createTestWorkspace(t *testing.T, database db.Database) string {
	t.Helper()
	wsID := uid.New("test_ws")
	err := db.Query.InsertWorkspace(context.Background(), database.RW(), db.InsertWorkspaceParams{
		ID:           wsID,
		OrgID:        uid.New("test_org"),
		Name:         uid.New("test"),
		Slug:         uid.New("slug"),
		CreatedAt:    time.Now().UnixMilli(),
		K8sNamespace: sql.NullString{Valid: true, String: uid.DNS1035()},
	})
	require.NoError(t, err)
	return wsID
}

func upsertQuota(t *testing.T, database db.Database, wsID string, limit, duration sql.NullInt32) {
	t.Helper()
	err := db.Query.UpsertQuota(context.Background(), database.RW(), db.UpsertQuotaParams{
		WorkspaceID:            wsID,
		RequestsPerMonth:       1_000_000,
		AuditLogsRetentionDays: 30,
		LogsRetentionDays:      30,
		Team:                   false,
		RatelimitLimit:         limit,
		RatelimitDuration:      duration,
	})
	require.NoError(t, err)
}

func testAuditContext() *namespace.AuditContext {
	return &namespace.AuditContext{
		ActorID:   uid.New("actor"),
		ActorName: "test-actor",
		ActorType: auditlog.RootKeyActor,
		RemoteIP:  "127.0.0.1",
		UserAgent: "test-agent",
	}
}

func TestCheckWorkspaceRateLimit_NullQuota_Unlimited(t *testing.T) {
	s, database := setupWorkspaceRatelimitTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)
	rootWsID := createTestWorkspace(t, database)

	// NULL limit and duration = unlimited
	upsertQuota(t, database, wsID, sql.NullInt32{}, sql.NullInt32{}) //nolint:exhaustruct

	err := s.checkWorkspaceRateLimit(ctx, WorkspaceRateLimitRequest{
		Session:               nil,
		AuthorizedWorkspaceID: wsID,
		RootKeyWorkspaceID:    rootWsID,
		Audit:                 testAuditContext(),
	})
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_NoQuota_FailsOpen(t *testing.T) {
	s, database := setupWorkspaceRatelimitTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)
	rootWsID := createTestWorkspace(t, database)

	// No quota row at all — should fail open
	err := s.checkWorkspaceRateLimit(ctx, WorkspaceRateLimitRequest{
		Session:               nil,
		AuthorizedWorkspaceID: wsID,
		RootKeyWorkspaceID:    rootWsID,
		Audit:                 testAuditContext(),
	})
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_ZeroLimit_BlocksAll(t *testing.T) {
	s, database := setupWorkspaceRatelimitTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)
	rootWsID := createTestWorkspace(t, database)

	// 0 limit = zero requests allowed
	upsertQuota(t, database, wsID,
		sql.NullInt32{Valid: true, Int32: 0},
		sql.NullInt32{Valid: true, Int32: 60000},
	)

	err := s.checkWorkspaceRateLimit(ctx, WorkspaceRateLimitRequest{
		Session:               nil,
		AuthorizedWorkspaceID: wsID,
		RootKeyWorkspaceID:    rootWsID,
		Audit:                 testAuditContext(),
	})
	require.Error(t, err)
	require.Contains(t, err.Error(), "workspace rate limit exceeded")
}

func TestCheckWorkspaceRateLimit_EnforcesLimit(t *testing.T) {
	s, database := setupWorkspaceRatelimitTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)
	rootWsID := createTestWorkspace(t, database)

	// Allow exactly 2 requests per 60s window
	upsertQuota(t, database, wsID,
		sql.NullInt32{Valid: true, Int32: 2},
		sql.NullInt32{Valid: true, Int32: 60000},
	)

	req := WorkspaceRateLimitRequest{
		Session:               nil,
		AuthorizedWorkspaceID: wsID,
		RootKeyWorkspaceID:    rootWsID,
		Audit:                 testAuditContext(),
	}

	// First two requests should succeed
	err := s.checkWorkspaceRateLimit(ctx, req)
	require.NoError(t, err)

	err = s.checkWorkspaceRateLimit(ctx, req)
	require.NoError(t, err)

	// Third request should be rate limited
	err = s.checkWorkspaceRateLimit(ctx, req)
	require.Error(t, err)
	require.Contains(t, err.Error(), "workspace rate limit exceeded")
}

func TestCheckWorkspaceRateLimit_CreatesNamespace(t *testing.T) {
	s, database := setupWorkspaceRatelimitTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)
	rootWsID := createTestWorkspace(t, database)

	upsertQuota(t, database, wsID,
		sql.NullInt32{Valid: true, Int32: 100},
		sql.NullInt32{Valid: true, Int32: 60000},
	)

	err := s.checkWorkspaceRateLimit(ctx, WorkspaceRateLimitRequest{
		Session:               nil,
		AuthorizedWorkspaceID: wsID,
		RootKeyWorkspaceID:    rootWsID,
		Audit:                 testAuditContext(),
	})
	require.NoError(t, err)

	// Verify namespace was created in the root key's workspace
	row, err := db.Query.FindRatelimitNamespaceByName(ctx, database.RO(), db.FindRatelimitNamespaceByNameParams{
		WorkspaceID: rootWsID,
		Name:        workspaceRatelimitNamespace,
	})
	require.NoError(t, err)
	require.Equal(t, workspaceRatelimitNamespace, row.Name)
	require.Equal(t, rootWsID, row.WorkspaceID)
}

func TestCheckWorkspaceRateLimit_ReturnsCorrectFaultCode(t *testing.T) {
	s, database := setupWorkspaceRatelimitTest(t)
	ctx := context.Background()
	wsID := createTestWorkspace(t, database)
	rootWsID := createTestWorkspace(t, database)

	upsertQuota(t, database, wsID,
		sql.NullInt32{Valid: true, Int32: 0},
		sql.NullInt32{Valid: true, Int32: 60000},
	)

	err := s.checkWorkspaceRateLimit(ctx, WorkspaceRateLimitRequest{
		Session:               nil,
		AuthorizedWorkspaceID: wsID,
		RootKeyWorkspaceID:    rootWsID,
		Audit:                 testAuditContext(),
	})
	require.Error(t, err)

	code, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Contains(t, code, "WorkspaceRateLimited")
}
