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
	"github.com/unkeyed/unkey/svc/ctrl/internal/invoicecloser"
)

// Config holds handler dependencies.
type Config struct {
	// Nil without ClickHouse. Handler is a no-op.
	UsageReader UsageReader
	// Must not be nil. Use billingmeter.NewNoop() to disable.
	Pusher billingmeter.Pusher
	// Must not be nil.
	DB db.Database
	// Must not be nil. Use healthcheck.NewNoop() when monitoring is off.
	Heartbeat healthcheck.Heartbeat
	// Must not be nil. Use invoicecloser.NewNoop() to disable.
	Closer invoicecloser.Closer
	// Must not be nil. Use healthcheck.NewNoop() when monitoring is off.
	CloseHeartbeat healthcheck.Heartbeat
}

// Handler runs RunDeployBillingPush and RunDeployBillingClose.
type Handler struct {
	usage          UsageReader
	pusher         billingmeter.Pusher
	db             db.Database
	heartbeat      healthcheck.Heartbeat
	closer         invoicecloser.Closer
	closeHeartbeat healthcheck.Heartbeat
}

func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.Pusher, "Pusher must not be nil; use billingmeter.NewNoop()"),
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Closer, "Closer must not be nil; use invoicecloser.NewNoop()"),
		assert.NotNil(cfg.CloseHeartbeat, "CloseHeartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{
		usage:          cfg.UsageReader,
		pusher:         cfg.Pusher,
		db:             cfg.DB,
		heartbeat:      cfg.Heartbeat,
		closer:         cfg.Closer,
		closeHeartbeat: cfg.CloseHeartbeat,
	}, nil
}

// Handle pushes month-to-date Deploy usage for the VO key ("YYYY-MM").
// Absolute totals; re-runs converge.
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
	// One journaled "now" for both the ClickHouse window (ms) and meter timestamp (s).
	nowMillis := nowTime.UnixMilli()
	nowUnixSeconds := nowTime.Unix()

	workspacesWithUsage, workspacesPushed, metersPushed, failedWorkspaceIDs, err := h.pushUsage(
		ctx, period, p, nowMillis, nowUnixSeconds,
	)
	if err != nil {
		return nil, err
	}
	workspacesFailed := len(failedWorkspaceIDs)

	logger.Info("deploy billing push complete",
		"billing_period", period,
		"workspaces_with_usage", workspacesWithUsage,
		"workspaces_pushed", workspacesPushed,
		"workspaces_failed", workspacesFailed,
		"meters_pushed", metersPushed,
	)

	// Failed pushes still complete the run. Skip heartbeat so monitoring fires.
	if workspacesFailed > 0 {
		return &hydrav1.RunDeployBillingPushResponse{}, nil
	}

	if err := h.pingHeartbeat(ctx); err != nil {
		return nil, err
	}
	return &hydrav1.RunDeployBillingPushResponse{}, nil
}

// pushUsage reads [p.Start(), endMillis), pushes absolute totals stamped with
// eventTimestamp. Hourly push uses now for both; close uses period end and T-1s.
func (h *Handler) pushUsage(
	ctx restate.ObjectContext,
	period string,
	p billingperiod.Period,
	endMillis int64,
	eventTimestamp int64,
) (workspacesWithUsage, workspacesPushed, metersPushed int, failedWorkspaceIDs []string, err error) {
	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return h.usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: "", // all workspaces; we filter to billable ones below
			Start:       p.Start().UnixMilli(),
			End:         endMillis,
		})
	}, restate.WithName("get period usage"))
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("get period usage: %w", err)
	}

	keyRows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.ActiveKeysUsage, error) {
		return h.usage.GetActiveKeysUsage(rc, clickhouse.GetActiveKeysUsageRequest{
			WorkspaceID: "", // all workspaces; we filter to billable ones below
			Year:        p.Year,
			Month:       p.Month,
		})
	}, restate.WithName("get active keys"))
	if err != nil {
		return 0, 0, 0, nil, fmt.Errorf("get active keys: %w", err)
	}

	valuesByWorkspace := aggregateUsage(rows)
	mergeActiveKeys(valuesByWorkspace, keyRows)
	if len(valuesByWorkspace) == 0 {
		logger.Info("no deploy usage this period", "billing_period", period)
		return 0, 0, 0, nil, nil
	}

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
		return 0, 0, 0, nil, fmt.Errorf("fetch workspace billing identities: %w", err)
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
		// Disabled workspaces still owe incurred usage. Only skip without a Stripe customer.
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
				Timestamp:        eventTimestamp,
			},
		})
	}

	workspacesPushed, metersPushed, failedWorkspaceIDs, err = h.pushAll(ctx, tasks)
	if err != nil {
		return 0, 0, 0, nil, err
	}

	return len(valuesByWorkspace), workspacesPushed, metersPushed, failedWorkspaceIDs, nil
}

func (h *Handler) pingHeartbeat(ctx restate.ObjectContext) error {
	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return fmt.Errorf("send heartbeat: %w", err)
	}
	return nil
}
