package deployspendcheck

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// CheckConfig holds the per-workspace check's dependencies.
type CheckConfig struct {
	// DB is the primary application database, used to persist the
	// deploy_spend_suspended column on the suspend/resume transition. Must not
	// be nil.
	DB db.Database
	// Admins resolves the org's admin emails for the alert. Use workos.NewNoop()
	// for a noop that resolves no recipients.
	Admins workos.Resolver
	// Email sends the alert. Use email.NewNoop() to log instead of sending.
	Email email.Sender
	// BillingBaseURL is the dashboard origin for the alert's billing link, e.g.
	// "https://app.unkey.com". The workspace slug is appended.
	BillingBaseURL string
}

// CheckHandler implements DeploySpendCheckService: it takes one workspace's
// priced month-to-date usage (computed by the orchestrator's single ClickHouse
// scan and carried in the request), compares gross total spend to the budget,
// and emails the workspace's admins on newly crossed budget thresholds. Fanned
// out to by the orchestrator, one invocation per actionable workspace, keyed by
// workspace id.
type CheckHandler struct {
	db             db.Database
	admins         workos.Resolver
	email          email.Sender
	billingBaseURL string
}

// NewCheckHandler constructs a CheckHandler. Dependencies are asserted at
// construction so a wiring mistake fails at boot, not on the first threshold
// crossing mid-invocation.
func NewCheckHandler(cfg CheckConfig) (*CheckHandler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Admins, "Admins must not be nil; use workos.NewNoop()"),
		assert.NotNil(cfg.Email, "Email must not be nil; use email.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &CheckHandler{
		db:             cfg.DB,
		admins:         cfg.Admins,
		email:          cfg.Email,
		billingBaseURL: cfg.BillingBaseURL,
	}, nil
}

// alertHighWaterKey is the VO state key for the gross spend (micro-cents) at
// which we last alerted, scoped to a billing period. Storing spend instead of a
// threshold keeps alerting correct across budget edits: the effective threshold
// is recomputed against the current budget each tick, and spend only grows
// within a period, so a budget edit alone can never re-alert. Zero value means
// nothing alerted this period.
func alertHighWaterKey(period string) string {
	return "spend_alert_high_water:" + period
}

// suspendGenerationKey counts suspension transitions this period. It feeds the
// stopped email's idempotency key: retries of one suspension dedupe, a later
// suspension sends.
func suspendGenerationKey(period string) string {
	return "spend_suspend_generation:" + period
}

// setSuspended persists the workspace's deploy_spend_suspended column on a
// suspend/resume transition. The column (not VO state) is authoritative: it
// lets the orchestrator keep dispatching a suspended workspace after its budget
// is removed, and the dashboard render a "suspended by spend cap" state.
func (h *CheckHandler) setSuspended(ctx restate.ObjectContext, workspaceID string, suspended bool) error {
	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.db.SetWorkspaceDeploySpendSuspended(rc, db.SetWorkspaceDeploySpendSuspendedParams{
			Suspended: suspended,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			ID:        workspaceID,
		})
	}, restate.WithName("set spend-suspended"))
}

