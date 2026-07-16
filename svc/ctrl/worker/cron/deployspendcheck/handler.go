package deployspendcheck

import (
	"fmt"
	"sort"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/billingmeter"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// Config holds the orchestrator's dependencies. The per-workspace check
// (threshold state, alert email) runs in CheckHandler; this handler resolves
// who to check, prices their usage from one scoped ClickHouse scan, and fans
// out only to the workspaces that are near or over their budget.
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
// is the billing period "YYYY-MM"), prices their month-to-date usage from a
// single ClickHouse scan scoped to that set (the same one-scan shape as the
// hourly billing push), fans out one check per ACTIONABLE workspace with the
// priced gross carried in the request, and awaits the outcomes, withholding
// the heartbeat when any check failed.
//
// Actionable means gross month-to-date metered spend has reached the lowest
// alert threshold. The cap is on total spend, credits included, so no included
// credit is subtracted here. Below the lowest threshold there is nothing to
// email and nothing to enforce, so dispatching would journal an invocation just
// to conclude that. This is what keeps the tick O(one scan + a handful of
// invocations) instead of O(budgeted workspaces): the per-workspace VO still
// owns the alert high-water mark and dedup, it just never runs for the quiet
// majority.
//
// Each dispatched check runs and retries in its own VO with a capped policy
// (see worker/run.go), so a broken workspace cannot stall the others; its
// failure surfaces here as a withheld heartbeat.
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

	// No budgeted workspaces: nothing to price and nothing to fan out to. Return
	// before priceUsage, because FleetMeterValues treats an empty id set as "no
	// filter" and would scan the whole fleet's month at cron cadence, the exact
	// cost the scoping exists to avoid. Still ping the heartbeat so the dead-man
	// switch stays satisfied on quiet ticks.
	if len(budgeted) == 0 {
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
			return h.heartbeat.Ping(rc)
		}, restate.WithName("send heartbeat")); err != nil {
			return nil, fmt.Errorf("send heartbeat: %w", err)
		}
		logger.Info("deploy spend check complete: no budgeted workspaces",
			"billing_period", period,
		)
		return &hydrav1.RunDeploySpendCheckResponse{}, nil
	}

	// Sort by id so the fan-out order is stable across replays: each dispatch
	// is journaled by position, so a different iteration order on replay would
	// dispatch a different request at the same journal index and diverge.
	sort.Slice(budgeted, func(i, j int) bool { return budgeted[i].ID < budgeted[j].ID })

	budgetedIDs := make([]string, 0, len(budgeted))
	for _, ws := range budgeted {
		budgetedIDs = append(budgetedIDs, ws.ID)
	}
	values, err := h.priceUsage(ctx, period, budgetedIDs)
	if err != nil {
		return nil, err
	}

	var dispatched, skippedBelowThreshold int32
	type checkFuture = restate.ResponseFuture[*hydrav1.CheckWorkspaceSpendResponse]
	checkFutures := make([]checkFuture, 0, len(budgeted))
	checkWorkspaceIDs := make([]string, 0, len(budgeted))
	for _, ws := range budgeted {
		if !ws.DeploySpendBudgetCents.Valid {
			continue // query filters these out; guard against a future query change
		}

		// All spend math is integer micro-cents: the pricing quantized once in
		// PriceMicroCents, and cents-denominated columns scale exactly. The cap
		// is on gross total metered spend, credits included: a $100 budget stops
		// at $100 of usage, not $100 on top of the plan's included credit.
		gross := deploybilling.PriceMicroCents(values[ws.ID])
		// Below the lowest alert threshold nothing can happen in the check;
		// skip the invocation entirely. At or past it, the check owns the
		// decision: its high-water mark decides whether an email is actually
		// due, so re-dispatching an already-alerted workspace is a cheap no-op.
		if crossedThreshold(gross, ws.DeploySpendBudgetCents.Int64*deploybilling.MicroCentsPerCent) == 0 {
			skippedBelowThreshold++
			continue
		}

		// Dispatch the check, then await the outcomes below. The child still
		// runs in its own VO with independent retries and owns state and the
		// email send; awaiting only surfaces success or failure here, so a
		// check that fails every tick withholds the heartbeat instead of
		// dying quietly as its own invocation.
		checkFutures = append(checkFutures, hydrav1.NewDeploySpendCheckServiceClient(ctx, ws.ID).
			CheckWorkspaceSpend().
			RequestFuture(&hydrav1.CheckWorkspaceSpendRequest{
				Period:          period,
				BudgetCents:     ws.DeploySpendBudgetCents.Int64,
				Stop:            ws.DeploySpendBudgetStop,
				OrgId:           ws.OrgID,
				WorkspaceName:   ws.Name,
				WorkspaceSlug:   ws.Slug,
				SpendMicroCents: gross,
			}))
		checkWorkspaceIDs = append(checkWorkspaceIDs, ws.ID)
		dispatched++
	}

	checkFailed := 0
	for i, fut := range checkFutures {
		if _, err := fut.Response(); err != nil {
			checkFailed++
			logger.Error("deploy spend check child failed",
				"billing_period", period,
				"workspace_id", checkWorkspaceIDs[i],
				"error", err,
			)
		}
	}

	logger.Info("deploy spend check complete",
		"billing_period", period,
		"workspaces_dispatched", dispatched,
		"workspaces_check_failed", checkFailed,
		"workspaces_skipped_below_threshold", skippedBelowThreshold,
	)

	// Withhold the heartbeat when any check failed, mirroring the billing
	// push orchestrator: a suspension or alert that cannot succeed must trip
	// the dead-man switch, not just log.
	if checkFailed == 0 {
		if err := restate.RunVoid(ctx, func(rc restate.RunContext) error {
			return h.heartbeat.Ping(rc)
		}, restate.WithName("send heartbeat")); err != nil {
			return nil, fmt.Errorf("send heartbeat: %w", err)
		}
	} else {
		logger.Error("deploy spend check withheld heartbeat after child failures",
			"billing_period", period,
			"workspaces_check_failed", checkFailed,
		)
	}

	return &hydrav1.RunDeploySpendCheckResponse{
		WorkspacesDispatched: dispatched,
	}, nil
}

// priceUsage reads the period's month-to-date usage for the budgeted
// workspaces in one grouped ClickHouse scan (two queries: instance meters and
// active keys, shared with the hourly push via FleetMeterValues) and returns
// the aggregated MeterValues keyed by workspace id. One grouped scan is how
// ClickHouse wants to be read; per-workspace point queries at fan-out scale
// are not, and scoping it to the budgeted set keeps the tight cadence from
// re-aggregating the whole fleet's month. The read is capped at the period's
// end so a stale invocation running after the roll cannot fold the next month
// into this period's decisions.
func (h *Handler) priceUsage(
	ctx restate.ObjectContext,
	period string,
	workspaceIDs []string,
) (map[string]billingmeter.MeterValues, error) {
	p, err := billingperiod.Parse(period)
	if err != nil {
		return nil, restate.TerminalError(fmt.Errorf("invalid billing period %q: %w", period, err))
	}
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}

	endMillis := now.UnixMilli()
	if periodEndMillis := p.End().UnixMilli(); endMillis > periodEndMillis {
		endMillis = periodEndMillis
	}

	return deploybilling.FleetMeterValues(ctx, h.usage, p, endMillis, workspaceIDs)
}
