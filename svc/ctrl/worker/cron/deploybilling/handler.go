package deploybilling

import (
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/invoicecloser"
)

// Config holds the orchestrator handler's dependencies. The push itself runs
// in DeployBillingPushService (see PushHandler); this handler only resolves
// who to bill and fans out to it.
type Config struct {
	// UsageReader queries month-to-date usage from ClickHouse. Optional: when
	// nil (ClickHouse not configured) the handler is a no-op.
	UsageReader UsageReader
	// DB is the primary application database, used to resolve each workspace's
	// Stripe subscription. Must not be nil.
	DB db.Database
	// Heartbeat is pinged when the hourly orchestration completes. Must not be
	// nil; use healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
	// Closer lists and finalizes draft invoices for the month-end close.
	// Must not be nil; use invoicecloser.NewNoop() to disable finalization.
	Closer invoicecloser.Closer
	// CloseHeartbeat is pinged when a month-end close completes. Must not be
	// nil; use healthcheck.NewNoop() if monitoring is not configured.
	CloseHeartbeat healthcheck.Heartbeat
	// FinalizeDelay is the wait between the close's final meter push and
	// invoice finalization, covering Stripe's asynchronous aggregation of
	// meter events into the draft's lines. Zero disables the wait (tests and
	// noop wiring); production passes DefaultFinalizeDelay.
	FinalizeDelay time.Duration
}

// Handler executes RunDeployBillingPush (see push.go) and RunDeployBillingClose
// (see close.go). It resolves billable workspaces and fans out to
// DeployBillingPushService; the provider push lives there so each workspace
// retries and fails in isolation.
type Handler struct {
	usage          UsageReader
	db             db.Database
	heartbeat      healthcheck.Heartbeat
	closer         invoicecloser.Closer
	closeHeartbeat healthcheck.Heartbeat
	finalizeDelay  time.Duration
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Closer, "Closer must not be nil; use invoicecloser.NewNoop()"),
		assert.NotNil(cfg.CloseHeartbeat, "CloseHeartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{
		usage:          cfg.UsageReader,
		db:             cfg.DB,
		heartbeat:      cfg.Heartbeat,
		closer:         cfg.Closer,
		closeHeartbeat: cfg.CloseHeartbeat,
		finalizeDelay:  cfg.FinalizeDelay,
	}, nil
}
