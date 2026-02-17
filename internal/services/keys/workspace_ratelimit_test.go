package keys

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// mockQuotaCache is a minimal mock implementing cache.Cache[string, db.Quotum].
type mockQuotaCache struct {
	swrFn func(ctx context.Context, key string, refresh func(context.Context) (db.Quotum, error), op func(error) cache.Op) (db.Quotum, cache.CacheHit, error)
}

func (m *mockQuotaCache) Get(_ context.Context, _ string) (db.Quotum, cache.CacheHit) {
	return db.Quotum{}, cache.Miss
}
func (m *mockQuotaCache) GetMany(_ context.Context, _ []string) (map[string]db.Quotum, map[string]cache.CacheHit) {
	return nil, nil
}
func (m *mockQuotaCache) Set(_ context.Context, _ string, _ db.Quotum)      {}
func (m *mockQuotaCache) SetMany(_ context.Context, _ map[string]db.Quotum) {}
func (m *mockQuotaCache) SetNull(_ context.Context, _ string)               {}
func (m *mockQuotaCache) SetNullMany(_ context.Context, _ []string)         {}
func (m *mockQuotaCache) Remove(_ context.Context, _ ...string)             {}
func (m *mockQuotaCache) SWR(ctx context.Context, key string, refresh func(context.Context) (db.Quotum, error), op func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
	return m.swrFn(ctx, key, refresh, op)
}
func (m *mockQuotaCache) SWRMany(_ context.Context, _ []string, _ func(context.Context, []string) (map[string]db.Quotum, error), _ func(error) cache.Op) (map[string]db.Quotum, map[string]cache.CacheHit, error) {
	return nil, nil, nil
}
func (m *mockQuotaCache) SWRWithFallback(_ context.Context, _ []string, _ func(context.Context) (db.Quotum, string, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
	return db.Quotum{}, cache.Miss, nil
}
func (m *mockQuotaCache) Dump(_ context.Context) ([]byte, error)    { return nil, nil }
func (m *mockQuotaCache) Restore(_ context.Context, _ []byte) error { return nil }
func (m *mockQuotaCache) Clear(_ context.Context)                   {}
func (m *mockQuotaCache) Name() string                              { return "mock_quota" }

// mockRateLimiter implements ratelimit.Service for testing.
type mockRateLimiter struct {
	fn func(context.Context, ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error)
}

func (m *mockRateLimiter) Ratelimit(ctx context.Context, req ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
	return m.fn(ctx, req)
}

func (m *mockRateLimiter) RatelimitMany(_ context.Context, _ []ratelimit.RatelimitRequest) ([]ratelimit.RatelimitResponse, error) {
	return nil, nil
}

// mockNamespaceService implements namespace.Service for testing.
type mockNamespaceService struct {
	getFn        func(ctx context.Context, workspaceID, nameOrID string) (db.FindRatelimitNamespace, bool, error)
	createFn     func(ctx context.Context, workspaceID, name string, audit *namespace.AuditContext) (db.FindRatelimitNamespace, error)
	getManyFn    func(ctx context.Context, workspaceID string, names []string) (map[string]db.FindRatelimitNamespace, []string, error)
	createManyFn func(ctx context.Context, workspaceID string, names []string, audit *namespace.AuditContext) (map[string]db.FindRatelimitNamespace, error)
	invalidateFn func(ctx context.Context, workspaceID string, ns db.FindRatelimitNamespace)
}

func (m *mockNamespaceService) Get(ctx context.Context, workspaceID, nameOrID string) (db.FindRatelimitNamespace, bool, error) {
	if m.getFn != nil {
		return m.getFn(ctx, workspaceID, nameOrID)
	}
	return db.FindRatelimitNamespace{}, false, nil //nolint:exhaustruct
}

func (m *mockNamespaceService) Create(ctx context.Context, workspaceID, name string, audit *namespace.AuditContext) (db.FindRatelimitNamespace, error) {
	if m.createFn != nil {
		return m.createFn(ctx, workspaceID, name, audit)
	}
	return db.FindRatelimitNamespace{}, nil //nolint:exhaustruct
}

func (m *mockNamespaceService) GetMany(ctx context.Context, workspaceID string, names []string) (map[string]db.FindRatelimitNamespace, []string, error) {
	if m.getManyFn != nil {
		return m.getManyFn(ctx, workspaceID, names)
	}
	return nil, names, nil
}

func (m *mockNamespaceService) CreateMany(ctx context.Context, workspaceID string, names []string, audit *namespace.AuditContext) (map[string]db.FindRatelimitNamespace, error) {
	if m.createManyFn != nil {
		return m.createManyFn(ctx, workspaceID, names, audit)
	}
	return nil, nil
}

func (m *mockNamespaceService) Invalidate(ctx context.Context, workspaceID string, ns db.FindRatelimitNamespace) {
	if m.invalidateFn != nil {
		m.invalidateFn(ctx, workspaceID, ns)
	}
}

func quotaCache(limit, duration int32) *mockQuotaCache {
	return &mockQuotaCache{
		swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
			return db.Quotum{
				RatelimitLimit:    sql.NullInt32{Valid: true, Int32: limit},
				RatelimitDuration: sql.NullInt32{Valid: true, Int32: duration},
			}, cache.Hit, nil
		},
	}
}

