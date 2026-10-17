package flags

import (
	"context"
	"maps"
	"time"

	"github.com/unkeyed/unkey/internal/services/flags/db"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/mysql"
)

// Config supplies the database and clock. The caller retains database ownership.
type Config struct {
	DB    mysql.MySQL
	Clock clock.Clock
}

type service struct {
	db    mysql.MySQL
	cache cache.Cache[string, map[string]Flag]
}

var _ Service = (*service)(nil)

// New creates a [Service] with a bounded cache of resolved flags per workspace.
// Entries are fresh for 10 seconds and can be served during background refresh
// for up to one minute after loading. Call Close at shutdown.
func New(config Config) (*service, error) {
	if err := assert.All(
		assert.NotNil(config.DB, "database is required"),
		assert.NotNil(config.Clock, "clock is required"),
	); err != nil {
		return nil, fault.Wrap(err, fault.Internal("invalid flags configuration"))
	}

	flagCache, err := cache.New(cache.Config[string, map[string]Flag]{
		Fresh:    10 * time.Second,
		Stale:    time.Minute,
		MaxSize:  10_000,
		Resource: "workspace_flags",
		Clock:    config.Clock,
	})
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("create flags cache"))
	}
	return &service{db: config.DB, cache: flagCache}, nil
}

// Close stops cache workers without closing the database. It is idempotent.
func (s *service) Close() {
	s.cache.Close()
}

func (s *service) Invalidate(ctx context.Context, workspaceID string) {
	s.cache.Remove(ctx, workspaceID)
}

func (s *service) Values(ctx context.Context, workspaceID string) (map[string]Flag, error) {
	values, err := s.load(ctx, workspaceID)
	if err != nil {
		return nil, err
	}
	result := make(map[string]Flag, len(values))
	maps.Copy(result, values)
	return result, nil
}

func (s *service) IsPrivateNetworkingEnabled(ctx context.Context, workspaceID string) bool {
	return s.enabled(ctx, workspaceID, "private_network_enable", false)
}

func (s *service) enabled(ctx context.Context, workspaceID, slug string, fallback bool) bool {
	values, err := s.load(ctx, workspaceID)
	if err != nil {
		logger.Warn("flag evaluation failed, using code default", "error", err, "workspace_id", workspaceID, "flag", slug)
		return fallback
	}
	if value, ok := values[slug]; ok {
		return value.Enabled
	}
	return fallback
}

func (s *service) load(ctx context.Context, workspaceID string) (map[string]Flag, error) {
	values, _, err := s.cache.SWR(ctx, workspaceID, func(ctx context.Context) (map[string]Flag, error) {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		rows, err := db.Query.ListFlagsByWorkspaceID(ctx, s.db.RO(), workspaceID)
		if err != nil {
			return nil, fault.Wrap(err, fault.Internal("load workspace flags"))
		}
		values := make(map[string]Flag, len(rows))
		for _, row := range rows {
			enabled := row.DefaultEnabled
			if row.OverrideEnabled.Valid {
				enabled = row.OverrideEnabled.Bool
			}
			values[row.Slug] = Flag{
				Enabled:     enabled,
				Description: row.Description,
				AllowOptIn:  row.AllowOptIn,
				AllowOptOut: row.AllowOptOut,
			}
		}
		return values, nil
	}, func(err error) cache.Op {
		if err != nil {
			return cache.Noop
		}
		return cache.WriteValue
	})
	return values, err
}
