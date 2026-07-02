package deployspendcheck

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// CheckConfig holds the per-workspace check's dependencies.
type CheckConfig struct {
	// Usage queries month-to-date Deploy usage from ClickHouse. May be nil
	// (ClickHouse not configured), making CheckWorkspaceSpend a no-op.
	Usage deploybilling.UsageReader
	// Admins resolves the org's admin emails for the alert. Use workos.New("")
	// for a noop that resolves no recipients.
	Admins workos.Resolver
	// Email sends the alert. Use email.NewNoop() to log instead of sending.
	Email email.Sender
	// BillingBaseURL is the dashboard origin for the alert's billing link, e.g.
	// "https://app.unkey.com". The workspace slug is appended.
	BillingBaseURL string
}

// CheckHandler implements DeploySpendCheckService: it prices one workspace's
// month-to-date Deploy usage, computes net-of-credit overage, and emails the
// workspace's admins on newly crossed budget thresholds. Fanned out to by the
// orchestrator, one invocation per workspace, keyed by workspace id.
type CheckHandler struct {
	usage          deploybilling.UsageReader
	admins         workos.Resolver
	email          email.Sender
	billingBaseURL string
}

// NewCheckHandler constructs a CheckHandler.
func NewCheckHandler(cfg CheckConfig) (*CheckHandler, error) {
	return &CheckHandler{
		usage:          cfg.Usage,
		admins:         cfg.Admins,
		email:          cfg.Email,
		billingBaseURL: cfg.BillingBaseURL,
	}, nil
}

// alertHighWaterKey is the VO state key for the highest budget threshold already
// alerted, scoped to a billing period. Scoping by period means the zero value is
// "nothing alerted yet this period", so a new month starts clean with no reset
// bookkeeping.
func alertHighWaterKey(period string) string {
	return "spend_alert_high_water:" + period
}

// CheckWorkspaceSpend prices the workspace's month-to-date Deploy usage from
// ClickHouse, computes overage = max(0, gross - included credit), and emails the
// admins for each 50/75/100% threshold newly crossed this period: at most one
// email per tick (the highest crossed), at most once per threshold per period.
func (h *CheckHandler) CheckWorkspaceSpend(
	ctx restate.ObjectContext,
	req *hydrav1.CheckWorkspaceSpendRequest,
) (*hydrav1.CheckWorkspaceSpendResponse, error) {
	workspaceID := restate.Key(ctx)

	if h.usage == nil {
		logger.Info("deploy spend check disabled (no usage reader configured)",
			"workspace_id", workspaceID,
		)
		return &hydrav1.CheckWorkspaceSpendResponse{}, nil
	}
	if req.GetBudgetCents() <= 0 {
		// A non-positive budget can't define a meaningful threshold; nothing to do.
		return &hydrav1.CheckWorkspaceSpendResponse{}, nil
	}

	p, err := billingperiod.Parse(req.GetPeriod())
	if err != nil {
		return nil, fmt.Errorf("invalid billing period %q: %w", req.GetPeriod(), err)
	}
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}

	startMillis := p.Start().UnixMilli()
	// Cap the read at the period's end: a stale invocation running after the
	// period rolled must price only its own period, not fold the next month's
	// usage into this period's alert and enforcement decision.
	endMillis := now.UnixMilli()
	if periodEndMillis := p.End().UnixMilli(); endMillis > periodEndMillis {
		endMillis = periodEndMillis
	}
	instanceRows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.InstanceMeterUsage, error) {
		return h.usage.GetInstanceMeterUsage(rc, clickhouse.GetInstanceMeterUsageRequest{
			WorkspaceID: workspaceID,
			Start:       startMillis,
			End:         endMillis,
		})
	}, restate.WithName("get instance usage"))
	if err != nil {
		return nil, fmt.Errorf("get instance usage: %w", err)
	}

	keyRows, err := restate.Run(ctx, func(rc restate.RunContext) ([]clickhouse.ActiveKeysUsage, error) {
		return h.usage.GetActiveKeysUsage(rc, clickhouse.GetActiveKeysUsageRequest{
			WorkspaceID: workspaceID,
			Month:       startMillis,
		})
	}, restate.WithName("get active keys usage"))
	if err != nil {
		return nil, fmt.Errorf("get active keys usage: %w", err)
	}

	// Price the exact same MeterValues the hourly push reports, then net out the
	// period's included credit. The maps are keyed by workspace id; this VO only
	// queried its own, so the single entry (if any) is this workspace's.
	values := deploybilling.AggregateUsage(instanceRows)
	deploybilling.MergeActiveKeys(values, keyRows)
	gross := deploybilling.PriceCents(values[workspaceID])
	overage := gross - float64(req.GetIncludedCreditCents())
	if overage < 0 {
		overage = 0
	}

	// crossed: highest threshold the overage reaches now. alerted: highest we've
	// already emailed this period. We email only when the overage has climbed to
	// a higher threshold than we've alerted, then raise the high-water mark.
	crossed := crossedThreshold(overage, req.GetBudgetCents())
	stateKey := alertHighWaterKey(req.GetPeriod())
	alerted, err := restate.Get[int32](ctx, stateKey)
	if err != nil {
		return nil, fmt.Errorf("get alert high-water: %w", err)
	}

	sentAlert := false
	if crossed > alerted {
		logger.Info("deploy spend threshold crossed",
			"workspace_id", workspaceID,
			"billing_period", req.GetPeriod(),
			"threshold", crossed,
			"overage_cents", int64(overage),
			"budget_cents", req.GetBudgetCents(),
			"gross_cents", int64(gross),
			"included_credit_cents", req.GetIncludedCreditCents(),
			"stop", req.GetStop(),
		)

		// Email the org admins. The send is journaled, so a retry of this
		// invocation does not re-send; the high-water mark is raised only after
		// the send succeeds, so a failure retries on the next tick rather than
		// being silently skipped.
		err = h.alert(ctx, budgetAlert{
			OrgID:         req.GetOrgId(),
			WorkspaceName: req.GetWorkspaceName(),
			WorkspaceSlug: req.GetWorkspaceSlug(),
			Threshold:     crossed,
			OverageCents:  overage,
			BudgetCents:   req.GetBudgetCents(),
			Year:          now.Year(),
		})
		if err != nil {
			return nil, err
		}

		restate.Set(ctx, stateKey, crossed)
		alerted = crossed
		sentAlert = true
	}

	return &hydrav1.CheckWorkspaceSpendResponse{
		GrossCents:       int64(gross),
		OverageCents:     int64(overage),
		ThresholdCrossed: alerted,
		Alerted:          sentAlert,
	}, nil
}
