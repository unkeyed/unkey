package deployspendcheck

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
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

	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
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

	// crossed: highest threshold gross spend reaches now. alerted: highest we've
	// already emailed this period. We email only when spend has climbed to a
	// higher threshold than we've alerted, then raise the high-water mark.
	crossed := crossedThreshold(gross, budgetMicroCents)
	stateKey := alertHighWaterKey(req.GetPeriod())
	alerted, err := restate.Get[int32](ctx, stateKey)
	if err != nil {
		return nil, fmt.Errorf("get alert high-water: %w", err)
	}

	sentAlert := false
	if crossed > alerted && !willSuspend {
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
		})
		if err != nil {
			return nil, err
		}
		sentAlert = true
		restate.Set(ctx, stateKey, crossed)
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
		})
		if err != nil {
			return nil, err
		}
		restate.Set(ctx, stateKey, crossed)
		alerted = crossed

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
