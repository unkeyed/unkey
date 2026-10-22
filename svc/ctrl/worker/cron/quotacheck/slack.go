package quotacheck

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

func quotaSlackMessage(e exceededWorkspace, billingBaseURL string) slack.Payload {
	title := "Monthly request quota exceeded"
	if e.IsFollowUp {
		title = "Monthly request quota still exceeded"
	}
	printer := message.NewPrinter(language.English)
	usage := fmt.Sprintf("%s / %s requests",
		printer.Sprint(number.Decimal(e.Used)),
		printer.Sprint(number.Decimal(e.Workspace.RequestsPerMonth.Int64)))
	detail := "*" + usage + "*"
	if e.Workspace.RequestsPerMonth.Int64 > 0 {
		over := fmt.Sprintf("%.1f%% over quota", (float64(e.Used)/float64(e.Workspace.RequestsPerMonth.Int64)-1)*100)
		usage += ", " + over
		detail += " · " + over
	}
	reminder := ""
	if e.IsFollowUp {
		detail += "\nWeekly reminder. Review usage and the customer's plan."
		reminder = "Weekly reminder. "
	}
	return slack.Payload{
		Text: fmt.Sprintf("%s. %s: %s. %sReview workspace billing.", title, slackName(e.Workspace.Name), usage, reminder),
		Blocks: append([]slack.Block{
			slack.NewHeaderBlock(title),
			slack.NewMarkdownBlock(detail),
		}, workspaceSlackDetails(e.Workspace, billingBaseURL)...),
	}
}

func throttledSlackMessage(ws db.GetWorkspacesForQuotaCheckByIDsRow, billingBaseURL string) slack.Payload {
	return slack.Payload{
		Text: fmt.Sprintf("%s: free-tier API requests are now limited to at most %d per minute. Review workspace billing.", slackName(ws.Name), throttleRequestsPerMinute),
		Blocks: append([]slack.Block{
			slack.NewHeaderBlock("Free-tier API requests rate limited"),
			slack.NewMarkdownBlock(fmt.Sprintf("Free-tier monthly quota exceeded.\nAPI requests are now limited to *at most %d per minute*.", throttleRequestsPerMinute)),
		}, workspaceSlackDetails(ws, billingBaseURL)...),
	}
}

func workspaceSlackDetails(ws db.GetWorkspacesForQuotaCheckByIDsRow, billingBaseURL string) []slack.Block {
	fields := []slack.Field{{Type: "mrkdwn", Text: "*Workspace*\n" + slackName(ws.Name), Verbatim: true}}
	if ws.Tier.Valid && ws.Tier.String != "" {
		fields = append(fields, slack.Field{Type: "mrkdwn", Text: "*Tier*\n" + slackName(ws.Tier.String), Verbatim: true})
	}
	blocks := []slack.Block{slack.NewSectionBlock(fields...)}
	var links []string
	base, err := url.Parse(billingBaseURL)
	if err == nil && base.Host != "" && (base.Scheme == "https" || base.Scheme == "http") && base.User == nil && base.RawQuery == "" && base.Fragment == "" && ws.Slug != "" {
		billingURL := strings.TrimRight(base.String(), "/") + "/" + url.PathEscape(ws.Slug) + "/settings/billing"
		links = append(links, "<"+escapeSlackText(billingURL)+"|Review workspace billing>")
	}
	if ws.StripeCustomerID.Valid && ws.StripeCustomerID.String != "" {
		links = append(links, "<https://dashboard.stripe.com/customers/"+url.PathEscape(ws.StripeCustomerID.String)+"|View Stripe customer>")
	}
	if len(links) > 0 {
		blocks = append(blocks, slack.NewMarkdownBlock(strings.Join(links, " · ")))
	}
	blocks = append(blocks,
		slackIDContext("Workspace ID", ws.ID),
		slackIDContext("Organization ID", ws.OrgID),
	)
	if ws.StripeCustomerID.Valid && ws.StripeCustomerID.String != "" {
		blocks = append(blocks, slackIDContext("Stripe customer", ws.StripeCustomerID.String))
	}
	return blocks
}

func slackIDContext(label, id string) slack.Block {
	return slack.Block{
		Type:     "context",
		Text:     nil,
		Fields:   nil,
		Elements: []slack.Text{{Type: "mrkdwn", Text: label + ": `" + escapeSlackText(id) + "`", Emoji: false, Verbatim: true}},
	}
}

func slackName(value string) string {
	runes := []rune(strings.Join(strings.Fields(value), " "))
	if len(runes) > 256 {
		runes = append(runes[:256], '…')
	}
	return escapeSlackText(string(runes))
}

func escapeSlackText(value string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(value)
}
