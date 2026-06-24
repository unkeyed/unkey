package deployteardown

import (
	"time"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
)

// VirtualObject implements DeployTeardownService. It serializes teardowns per
// workspace (the virtual object key is the workspace id) and drives the
// stop-and-drain against the database, fanning out to the per-deployment
// DeploymentService for the actual state change.
type VirtualObject struct {
	hydrav1.UnimplementedDeployTeardownServiceServer
	db                db.Database
	drainPollInterval time.Duration
	drainGraceTimeout time.Duration
}

var _ hydrav1.DeployTeardownServiceServer = (*VirtualObject)(nil)

// Config holds the dependencies required to create a VirtualObject.
type Config struct {
	// DB is the primary application database, used to snapshot running
	// deployments, clear current-deployment pointers, and poll for drain. Must
	// not be nil.
	DB db.Database

	// DrainPollInterval is how long Teardown sleeps between drain checks. Zero
	// uses the production default (defaultDrainPollInterval). Tests set a short
	// interval to keep the drain loop fast.
	DrainPollInterval time.Duration

	// DrainGraceTimeout bounds the wait for compute to drain before Teardown
	// forces completion. Zero uses the production default
	// (defaultDrainGraceTimeout).
	DrainGraceTimeout time.Duration
}

// New constructs a VirtualObject. It returns an error if any required
// dependency is missing.
func New(cfg Config) (*VirtualObject, error) {
	if err := assert.NotNil(cfg.DB, "DB must not be nil"); err != nil {
		return nil, err
	}
	if cfg.DrainPollInterval <= 0 {
		cfg.DrainPollInterval = defaultDrainPollInterval
	}
	if cfg.DrainGraceTimeout <= 0 {
		cfg.DrainGraceTimeout = defaultDrainGraceTimeout
	}
	return &VirtualObject{
		UnimplementedDeployTeardownServiceServer: hydrav1.UnimplementedDeployTeardownServiceServer{},
		db:                                       cfg.DB,
		drainPollInterval:                        cfg.DrainPollInterval,
		drainGraceTimeout:                        cfg.DrainGraceTimeout,
	}, nil
}
