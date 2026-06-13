package deploybilling

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
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
}

// Handler executes RunDeployBillingPush and RunDeployBillingClose. It resolves
// billable workspaces and fans out to DeployBillingPushService; the provider
// push lives there so each workspace retries and fails in isolation.
type Handler struct {
	usage          UsageReader
	db             db.Database
	heartbeat      healthcheck.Heartbeat
	closer         invoicecloser.Closer
	closeHeartbeat healthcheck.Heartbeat
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
	}, nil
}

// Handle computes month-to-date Deploy usage for the billing period (the VO
// key, "YYYY-MM") and fans out one push invocation per billable workspace,
// fire-and-forget. Each push runs, retries, and fails on its own; this handler
// only dispatches, so a broken workspace cannot stall the others or this tick.
// The pushed quantity is absolute, so re-runs and overlapping ticks converge.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeployBillingPushRequest,
) (*hydrav1.RunDeployBillingPushResponse, error) {
	period := restate.Key(ctx)
	logger.Info("running deploy billing push", "billing_period", period)

	if h.usage == nil {
		logger.Info("deploy billing push disabled (no usage reader configured)")
		return &hydrav1.RunDeployBillingPushResponse{}, nil
	}

	p, err := billingperiod.Parse(period)
	if err != nil {
		return nil, fmt.Errorf("invalid billing period %q: %w", period, err)
	}

	nowTime, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	nowUnix := nowTime.Unix()

	tasks, workspacesWithUsage, err := h.resolvePushTasks(ctx, period, p, nowUnix*1000, nowUnix)
	if err != nil {
		return nil, err
	}

	// Fire-and-forget: dispatch each workspace's push and move on. Send is a
	// journaled action, so a replay of this orchestrator does not re-dispatch.
	// The child invocation owns retries and failure; this tick succeeds once
	// the work is handed off.
	for _, task := range tasks {
		hydrav1.NewDeployBillingPushServiceClient(ctx, task.workspaceID).
			PushWorkspaceUsage().
			Send(task.pushRequest())
	}

	logger.Info("deploy billing push dispatched",
		"billing_period", period,
		"workspaces_with_usage", workspacesWithUsage,
		"workspaces_dispatched", len(tasks),
	)

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	// MetersPushed is not known here (the children push); report dispatched
	// workspaces so the orchestration still has a meaningful count.
	return &hydrav1.RunDeployBillingPushResponse{
		WorkspacesPushed: int32(len(tasks)),
	}, nil
}
