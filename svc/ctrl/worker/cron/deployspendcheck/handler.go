package deployspendcheck

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
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// Config holds the orchestrator's dependencies. The per-workspace check
// (threshold state, alert email) runs in CheckHandler; this handler resolves
// who to check, prices everyone's usage from one ClickHouse scan, and fans out
// only to the workspaces that are near or over their budget.
type Config struct {
	// DB is the primary application database, used to list workspaces with a
	// configured Deploy spend budget. Must not be nil.
	DB db.Database

	// Usage queries month-to-date Deploy usage from ClickHouse. May be nil
	// (ClickHouse not configured), making the spend check a no-op.
	Usage deploybilling.UsageReader

	// Heartbeat is pinged when the orchestration completes. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
}

// Handler executes RunDeploySpendCheck: it lists budgeted workspaces, prices
// their month-to-date usage, and fans out to DeploySpendCheckService.
type Handler struct {
	db        db.Database
	usage     deploybilling.UsageReader
	heartbeat healthcheck.Heartbeat
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{db: cfg.DB, usage: cfg.Usage, heartbeat: cfg.Heartbeat}, nil
}

// Handle lists the workspaces that configured a Deploy spend budget (the VO key
// is the billing period "YYYY-MM"), prices every workspace's month-to-date
// usage from a single ClickHouse scan (the same one-scan shape as the hourly
// billing push), and fans out one check per ACTIONABLE workspace,
// fire-and-forget, with the priced gross carried in the request.
//
// Actionable means the net-of-credit overage has reached the lowest alert
// threshold: below it there is nothing to email and nothing to enforce, so
// dispatching would journal an invocation just to conclude that. This is what
// keeps the tick O(one scan + a handful of invocations) instead of O(budgeted
// workspaces): the per-workspace VO still owns the alert high-water mark and
// dedup, it just never runs for the quiet majority.
//
// A workspace whose included credit is not yet known is skipped: without it
// the overage can't be priced without counting the full gross, which would
// false-alarm. Each dispatched check runs, retries, and fails on its own, so a
// broken workspace cannot stall the others or this tick.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunDeploySpendCheckRequest,
) (*hydrav1.RunDeploySpendCheckResponse, error) {
	period := restate.Key(ctx)
	logger.Info("running deploy spend check", "billing_period", period)

	if h.usage == nil {
		logger.Info("deploy spend check disabled (no usage reader configured)",
			"billing_period", period,
		)
		return &hydrav1.RunDeploySpendCheckResponse{}, nil
	}

	budgeted, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesWithDeployBudgetRow, error) {
		return h.db.ListWorkspacesWithDeployBudget(rc)
	}, restate.WithName("list budgeted workspaces"))
	if err != nil {
		return nil, fmt.Errorf("list budgeted workspaces: %w", err)
	}

	// Sort by id so the fan-out order is stable across replays: each Send is
	// journaled by position, so a different iteration order on replay would
	// dispatch a different request at the same journal index and diverge.
	sort.Slice(budgeted, func(i, j int) bool { return budgeted[i].ID < budgeted[j].ID })

	values, err := h.priceUsage(ctx, period)
	if err != nil {
		return nil, err
	}

	var dispatched, skippedNoCredit, skippedBelowThreshold int32
	for _, ws := range budgeted {
		if !ws.DeploySpendBudgetCents.Valid {
			continue // query filters these out; guard against a future query change
		}

		if !ws.DeployIncludedCreditCents.Valid {
			skippedNoCredit++
			// Error, not info: an unknown credit disables both budget alerts
			// and the spend cap for this workspace for as long as it persists.
			// The dashboard webhook re-persists it on the next invoice event,
			// so a workspace stuck here means that recovery path is broken.
			logger.Error("skip spend check: included credit unknown; alerts and spend cap disabled for this workspace",
				"workspace_id", ws.ID,
				"billing_period", period,
			)
			continue
		}

		// All spend math is integer micro-cents: the pricing quantized once in
		// PriceMicroCents, and cents-denominated columns scale exactly.
		gross := deploybilling.PriceMicroCents(values[ws.ID])
		overage := gross - ws.DeployIncludedCreditCents.Int64*deploybilling.MicroCentsPerCent
		if overage < 0 {
			overage = 0
		}
		// Below the lowest alert threshold nothing can happen in the check;
		// skip the invocation entirely. At or past it, the check owns the
		// decision: its high-water mark decides whether an email is actually
		// due, so re-dispatching an already-alerted workspace is a cheap no-op.
		if crossedThreshold(overage, ws.DeploySpendBudgetCents.Int64*deploybilling.MicroCentsPerCent) == 0 {
			skippedBelowThreshold++
			continue
		}

		// Fire-and-forget: Send is journaled, so a replay does not re-dispatch.
		// The child invocation owns retries, state, and the email send.
		hydrav1.NewDeploySpendCheckServiceClient(ctx, ws.ID).
			CheckWorkspaceSpend().
			Send(&hydrav1.CheckWorkspaceSpendRequest{
				Period:              period,
				BudgetCents:         ws.DeploySpendBudgetCents.Int64,
				IncludedCreditCents: ws.DeployIncludedCreditCents.Int64,
				Stop:                ws.DeploySpendBudgetStop,
				OrgId:               ws.OrgID,
				WorkspaceName:       ws.Name,
				WorkspaceSlug:       ws.Slug,
				GrossMicroCents:     gross,
			})
		dispatched++
	}

	logger.Info("deploy spend check dispatched",
		"billing_period", period,
		"workspaces_dispatched", dispatched,
		"workspaces_skipped_no_credit", skippedNoCredit,
		"workspaces_skipped_below_threshold", skippedBelowThreshold,
	)

	if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunDeploySpendCheckResponse{
		WorkspacesDispatched:      dispatched,
		WorkspacesSkippedNoCredit: skippedNoCredit,
	}, nil
}

// priceUsage reads the period's month-to-date usage for ALL workspaces in one
// grouped ClickHouse scan (two queries: instance meters and active keys) and
// returns the aggregated MeterValues keyed by workspace id. One scan for the
// whole fleet is how ClickHouse wants to be read; per-workspace point queries
// at fan-out scale are not. The read is capped at the period's end so a stale
// invocation running after the roll cannot fold the next month into this
// period's decisions.
func (h *Handler) priceUsage(
	ctx restate.ObjectContext,
	period string,
) (map[string]billingmeter.MeterValues, error) {
	p, err := billingperiod.Parse(period)
	if err != nil {
		return nil, restate.TerminalError(fmt.Errorf("invalid billing period %q: %w", period, err))
	}
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}

	startMillis := p.Start().UnixMilli()
	endMillis := now.UnixMilli()
	if periodEndMillis := p.End().UnixMilli(); endMillis > periodEndMillis {
		endMillis = periodEndMillis
	}

	instanceRows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return h.usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: "", // all workspaces, same shape as the hourly push
			Start:       startMillis,
			End:         endMillis,
		})
	}, restate.WithName("get instance usage"))
	if err != nil {
		return nil, fmt.Errorf("get instance usage: %w", err)
	}

	keyRows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.ActiveKeysUsage, error) {
		return h.usage.GetActiveKeysUsage(rc, clickhouse.GetActiveKeysUsageRequest{
			WorkspaceID: "", // all workspaces, same shape as the hourly push
			Month:       startMillis,
		})
	}, restate.WithName("get active keys usage"))
	if err != nil {
		return nil, fmt.Errorf("get active keys usage: %w", err)
	}

	values := deploybilling.AggregateUsage(instanceRows)
	deploybilling.MergeActiveKeys(values, keyRows)
	return values, nil
}
