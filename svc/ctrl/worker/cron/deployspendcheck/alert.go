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
// budget threshold warning (50/75/100%). budgetStoppedTemplate is the alias for
// the "compute stopped" email sent when the spend cap actually suspends compute.
// Each template owns its subject and sender; this handler supplies only the
// recipients and variables. Both aliases must exist in Resend.
const (
	budgetAlertTemplate   = "compute-budget-alert"
	budgetStoppedTemplate = "compute-budget-stopped"
)

// budgetAlert is the data for one budget-threshold alert email.
type budgetAlert struct {
	WorkspaceID   string
	Period        string
	OrgID         string
	WorkspaceName string
	WorkspaceSlug string
	// Threshold is the budget percentage crossed: 50, 75 or 100.
	Threshold int32
	// SpendMicroCents is the gross metered spend so far (the "used" figure),
	// in integer micro-cents. The cap is on total spend, credits included.
	SpendMicroCents int64
	BudgetCents     int64
	Year            int
}

// alert emails the org admins the budget threshold warning for a newly crossed
// 50/75/100% level.
func (h *CheckHandler) alert(ctx restate.ObjectContext, a budgetAlert) error {
	return h.sendToAdmins(ctx, a, budgetAlertTemplate, map[string]string{
		"PERCENT":        strconv.Itoa(int(a.Threshold)),
		"USAGE":          deploybilling.FormatDollars(a.SpendMicroCents),
		"BUDGET":         deploybilling.FormatDollars(a.BudgetCents * deploybilling.MicroCentsPerCent),
		"WORKSPACE_NAME": a.WorkspaceName,
		"BILLING_URL":    fmt.Sprintf("%s/%s/settings/billing", h.billingBaseURL, a.WorkspaceSlug),
		"YEAR":           strconv.Itoa(a.Year),
	})
}

// stoppedAlert emails the org admins that the spend cap has stopped their
// Compute. It replaces the 100% threshold warning when stopping is enabled: the
// action, not the warning, is what they need to see.
func (h *CheckHandler) stoppedAlert(ctx restate.ObjectContext, a budgetAlert) error {
	return h.sendToAdmins(ctx, a, budgetStoppedTemplate, map[string]string{
		"USAGE":          deploybilling.FormatDollars(a.SpendMicroCents),
		"BUDGET":         deploybilling.FormatDollars(a.BudgetCents * deploybilling.MicroCentsPerCent),
		"WORKSPACE_NAME": a.WorkspaceName,
		"BILLING_URL":    fmt.Sprintf("%s/%s/settings/billing", h.billingBaseURL, a.WorkspaceSlug),
		"YEAR":           strconv.Itoa(a.Year),
	})
}

// sendToAdmins resolves the workspace's org admins and sends them one template
// email. The WorkOS lookup and the send are each journaled, so a replay repeats
// neither. A workspace with no resolvable admins sends nothing.
func (h *CheckHandler) sendToAdmins(ctx restate.ObjectContext, a budgetAlert, templateID string, variables map[string]string) error {
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
			"template", templateID,
		)
		return nil
	}

	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		msg := email.Email{
			To:             recipients,
			TemplateID:     templateID,
			Variables:      variables,
			From:           "",
			Subject:        "",
			IdempotencyKey: "",
		}
		if templateID == budgetAlertTemplate {
			msg.IdempotencyKey = budgetAlertIdempotencyKey(a.WorkspaceID, a.Period, a.Threshold)
		}
		return h.email.Send(rc, msg)
	}, restate.WithName("send budget alert"))
}

func budgetAlertIdempotencyKey(workspaceID, period string, threshold int32) string {
	return fmt.Sprintf("budget-alert/%s/%s/%d", workspaceID, period, threshold)
}
