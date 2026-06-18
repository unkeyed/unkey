package deployspendcheck

import (
	"fmt"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// CheckConfig holds the per-workspace check's dependencies.
type CheckConfig struct {
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
// scan and carried in the request), computes net-of-credit overage, and emails
// the workspace's admins on newly crossed budget thresholds. Fanned out to by
// the orchestrator, one invocation per actionable workspace, keyed by
// workspace id.
type CheckHandler struct {
	admins         workos.Resolver
	email          email.Sender
	billingBaseURL string
}

// NewCheckHandler constructs a CheckHandler.
func NewCheckHandler(cfg CheckConfig) (*CheckHandler, error) {
	return &CheckHandler{
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

// CheckWorkspaceSpend takes the workspace's priced month-to-date usage from
// the request, computes overage = max(0, gross - included credit), and emails
// the admins for each 50/75/100% threshold newly crossed this period: at most
// one email per tick (the highest crossed), at most once per threshold per
// period. It reads no usage itself: the orchestrator prices every workspace
// from one ClickHouse scan and only dispatches the ones near or over budget,
// so this handler is pure state transition plus email.
func (h *CheckHandler) CheckWorkspaceSpend(
	ctx restate.ObjectContext,
	req *hydrav1.CheckWorkspaceSpendRequest,
) (*hydrav1.CheckWorkspaceSpendResponse, error) {
	workspaceID := restate.Key(ctx)

	if req.GetBudgetCents() <= 0 {
		// A non-positive budget can't define a meaningful threshold; nothing to do.
		return &hydrav1.CheckWorkspaceSpendResponse{}, nil
	}

	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}

	// All spend math is integer micro-cents: the orchestrator quantized the
	// gross once at the pricing boundary, and cents-denominated fields scale
	// exactly.
	gross := req.GetGrossMicroCents()
	overage := gross - req.GetIncludedCreditCents()*deploybilling.MicroCentsPerCent
	if overage < 0 {
		overage = 0
	}
	budgetMicroCents := req.GetBudgetCents() * deploybilling.MicroCentsPerCent

	// crossed: highest threshold the overage reaches now. alerted: highest we've
	// already emailed this period. We email only when the overage has climbed to
	// a higher threshold than we've alerted, then raise the high-water mark.
	crossed := crossedThreshold(overage, budgetMicroCents)
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
			"overage_cents", overage/deploybilling.MicroCentsPerCent,
			"budget_cents", req.GetBudgetCents(),
			"gross_cents", gross/deploybilling.MicroCentsPerCent,
			"included_credit_cents", req.GetIncludedCreditCents(),
			"stop", req.GetStop(),
		)

		// Email the org admins. The send is journaled, so a retry of this
		// invocation does not re-send; the high-water mark is raised only after
		// the send succeeds, so a failure retries on the next tick rather than
		// being silently skipped.
		err = h.alert(ctx, budgetAlert{
			OrgID:             req.GetOrgId(),
			WorkspaceName:     req.GetWorkspaceName(),
			WorkspaceSlug:     req.GetWorkspaceSlug(),
			Threshold:         crossed,
			OverageMicroCents: overage,
			BudgetCents:       req.GetBudgetCents(),
			Year:              now.Year(),
		})
		if err != nil {
			return nil, err
		}

		restate.Set(ctx, stateKey, crossed)
		alerted = crossed
		sentAlert = true
	}

	return &hydrav1.CheckWorkspaceSpendResponse{
		GrossCents:       gross / deploybilling.MicroCentsPerCent,
		OverageCents:     overage / deploybilling.MicroCentsPerCent,
		ThresholdCrossed: alerted,
		Alerted:          sentAlert,
	}, nil
}
