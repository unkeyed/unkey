package deploybilling

import (
	"fmt"
	"sort"
	"time"

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
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/enduserbillingpush"
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
	// EndUserBilling optionally closes the same billing period for end-user
	// billing (customers billing their own users via Stripe Connect). Nil
	// disables the phase. It shares this cron's monthly tick because the
	// proto toolchain to mint a dedicated CronService RPC is unavailable.
	EndUserBilling *enduserbillingpush.PeriodClose
}

// Handler executes RunDeployBillingPush.
type Handler struct {
	usage          UsageReader
	pusher         billingmeter.Pusher
	db             db.Database
	heartbeat      healthcheck.Heartbeat
	endUserBilling *enduserbillingpush.PeriodClose
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
		usage:          cfg.UsageReader,
		pusher:         cfg.Pusher,
		db:             cfg.DB,
		heartbeat:      cfg.Heartbeat,
		endUserBilling: cfg.EndUserBilling,
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

	// End-user billing phase: independent of Deploy usage, so it runs before
	// the deploy-usage early return. One journaled step; errors inside a
	// workspace are collected in the summary, not fatal to the deploy push.
	//
	// Unlike the Deploy push (which reports absolute, set-wins meter events for
	// the open month), end-user billing dispatches ADDITIVE Stripe invoice
	// items, so it must only ever bill a CLOSED period: it bills the month
	// before the tick's period, never the open one. runEndUserBilling also
	// gates on a short settle window and relies on per-identity run-once
	// markers so the hourly cadence bills each closed period exactly once.
	if h.endUserBilling != nil {
		h.runEndUserBilling(ctx, p, nowTime)
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

// endUserBillingSettleDays is how many days into the current month the
// previous month must have been closed before its end-user usage is billed.
// It gives late-arriving usage events (timestamped in the closed month but
// ingested early in the new one) time to settle in ClickHouse before the
// additive invoice items are cut. The hourly cadence bills the closed month on
// the first tick past this window; per-identity run-once markers keep every
// later tick a no-op.
const endUserBillingSettleDays = 2

// runEndUserBilling closes end-user billing for the month BEFORE p (the most
// recently closed period), gated on the settle window. Per-workspace failures
// are collected in the summary rather than failing the step, so one customer's
// misconfiguration never blocks the Deploy push or another customer's billing;
// a non-empty summary is surfaced at Warn.
// closedPeriodToBill returns the closed month that end-user billing should
// bill for the tick's (open) period p, and whether the settle window has
// elapsed. The billed month is always p's predecessor (closed, usage final);
// ready is false until the current month is at least endUserBillingSettleDays
// old, giving late-arriving usage time to settle before the additive invoice
// items are cut. Pure so it can be tested without a restate context.
func closedPeriodToBill(p billingperiod.Period, nowTime time.Time) (year, month int, ready bool) {
	closed := p.Start().AddDate(0, -1, 0)
	return closed.Year(), int(closed.Month()), nowTime.Day() >= endUserBillingSettleDays
}

func (h *Handler) runEndUserBilling(ctx restate.ObjectContext, p billingperiod.Period, nowTime time.Time) {
	year, month, ready := closedPeriodToBill(p, nowTime)
	if !ready {
		logger.Info("end-user billing deferred until closed period settles",
			"closed_year", year, "closed_month", month, "day_of_month", nowTime.Day(),
		)
		return
	}

	euSummary, euErr := restate.Run(ctx, func(rc restate.RunContext) (enduserbillingpush.Summary, error) {
		return h.endUserBilling.Run(rc, year, month)
	}, restate.WithName("end-user billing period close"))
	if euErr != nil {
		// A hard failure (e.g. listing connected workspaces) is logged, not
		// propagated: the Deploy push must still run. The step is journaled, so
		// the next tick retries the closed period, and run-once markers keep it
		// safe.
		logger.Error("end-user billing period close failed",
			"closed_year", year, "closed_month", month, "error", euErr.Error(),
		)
		return
	}
	if len(euSummary.Errors) > 0 {
		logger.Warn("end-user billing period close completed with per-workspace failures",
			"closed_year", year, "closed_month", month,
			"workspaces", euSummary.Workspaces,
			"records_pushed", euSummary.RecordsPushed,
			"failures", len(euSummary.Errors),
		)
	}
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
