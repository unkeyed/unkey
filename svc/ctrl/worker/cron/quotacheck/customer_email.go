package quotacheck

import (
	"fmt"
	"strconv"
	"strings"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	usageExceededTemplate               = "api-usage-exceeded"
	usageRatelimitFollowUpTemplate      = "api-usage-ratelimit-follow-up"
	customerEmailMaxAttempts       uint = 5
)

func (h *Handler) sendCustomerEmail(
	ctx restate.ObjectContext,
	period string,
	year int,
	attempt int64,
	e exceededWorkspace,
) (bool, error) {
	if !h.customerEmailEnabled {
		return false, nil
	}

	templateID, ok := customerEmailTemplate(e.Workspace.Tier.String, attempt)
	if !ok {
		return false, nil
	}

	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		return h.admins.AdminEmails(rc, e.Workspace.OrgID)
	},
		restate.WithName("resolve quota email recipients "+e.Workspace.ID),
		restate.WithMaxRetryAttempts(customerEmailMaxAttempts),
	)
	if err != nil {
		return false, fmt.Errorf("resolve org admins: %w", err)
	}
	if len(recipients) == 0 {
		logger.Warn("quota email has no recipients",
			"org_id", e.Workspace.OrgID,
			"workspace_id", e.Workspace.ID,
			"template", templateID,
		)
		return false, nil
	}

	printer := message.NewPrinter(language.English)
	dashboardURL := strings.TrimRight(h.billingBaseURL, "/")
	variables := map[string]string{
		"WORKSPACE_NAME": e.Workspace.Name,
		"USED":           printer.Sprint(number.Decimal(e.Used)),
		"LIMIT":          printer.Sprint(number.Decimal(e.Workspace.RequestsPerMonth.Int64)),
		"BILLING_URL":    fmt.Sprintf("%s/%s/settings/billing", dashboardURL, e.Workspace.Slug),
		"YEAR":           strconv.Itoa(year),
	}

	err = restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.email.Send(rc, email.Email{
			To:             recipients,
			TemplateID:     templateID,
			Variables:      variables,
			From:           "",
			Subject:        "",
			IdempotencyKey: customerEmailIdempotencyKey(e.Workspace.ID, period, attempt),
		})
	},
		restate.WithName("send quota customer email "+e.Workspace.ID),
		restate.WithMaxRetryAttempts(customerEmailMaxAttempts),
	)
	if err != nil {
		return false, fmt.Errorf("send email: %w", err)
	}

	return true, nil
}

func customerEmailTemplate(tier string, attempt int64) (string, bool) {
	if tier != "Free" {
		return "", false
	}
	switch attempt {
	case 0:
		return usageExceededTemplate, true
	case 1:
		return usageRatelimitFollowUpTemplate, true
	default:
		return "", false
	}
}

func customerEmailIdempotencyKey(workspaceID, period string, attempt int64) string {
	return fmt.Sprintf("quota-alert/%s/%s/%d", workspaceID, period, attempt+1)
}
