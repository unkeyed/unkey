package deployspendcheck

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// CheckConfig holds the per-workspace check's dependencies.
type CheckConfig struct {
	// DB is the primary application database, used to persist the
	// deploy_spend_suspended column on the suspend/resume transition. Must not
	// be nil.
	DB db.Database
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
	db             db.Database
	usage          deploybilling.UsageReader
	admins         workos.Resolver
	email          email.Sender
	billingBaseURL string
}

// NewCheckHandler constructs a CheckHandler.
func NewCheckHandler(cfg CheckConfig) (*CheckHandler, error) {
	if err := assert.NotNil(cfg.DB, "DB must not be nil"); err != nil {
		return nil, err
	}
	return &CheckHandler{
		db:             cfg.DB,
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

// setSuspended persists the workspace's deploy_spend_suspended column on a
// suspend/resume transition. The column (not VO state) is authoritative: it
// lets the orchestrator keep dispatching a suspended workspace after its budget
// is removed, and the dashboard render a "suspended by spend cap" state.
func (h *CheckHandler) setSuspended(ctx restate.ObjectContext, workspaceID string, suspended bool) error {
	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return db.Query.SetWorkspaceDeploySpendSuspended(rc, h.db.RW(), db.SetWorkspaceDeploySpendSuspendedParams{
			Suspended: suspended,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:        workspaceID,
		})
	}, restate.WithName("set spend-suspended"))
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
		if req.GetCurrentlySuspended() {
			// Budget was removed while suspended: nothing caps spend anymore, so
			// bring compute back and clear the flag.
			hydrav1.NewDeployTeardownServiceClient(ctx, workspaceID).
				Resume().
				Send(&hydrav1.ResumeRequest{})
			if err := h.setSuspended(ctx, workspaceID, false); err != nil {
				return nil, fmt.Errorf("clear spend-suspended: %w", err)
			}
			logger.Info("deploy spend cap: resumed after budget removed",
				"workspace_id", workspaceID)
		}
		// A non-positive budget can't define a meaningful threshold; nothing else
		// to do.
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

	// The spend-suspended flag comes from the workspace's deploy_spend_suspended
	// column, passed in by the orchestrator. It gates both the email (stopping
	// replaces the 100% warning with a "stopped" email) and the enforcement
	// action below.
	suspended := req.GetCurrentlySuspended()

	// willSuspend is true when this tick will stop compute: the workspace opted
	// in, the net-of-credit overage reached the budget, and it is not already
	// suspended. crossedThreshold returns 100 exactly when overage >= budget, so
	// willSuspend implies crossed == 100.
	willSuspend := req.GetStop() && !suspended && overage >= float64(req.GetBudgetCents())

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

		// When this tick is going to stop compute, the "stopped" email in the
		// enforcement block below replaces the 100% warning, so skip the warning
		// to avoid two emails for the same event. Either way advance the
		// high-water mark so we never re-warn this period. The send is journaled,
		// so a retry does not re-send; the mark is raised only after a successful
		// send, so a failure retries on the next tick rather than being skipped.
		if !willSuspend {
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
			sentAlert = true
		}

		restate.Set(ctx, stateKey, crossed)
		alerted = crossed
	}

	// Enforcement: suspend compute when the net-of-credit overage reaches the
	// budget and the workspace opted into stopping; resume it once overage falls
	// back under budget (a mid-period budget raise or the period-boundary reset).
	// The flag makes both transitions idempotent across ticks.
	switch {
	case willSuspend:
		// Net-of-credit overage reached the budget and the workspace opted into
		// stopping. Suspend compute (resumable). Idempotent via the flag.
		hydrav1.NewDeployTeardownServiceClient(ctx, workspaceID).
			Teardown().
			Send(&hydrav1.TeardownRequest{Mode: hydrav1.TeardownMode_TEARDOWN_MODE_SUSPEND})

		// Persist the flag immediately after dispatching the stop, before any
		// step that can fail (the email below talks to WorkOS and Resend).
		// Nothing fallible may sit between the destructive action and the
		// record of it: compute stopped with the flag still false shows an
		// unsuspended dashboard and hides the workspace from the resume path
		// until this invocation's retries get past the failing step.
		if err := h.setSuspended(ctx, workspaceID, true); err != nil {
			return nil, fmt.Errorf("set spend-suspended: %w", err)
		}
		suspended = true
		logger.Info("deploy spend cap: suspended compute",
			"workspace_id", workspaceID, "billing_period", req.GetPeriod(),
			"overage_cents", int64(overage), "budget_cents", req.GetBudgetCents())

		// Tell the admins their Compute was stopped. This is the action email,
		// standing in for the 100% warning; it is sent on the suspend transition
		// so it fires even if 100% was already warned before stopping was enabled.
		// A failure retries the invocation until the send succeeds (the suspend
		// steps above are journaled and replay as no-ops).
		err = h.stoppedAlert(ctx, budgetAlert{
			OrgID:         req.GetOrgId(),
			WorkspaceName: req.GetWorkspaceName(),
			WorkspaceSlug: req.GetWorkspaceSlug(),
			Threshold:     100,
			OverageCents:  overage,
			BudgetCents:   req.GetBudgetCents(),
			Year:          now.Year(),
		})
		if err != nil {
			return nil, err
		}

	case suspended && overage < float64(req.GetBudgetCents()):
		// Either the budget was raised above the frozen overage, or the period
		// rolled and usage reset. Bring compute back.
		hydrav1.NewDeployTeardownServiceClient(ctx, workspaceID).
			Resume().
			Send(&hydrav1.ResumeRequest{})
		if err := h.setSuspended(ctx, workspaceID, false); err != nil {
			return nil, fmt.Errorf("clear spend-suspended: %w", err)
		}
		suspended = false
		logger.Info("deploy spend cap: resumed compute",
			"workspace_id", workspaceID, "billing_period", req.GetPeriod(),
			"overage_cents", int64(overage), "budget_cents", req.GetBudgetCents())
	}

	return &hydrav1.CheckWorkspaceSpendResponse{
		GrossCents:       int64(gross),
		OverageCents:     int64(overage),
		ThresholdCrossed: alerted,
		Alerted:          sentAlert,
		Suspended:        suspended,
	}, nil
}
