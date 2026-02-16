package keys

import (
	"context"
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/internal/services/ratelimit"
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

func (m *mockRateLimiter) RatelimitMany(ctx context.Context, reqs []ratelimit.RatelimitRequest) ([]ratelimit.RatelimitResponse, error) {
	return nil, nil
}

func TestCheckWorkspaceRateLimit_NilCache(t *testing.T) {
	t.Parallel()

	s := &service{quotaCache: nil}

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
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_LimitZero(t *testing.T) {
	t.Parallel()

	s := &service{
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{
					RatelimitLimit:    sql.NullInt64{Valid: true, Int64: 0},
					RatelimitDuration: sql.NullInt64{Valid: true, Int64: 60000},
				}, cache.Hit, nil
			},
		},
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_DurationZero(t *testing.T) {
	t.Parallel()

	s := &service{
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{
					RatelimitLimit:    sql.NullInt64{Valid: true, Int64: 100},
					RatelimitDuration: sql.NullInt64{Valid: true, Int64: 0},
				}, cache.Hit, nil
			},
		},
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_UnderLimit(t *testing.T) {
	t.Parallel()

	rl := &mockRateLimiter{
		fn: func(_ context.Context, req ratelimit.RatelimitRequest) (ratelimit.RatelimitResponse, error) {
			// Without a namespace service, falls back to the constant name
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
		rateLimiter: rl,
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{
					RatelimitLimit:    sql.NullInt64{Valid: true, Int64: 100},
					RatelimitDuration: sql.NullInt64{Valid: true, Int64: 60000},
				}, cache.Hit, nil
			},
		},
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
		rateLimiter: rl,
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{
					RatelimitLimit:    sql.NullInt64{Valid: true, Int64: 100},
					RatelimitDuration: sql.NullInt64{Valid: true, Int64: 60000},
				}, cache.Hit, nil
			},
		},
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
		rateLimiter: rl,
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{
					RatelimitLimit:    sql.NullInt64{Valid: true, Int64: 100},
					RatelimitDuration: sql.NullInt64{Valid: true, Int64: 60000},
				}, cache.Hit, nil
			},
		},
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}

func TestCheckWorkspaceRateLimit_NegativeLimit(t *testing.T) {
	t.Parallel()

	s := &service{
		quotaCache: &mockQuotaCache{
			swrFn: func(_ context.Context, _ string, _ func(context.Context) (db.Quotum, error), _ func(error) cache.Op) (db.Quotum, cache.CacheHit, error) {
				return db.Quotum{
					RatelimitLimit:    sql.NullInt64{Valid: true, Int64: -1},
					RatelimitDuration: sql.NullInt64{Valid: true, Int64: 60000},
				}, cache.Hit, nil
			},
		},
	}

	err := s.checkWorkspaceRateLimit(context.Background(), nil, "ws_123", nil)
	require.NoError(t, err)
}
