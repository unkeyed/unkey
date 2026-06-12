package deploybilling

import (
	"fmt"
	"sort"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
)

// Config holds the handler's dependencies.
type Config struct {
	// UsageReader queries month-to-date usage from ClickHouse. Optional: when
	// nil (ClickHouse not configured) the handler is a no-op.
	UsageReader UsageReader
	// Pusher reports usage to the billing provider. Must not be nil; use
	// billingmeter.NewNoop() to disable pushing.
	Pusher billingmeter.Pusher
	// DB is the primary application database, used to resolve each workspace's
	// Stripe subscription. Must not be nil.
	DB db.Database
	// Heartbeat is pinged on successful completion. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunDeployBillingPush.
type Handler struct {
	usage     UsageReader
	pusher    billingmeter.Pusher
	db        db.Database
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.Pusher, "Pusher must not be nil; use billingmeter.NewNoop()"),
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{
		usage:     cfg.UsageReader,
		pusher:    cfg.Pusher,
		db:        cfg.DB,
		heartbeat: cfg.Heartbeat,
	}, nil
}

// Handle computes month-to-date Deploy usage for the billing period (the VO
// key, "YYYY-MM") and pushes each billable workspace's running total to the
// provider, fanning out the pushes in bounded batches. The window runs from the
// first of the month to now; the pushed quantity is absolute, so re-runs and
// overlapping ticks converge.
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

	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return h.usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: "", // all workspaces; we filter to billable ones below
			Start:       p.Start().UnixMilli(),
			End:         nowUnix * 1000,
		})
	}, restate.WithName("get month-to-date usage"))
	if err != nil {
		return nil, fmt.Errorf("get month-to-date usage: %w", err)
	}

	valuesByWorkspace := aggregateUsage(rows)
	if len(valuesByWorkspace) == 0 {
		logger.Info("no deploy usage this period", "billing_period", period)
		return h.done(ctx, 0, 0)
	}

	// Sort so the downstream journaled steps (db fetch, per-workspace push)
	// replay in a stable order.
	workspaceIDs := make([]string, 0, len(valuesByWorkspace))
	for id := range valuesByWorkspace {
		workspaceIDs = append(workspaceIDs, id)
	}
	sort.Strings(workspaceIDs)

	billing, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesForDeployBillingByIDsRow, error) {
		return db.Query.ListWorkspacesForDeployBillingByIDs(rc, h.db.RO(), workspaceIDs)
	}, restate.WithName("fetch workspace billing identities"))
	if err != nil {
		return nil, fmt.Errorf("fetch workspace billing identities: %w", err)
	}

	billingByID := make(map[string]db.ListWorkspacesForDeployBillingByIDsRow, len(billing))
	for _, b := range billing {
		billingByID[b.ID] = b
	}

	tasks := make([]pushTask, 0, len(workspaceIDs))
	for _, id := range workspaceIDs {
		values := valuesByWorkspace[id]
		if !values.Positive() {
			continue
		}

		b, ok := billingByID[id]
		if !ok || !b.Enabled {
			continue
		}
		if !b.StripeCustomerID.Valid || b.StripeCustomerID.String == "" {
			logger.Info("workspace has deploy usage but no stripe customer; skipping",
				"workspace_id", id,
				"billing_period", period,
			)
			continue
		}

		tasks = append(tasks, pushTask{
			workspaceID: id,
			req: billingmeter.PushRequest{
				WorkspaceID:      id,
				StripeCustomerID: b.StripeCustomerID.String,
				Values:           values,
				Timestamp:        nowUnix,
			},
		})
	}

	workspacesPushed, metersPushed, err := h.pushAll(ctx, tasks)
	if err != nil {
		return nil, err
	}

	logger.Info("deploy billing push complete",
		"billing_period", period,
		"workspaces_with_usage", len(valuesByWorkspace),
		"workspaces_pushed", workspacesPushed,
		"meters_pushed", metersPushed,
	)
	return h.done(ctx, workspacesPushed, metersPushed)
}

// done pings the heartbeat and returns the response. Pulled out so the
// early-return (no-usage) path also reports a successful run.
func (h *Handler) done(ctx restate.ObjectContext, workspacesPushed, metersPushed int) (*hydrav1.RunDeployBillingPushResponse, error) {
	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}
	return &hydrav1.RunDeployBillingPushResponse{
		WorkspacesPushed: int32(workspacesPushed),
		MetersPushed:     int32(metersPushed),
	}, nil
}
