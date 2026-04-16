package sentinel

import (
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
)

// Service implements the SentinelService Restate service for deploying
// and tracking individual sentinel configuration changes.
type Service struct {
	hydrav1.UnimplementedSentinelServiceServer
	db db.Database
}

var _ hydrav1.SentinelServiceServer = (*Service)(nil)

// Config holds the configuration for the sentinel service.
type Config struct {
	DB db.Database
}

// New creates a new sentinel service.
func New(cfg Config) *Service {
	return &Service{
		UnimplementedSentinelServiceServer: hydrav1.UnimplementedSentinelServiceServer{},
		db:                                 cfg.DB,
	}
}