// CheckWorkspaceSpend takes the workspace's priced month-to-date usage from
// the request and emails the admins for each 50/75/100% threshold of gross
// total spend newly crossed this period: at most one email per tick (the
// highest crossed), at most once per threshold per period, and suspends or
// resumes compute on the enforcement transitions. The cap is on total metered
// spend, credits included. It reads no usage itself: the orchestrator prices
// every workspace from one ClickHouse scan and only dispatches the ones near or
// over budget (or currently suspended), so this handler is pure state
// transition plus email.
func (h *CheckHandler) CheckWorkspaceSpend(
	ctx restate.ObjectContext,
	req *hydrav1.CheckWorkspaceSpendRequest,
) (*hydrav1.CheckWorkspaceSpendResponse, error) {
	workspaceID := restate.Key(ctx)

	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}

	// Enforce nothing outside the period this tick was priced for: a check
	// delayed across the month roll would otherwise act on the new month with
	// old spend. Journaled Now() so replays agree.
	reqPeriod, err := billingperiod.Parse(req.GetPeriod())
	if err != nil {
		return nil, restate.TerminalError(fmt.Errorf("invalid period %q: %w", req.GetPeriod(), err))
	}
	if reqPeriod != billingperiod.From(now) {
		logger.Info("deploy spend cap: skipping stale-period tick",
			"workspace_id", workspaceID,
			"request_period", req.GetPeriod(),
			"current_period", billingperiod.From(now).Key(),
		)
		return &hydrav1.CheckWorkspaceSpendResponse{}, nil
	}

	// Clear the previous period's keys so workspaces don't accrue an immortal
	// key pair per month. Clear is idempotent, so no first-tick bookkeeping.
	prev := reqPeriod.Prev()
	restate.Clear(ctx, alertHighWaterKey(prev.Key()))
	restate.Clear(ctx, suspendGenerationKey(prev.Key()))

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

	// All spend math is integer micro-cents: the orchestrator quantized the
	// gross once at the pricing boundary, and cents-denominated fields scale
	// exactly. The cap is on gross total metered spend, credits included.
	gross := req.GetSpendMicroCents()
	budgetMicroCents := req.GetBudgetCents() * deploybilling.MicroCentsPerCent

	// The spend-suspended flag comes from the workspace's deploy_spend_suspended
	// column, passed in by the orchestrator. It gates both the email (stopping
	// replaces the 100% warning with a "stopped" email) and the enforcement
	// action below.
	suspended := req.GetCurrentlySuspended()

	// willSuspend is true when this tick will stop compute: the workspace opted
	// in, gross spend reached the budget, and it is not already suspended.
	// crossedThreshold returns 100 exactly when gross >= budget, so willSuspend
	// implies crossed == 100.
	willSuspend := req.GetStop() && !suspended && gross >= budgetMicroCents

	// crossed: highest threshold gross reaches now. alerted: recomputed from the
	// spend we last alerted at, against the current budget, so raising the
	// budget re-arms higher thresholds and lowering it suppresses re-sends. Only
	// real spend growth into a new bucket emails; budget churn alone never does.
	crossed := crossedThreshold(gross, budgetMicroCents)
	stateKey := alertHighWaterKey(req.GetPeriod())
	alertedSpend, err := restate.Get[int64](ctx, stateKey)
	if err != nil {
		return nil, fmt.Errorf("get alert high-water: %w", err)
	}
	alerted := crossedThreshold(alertedSpend, budgetMicroCents)

	// No threshold warnings while suspended: the stopped email owns that
	// messaging, and a suspend tick that died before recording the mark must not
	// yield a stale 100% warning on the next tick.
	sentAlert := false
	if crossed > alerted && !willSuspend && !suspended {
		logger.Info("deploy spend threshold crossed",
			"workspace_id", workspaceID,
			"billing_period", req.GetPeriod(),
			"threshold", crossed,
			"gross_cents", gross/deploybilling.MicroCentsPerCent,
			"budget_cents", req.GetBudgetCents(),
			"stop", req.GetStop(),
		)

		err = h.alert(ctx, budgetAlert{
			WorkspaceID:     workspaceID,
			Period:          req.GetPeriod(),
			OrgID:           req.GetOrgId(),
			WorkspaceName:   req.GetWorkspaceName(),
			WorkspaceSlug:   req.GetWorkspaceSlug(),
			Threshold:       crossed,
			SpendMicroCents: gross,
			BudgetCents:     req.GetBudgetCents(),
			Year:            now.Year(),
			// Only the stopped email keys on Suspension.
			Suspension: 0,
		})
		if err != nil {
			return nil, err
		}
		sentAlert = true
		restate.Set(ctx, stateKey, gross)
		alerted = crossed
	}

	// Enforcement: suspend compute when gross spend reaches the budget and the
	// workspace opted into stopping; resume when spend falls back under budget,
	// stopping is turned off, or the period rolls.
	switch {
	case willSuspend:
		logger.Info("deploy spend threshold crossed",
			"workspace_id", workspaceID,
			"billing_period", req.GetPeriod(),
			"threshold", int32(100),
			"gross_cents", gross/deploybilling.MicroCentsPerCent,
			"budget_cents", req.GetBudgetCents(),
			"stop", req.GetStop(),
		)

		// Persist the flag before teardown so new deployments are blocked
		// immediately; then stop running compute.
		if err := h.setSuspended(ctx, workspaceID, true); err != nil {
			return nil, fmt.Errorf("set spend-suspended: %w", err)
		}
		suspended = true

		hydrav1.NewDeployTeardownServiceClient(ctx, workspaceID).
			Teardown().
			Send(&hydrav1.TeardownRequest{Mode: hydrav1.TeardownMode_TEARDOWN_MODE_SUSPEND})

		logger.Info("deploy spend cap: suspended compute",
			"workspace_id", workspaceID, "billing_period", req.GetPeriod(),
			"gross_cents", gross/deploybilling.MicroCentsPerCent, "budget_cents", req.GetBudgetCents())

		// Journaled read+write: a replay reuses this generation and dedupes at
		// Resend; a later suspension reads a higher value and sends.
		genKey := suspendGenerationKey(req.GetPeriod())
		generation, err := restate.Get[int32](ctx, genKey)
		if err != nil {
			return nil, fmt.Errorf("get suspend generation: %w", err)
		}
		generation++
		restate.Set(ctx, genKey, generation)

		err = h.stoppedAlert(ctx, budgetAlert{
			WorkspaceID:     workspaceID,
			Period:          req.GetPeriod(),
			OrgID:           req.GetOrgId(),
			WorkspaceName:   req.GetWorkspaceName(),
			WorkspaceSlug:   req.GetWorkspaceSlug(),
			Threshold:       100,
			SpendMicroCents: gross,
			BudgetCents:     req.GetBudgetCents(),
			Year:            now.Year(),
			Suspension:      generation,
		})
		if err != nil {
			return nil, err
		}
		restate.Set(ctx, stateKey, gross)
		alerted = crossed

	case suspended && req.GetStop() && gross >= budgetMicroCents:
		// The flag can say stopped while compute still runs (kill before the
		// teardown send, drain grace expired, create racing the gate). Re-send
		// Teardown: it early-returns when nothing runs and merges into the
		// recorded suspension state so previously stopped apps stay resumable.
		// No email, no flag change: enforcement catching up, not a transition.
		hydrav1.NewDeployTeardownServiceClient(ctx, workspaceID).
			Teardown().
			Send(&hydrav1.TeardownRequest{Mode: hydrav1.TeardownMode_TEARDOWN_MODE_SUSPEND})
		logger.Info("deploy spend cap: re-enforced suspension",
			"workspace_id", workspaceID, "billing_period", req.GetPeriod(),
			"gross_cents", gross/deploybilling.MicroCentsPerCent, "budget_cents", req.GetBudgetCents())

	case suspended && (!req.GetStop() || gross < budgetMicroCents):
		// Budget raised above gross spend, period reset, or stopping turned
		// off while suspended. Bring compute back.
		hydrav1.NewDeployTeardownServiceClient(ctx, workspaceID).
			Resume().
			Send(&hydrav1.ResumeRequest{})
		if err := h.setSuspended(ctx, workspaceID, false); err != nil {
			return nil, fmt.Errorf("clear spend-suspended: %w", err)
		}
		suspended = false
		logger.Info("deploy spend cap: resumed compute",
			"workspace_id", workspaceID, "billing_period", req.GetPeriod(),
			"gross_cents", gross/deploybilling.MicroCentsPerCent, "budget_cents", req.GetBudgetCents())
	}

	return &hydrav1.CheckWorkspaceSpendResponse{
		SpendCents:       gross / deploybilling.MicroCentsPerCent,
		ThresholdCrossed: alerted,
		Alerted:          sentAlert,
		Suspended:        suspended,
	}, nil
}
