package deployspendcheck

import (
	"fmt"
	"strconv"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/worker/cron/deploybilling"
)

// budgetAlertTemplate is the published Resend template alias for the Compute
// budget alert. It owns the subject and sender; this handler supplies only the
// recipients and variables.
const budgetAlertTemplate = "compute-budget-alert"

// budgetAlert is the data for one budget-threshold alert email.
type budgetAlert struct {
	OrgID         string
	WorkspaceName string
	WorkspaceSlug string
	// Threshold is the budget percentage crossed: 50, 75 or 100.
	Threshold int32
	// OverageCents is the net-of-credit spend so far (the "used" figure).
	OverageCents float64
	BudgetCents  int64
	Year         int
}

// alert resolves the workspace's org admins and emails them the budget alert.
// The WorkOS lookup and the send are each journaled, so a replay repeats
// neither. A workspace with no resolvable admins sends nothing.
func (h *CheckHandler) alert(ctx restate.ObjectContext, a budgetAlert) error {
	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		return h.admins.AdminEmails(rc, a.OrgID)
	}, restate.WithName("resolve org admins"))
	if err != nil {
		return fmt.Errorf("resolve org admins: %w", err)
	}
	if len(recipients) == 0 {
		logger.Warn("budget alert has no recipients",
			"org_id", a.OrgID,
			"workspace_name", a.WorkspaceName,
			"threshold", a.Threshold,
		)
		return nil
	}

	variables := map[string]string{
		"PERCENT":        strconv.Itoa(int(a.Threshold)),
		"USAGE":          deploybilling.FormatDollars(a.OverageCents),
		"BUDGET":         deploybilling.FormatDollars(float64(a.BudgetCents)),
		"WORKSPACE_NAME": a.WorkspaceName,
		"BILLING_URL":    fmt.Sprintf("%s/%s/settings/billing", h.billingBaseURL, a.WorkspaceSlug),
		"YEAR":           strconv.Itoa(a.Year),
	}
	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.email.Send(rc, email.Email{
			To:         recipients,
			TemplateID: budgetAlertTemplate,
			Variables:  variables,
			// From and Subject left empty: the published template owns both (its
			// subject interpolates PERCENT), so the sender uses its configured
			// default From and the template's subject.
			From:    "",
			Subject: "",
		})
	}, restate.WithName("send budget alert"))
}
