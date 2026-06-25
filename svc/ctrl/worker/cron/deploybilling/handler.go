package deploybilling

import (
	"fmt"
	"sort"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
	// The ClickHouse usage query is bounded in milliseconds; the Stripe meter
	// event timestamp is unix seconds. Both come from the one journaled "now"
	// so the window and the event agree.
	nowMillis := nowTime.UnixMilli()
	nowUnixSeconds := nowTime.Unix()

	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return h.usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: "", // all workspaces; we filter to billable ones below
			Start:       p.Start().UnixMilli(),
			End:         nowMillis,
		})
	}, restate.WithName("get month-to-date usage"))
	if err != nil {
		return nil, fmt.Errorf("get month-to-date usage: %w", err)
	}

	if len(rows) == 0 {
		logger.Info("no deploy usage this period", "billing_period", period)
		if err := h.pingHeartbeat(ctx); err != nil {
			return nil, err
		}
		return &hydrav1.RunDeployBillingPushResponse{}, nil
	}

	valuesByWorkspace := aggregateUsage(rows)

	// Sort so the downstream journaled steps (db fetch, per-workspace push)
	// replay in a stable order.
	workspaceIDs := make([]string, 0, len(valuesByWorkspace))
	for id := range valuesByWorkspace {
		workspaceIDs = append(workspaceIDs, id)
	}
	sort.Strings(workspaceIDs)

	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesForDeployBillingByIDsRow, error) {
		return h.db.ListWorkspacesForDeployBillingByIDs(rc, workspaceIDs)
	}, restate.WithName("fetch workspace billing identities"))
	if err != nil {
		return nil, fmt.Errorf("fetch workspace billing identities: %w", err)
	}

	workspacesByID := make(map[string]db.ListWorkspacesForDeployBillingByIDsRow, len(workspaces))
	for _, w := range workspaces {
		workspacesByID[w.ID] = w
	}

	tasks := make([]pushTask, 0, len(workspaceIDs))
	for _, id := range workspaceIDs {
		values := valuesByWorkspace[id]
		if !values.Positive() {
			continue
		}

		w, ok := workspacesByID[id]
		if !ok {
			continue
		}
		// A disabled workspace is still billed: usage already incurred is owed
		// regardless of the workspace's current state. The only blocker is a
		// missing Stripe customer, since there is nothing to map the usage onto.
		if !w.StripeCustomerID.Valid || w.StripeCustomerID.String == "" {
			logger.Info("workspace has deploy usage but no stripe customer; skipping",
				"workspace_id", id,
				"billing_period", period,
			)
			continue
		}

		tasks = append(tasks, pushTask{
			workspaceID: id,
			req: billingmeter.PushRequest{
				StripeCustomerID: w.StripeCustomerID.String,
				Values:           values,
				Timestamp:        nowUnixSeconds,
			},
		})
	}

	workspacesPushed, metersPushed, err := h.pushAll(ctx, tasks)
	if err != nil {
		return nil, err
	}

	if err := h.pingHeartbeat(ctx); err != nil {
		return nil, err
	}

	logger.Info("deploy billing push complete",
		"billing_period", period,
		"workspaces_with_usage", len(valuesByWorkspace),
		"workspaces_pushed", workspacesPushed,
		"meters_pushed", metersPushed,
	)
	return &hydrav1.RunDeployBillingPushResponse{}, nil
}

// pingHeartbeat reports a successful run to the monitoring heartbeat.
func (h *Handler) pingHeartbeat(ctx restate.ObjectContext) error {
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	return nil
}
