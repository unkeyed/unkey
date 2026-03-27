package keylastusedsync

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
)

// Service implements the KeyLastUsedSyncService orchestrator.
// It fans out to KeyLastUsedPartitionService instances and collects results.
type Service struct {
	hydrav1.UnimplementedKeyLastUsedSyncServiceServer
	heartbeat healthcheck.Heartbeat
}

var _ hydrav1.KeyLastUsedSyncServiceServer = (*Service)(nil)

// Config holds the configuration for the orchestrator service.
type Config struct {
	// Heartbeat sends health signals after successful sync runs.
	// Must not be nil - use healthcheck.NewNoop() if monitoring is not needed.
	Heartbeat healthcheck.Heartbeat
}

// New creates a new orchestrator service.
func New(cfg Config) (*Service, error) {
	if err := assert.All(
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop() if not needed"),
	); err != nil {
		return nil, err
	}

	return &Service{
		UnimplementedKeyLastUsedSyncServiceServer: hydrav1.UnimplementedKeyLastUsedSyncServiceServer{},
		heartbeat: cfg.Heartbeat,
	}, nil
}
