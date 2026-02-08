package keyrefill

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// Service implements the KeyRefillService Restate virtual object.
type Service struct {
	hydrav1.UnimplementedKeyRefillServiceServer
	db        db.Database
	logger    logging.Logger
	heartbeat healthcheck.Heartbeat
}

var _ hydrav1.KeyRefillServiceServer = (*Service)(nil)

// Config holds the configuration for the key refill service.
type Config struct {
	DB     db.Database
	Logger logging.Logger
	// Heartbeat sends health signals after successful refill runs.
	// Must not be nil - use healthcheck.NewNoop() if monitoring is not needed.
	Heartbeat healthcheck.Heartbeat
}

// New creates a new key refill service.
func New(cfg Config) (*Service, error) {
	if err := assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop() if not needed"); err != nil {
		return nil, err
	}

	return &Service{
		UnimplementedKeyRefillServiceServer: hydrav1.UnimplementedKeyRefillServiceServer{},
		db:                                  cfg.DB,
		logger:                              cfg.Logger,
		heartbeat:                           cfg.Heartbeat,
	}, nil
}