func unlimitedQuotaCache() *mockQuotaCache {
	return &mockQuotaCache{
		swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
			// NULL = unlimited
			return db.Quotum{}, cache.Hit, nil
		},
	}
}

func noopNamespaceService() *mockNamespaceService {
	return &mockNamespaceService{}
}

func TestCheckWorkspaceRateLimit_NullLimit_Unlimited(t *testing.T) {
	t.Parallel()

	s := &service{
		quotaCache:                unlimitedQuotaCache(),
		ratelimitNamespaceService: noopNamespaceService(),
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_NoQuotaRow(t *testing.T) {
	t.Parallel()

	s := &service{
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{}, cache.Null, nil
			},
		},
		ratelimitNamespaceService: noopNamespaceService(),
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_LimitZero_BlocksAll(t *testing.T) {
	t.Parallel()

	rl := &mockRateLimiter{
		fn: func(_ context.Context, req ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
			require.Equal(t, int64(0), req.Limit)
			return ratelimit.RatelimitResponse{
				Success:   false,
				Limit:     0,
				Remaining: 0,
			}, nil
		},
	}

	s := &service{
		rateLimiter:               rl,
		quotaCache:                quotaCache(0, 60000),
		ratelimitNamespaceService: noopNamespaceService(),
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.User.TooManyRequests.WorkspaceRateLimited.URN(), urn)
}

func TestCheckWorkspaceRateLimit_UnderLimit(t *testing.T) {
	t.Parallel()

	rl := &mockRateLimiter{
		fn: func(_ context.Context, req ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
			require.Equal(t, "workspace.ratelimit", req.Name)
			require.Equal(t, "ws_123", req.Identifier)
			require.Equal(t, int64(100), req.Limit)
			require.Equal(t, 60*time.Second, req.Duration)
			require.Equal(t, int64(1), req.Cost)

			return ratelimit.RatelimitResponse{
				Success:   true,
				Limit:     100,
				Remaining: 99,
			}, nil
		},
	}

	s := &service{
		rateLimiter:               rl,
		quotaCache:                quotaCache(100, 60000),
		ratelimitNamespaceService: noopNamespaceService(),
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_OverLimit(t *testing.T) {
	t.Parallel()

	rl := &mockRateLimiter{
		fn: func(_ context.Context, _ ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
			return ratelimit.RatelimitResponse{
				Success:   false,
				Limit:     100,
				Remaining: 0,
			}, nil
		},
	}

	s := &service{
		rateLimiter:               rl,
		quotaCache:                quotaCache(100, 60000),
		ratelimitNamespaceService: noopNamespaceService(),
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.Error(t, err)

	urn, ok := fault.GetCode(err)
	require.True(t, ok)
	require.Equal(t, codes.User.TooManyRequests.WorkspaceRateLimited.URN(), urn)
}

func TestCheckWorkspaceRateLimit_CacheError_FailsOpen(t *testing.T) {
	t.Parallel()

	s := &service{
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{}, cache.Miss, fmt.Errorf("cache unavailable")
			},
		},
		ratelimitNamespaceService: noopNamespaceService(),
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_RateLimiterError_FailsOpen(t *testing.T) {
	t.Parallel()

	rl := &mockRateLimiter{
		fn: func(_ context.Context, _ ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
			return ratelimit.RatelimitResponse{}, fmt.Errorf("redis unavailable")
		},
	}

	s := &service{
		rateLimiter:               rl,
		quotaCache:                quotaCache(100, 60000),
		ratelimitNamespaceService: noopNamespaceService(),
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}
