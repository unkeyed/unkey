package keylastusedsync

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
)

// Service implements the KeyLastUsedSyncService Restate virtual object.
type Service struct {
	hydrav1.UnimplementedKeyLastUsedSyncServiceServer
	db         db.Database
	clickhouse clickhouse.ClickHouse
	heartbeat  healthcheck.Heartbeat
}

var _ hydrav1.KeyLastUsedSyncServiceServer = (*Service)(nil)

// Config holds the configuration for the key last used sync service.
type Config struct {
	DB         db.Database
	Clickhouse clickhouse.ClickHouse
	// Heartbeat sends health signals after successful sync runs.
	// Must not be nil - use healthcheck.NewNoop() if monitoring is not needed.
	Heartbeat healthcheck.Heartbeat
}

// New creates a new key last used sync service.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop() if not needed"),
	); err != nil {
		return nil, err
	}

	return &Service{
		UnimplementedKeyLastUsedSyncServiceServer: hydrav1.UnimplementedKeyLastUsedSyncServiceServer{},
		db:         cfg.DB,
		clickhouse: cfg.Clickhouse,
		heartbeat:  cfg.Heartbeat,
	}, nil
}
