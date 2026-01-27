package quotacheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	stateKeyNotifiedWorkspaces = "notified_workspaces"
	// checkInterval is how often the quota check runs (daily)
	checkInterval = 24 * time.Hour
)

// exceededWorkspace holds info about a workspace that exceeded its quota.
type exceededWorkspace struct {
	Workspace db.Workspace
	Quota     db.Quotum
	Used      int64
}

// RunCheck queries all workspace usage and sends Slack notifications for newly exceeded quotas.
func (s *Service) RunCheck(
	ctx restate.ObjectContext,
	req *hydrav1.RunCheckRequest,
) (*hydrav1.RunCheckResponse, error) {
	billingPeriod := restate.Key(ctx)
	s.logger.Info("running quota check", "billing_period", billingPeriod)

	// Parse billing period to get year/month
	year, month, err := parseBillingPeriod(billingPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid billing period %q: %w", billingPeriod, err)
	}

	// Step 1: Get all workspace quotas from MySQL
	workspaces, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesWithQuotasRow, error) {
		return db.Query.ListWorkspacesWithQuotas(rc, s.db.RO())
	}, restate.WithName("fetch workspaces"))
	if err != nil {
		return nil, fmt.Errorf("fetch workspaces: %w", err)
	}

	// Step 2: Get all billable usage from ClickHouse (bulk query)
	usage, err := restate.Run(ctx, func(rc restate.RunContext) (map[string]int64, error) {
		allUsage, err := s.clickhouse.GetAllBillableUsage(rc, year, month)
		if err != nil {
			return nil, err
		}
		// Flatten to total usage per workspace
		result := make(map[string]int64)
		for wsID, u := range allUsage {
			result[wsID] = u.Verifications + u.Ratelimits
		}
		return result, nil
	}, restate.WithName("fetch usage"))
	if err != nil {
		return nil, fmt.Errorf("fetch usage: %w", err)
	}

	// Step 3: Load already-notified workspaces from state
	notified, err := restate.Get[map[string]bool](ctx, stateKeyNotifiedWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("get notified state: %w", err)
	}
	if notified == nil {
		notified = make(map[string]bool)
	}

	// Step 4: Find exceeded workspaces
	var exceeded []exceededWorkspace
	for _, ws := range workspaces {
		if !ws.Workspace.Enabled {
			continue
		}
		wsUsage := usage[ws.Workspace.ID]
		if wsUsage > ws.Quotas.RequestsPerMonth {
			exceeded = append(exceeded, exceededWorkspace{
				Workspace: ws.Workspace,
				Quota:     ws.Quotas,
				Used:      wsUsage,
			})
		}
	}

	// Step 5: Send notifications for newly exceeded workspaces
	var newlyNotified []string
	if req.GetSlackWebhookUrl() != "" {
		for _, e := range exceeded {
			if notified[e.Workspace.ID] {
				continue // Already notified this period
			}

			_, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
				return restate.Void{}, sendSlackNotification(req.GetSlackWebhookUrl(), e)
			}, restate.WithName("notify "+e.Workspace.ID))
			if err != nil {
				s.logger.Error("failed to send slack notification",
					"workspace_id", e.Workspace.ID,
					"error", err,
				)
				continue
			}

			newlyNotified = append(newlyNotified, e.Workspace.ID)
		}
	}

	// Step 6: Update state with newly notified workspaces
	if len(newlyNotified) > 0 {
		for _, wsID := range newlyNotified {
			notified[wsID] = true
		}
		restate.Set(ctx, stateKeyNotifiedWorkspaces, notified)
	}

	s.logger.Info("quota check complete",
		"billing_period", billingPeriod,
		"workspaces_checked", len(workspaces),
		"workspaces_exceeded", len(exceeded),
		"notifications_sent", len(newlyNotified),
	)

	// Schedule next run - this creates the Restate cron pattern
	// The job will run again after checkInterval (daily)
	now := time.Now().UTC()
	nextRun := now.Add(checkInterval)
	nextBillingPeriod := nextRun.Format("2006-01")

	selfClient := hydrav1.NewQuotaCheckServiceClient(ctx, nextBillingPeriod)
	selfClient.RunCheck().Send(
		&hydrav1.RunCheckRequest{
			SlackWebhookUrl: req.GetSlackWebhookUrl(),
		},
		restate.WithDelay(checkInterval),
		restate.WithIdempotencyKey(fmt.Sprintf("quota-check-%s", nextRun.Format("2006-01-02"))),
	)
	s.logger.Info("scheduled next quota check", "delay", checkInterval, "billing_period", nextBillingPeriod)

	// Check if today is the last day of the month - if so, schedule monthly summary
	tomorrow := now.AddDate(0, 0, 1)
	if tomorrow.Month() != now.Month() {
		// Tomorrow is a new month, so today is the last day
		// Schedule summary to run in 1 hour (giving time for final daily check to complete)
		summaryDelay := 1 * time.Hour
		selfClient = hydrav1.NewQuotaCheckServiceClient(ctx, billingPeriod)
		selfClient.SendMonthlySummary().Send(
			&hydrav1.SendMonthlySummaryRequest{
				SlackWebhookUrl: req.GetSlackWebhookUrl(),
			},
			restate.WithDelay(summaryDelay),
			restate.WithIdempotencyKey(fmt.Sprintf("quota-summary-%s", billingPeriod)),
		)
		s.logger.Info("scheduled monthly summary", "billing_period", billingPeriod)
	}

	return &hydrav1.RunCheckResponse{
		WorkspacesChecked:  int32(len(workspaces)),
		WorkspacesExceeded: int32(len(exceeded)),
		NotificationsSent:  int32(len(newlyNotified)),
	}, nil
}

// SendMonthlySummary sends a summary of all exceeded workspaces for the billing period.
func (s *Service) SendMonthlySummary(
	ctx restate.ObjectContext,
	req *hydrav1.SendMonthlySummaryRequest,
) (*hydrav1.SendMonthlySummaryResponse, error) {
	billingPeriod := restate.Key(ctx)
	s.logger.Info("sending monthly summary", "billing_period", billingPeriod)

	if req.GetSlackWebhookUrl() == "" {
		return &hydrav1.SendMonthlySummaryResponse{WorkspacesInSummary: 0}, nil
	}

	// Load notified workspaces from state
	notified, err := restate.Get[map[string]bool](ctx, stateKeyNotifiedWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("get notified state: %w", err)
	}
	if notified == nil || len(notified) == 0 {
		s.logger.Info("no exceeded workspaces to summarize", "billing_period", billingPeriod)
		return &hydrav1.SendMonthlySummaryResponse{WorkspacesInSummary: 0}, nil
	}

	// Send summary notification
	_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, sendSummaryNotification(req.GetSlackWebhookUrl(), billingPeriod, notified)
	}, restate.WithName("send summary"))
	if err != nil {
		return nil, fmt.Errorf("send summary: %w", err)
	}

	return &hydrav1.SendMonthlySummaryResponse{
		WorkspacesInSummary: int32(len(notified)),
	}, nil
}

func parseBillingPeriod(period string) (year, month int, err error) {
	parts := strings.Split(period, "-")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected YYYY-MM format")
	}
	year, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid year: %w", err)
	}
	month, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, fmt.Errorf("invalid month: %w", err)
	}
	if month < 1 || month > 12 {
		return 0, 0, fmt.Errorf("month must be 1-12")
	}
	return year, month, nil
}

func sendSlackNotification(webhookURL string, e exceededWorkspace) error {
	printer := message.NewPrinter(language.English)

	payload := map[string]any{
		"text": fmt.Sprintf("Quota Exceeded: %s", e.Workspace.Name),
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]any{
					"type":  "plain_text",
					"text":  fmt.Sprintf("Quota Exceeded: %s", e.Workspace.Name),
					"emoji": true,
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Workspace ID:*\n`%s`", e.Workspace.ID)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Workspace Name:*\n%s", e.Workspace.Name)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Organisation ID:*\n`%s`", e.Workspace.OrgID)},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Stripe ID:*\n`%s`", e.Workspace.StripeCustomerID.String)},
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Tier:*\n%s", e.Workspace.Tier.String)},
					{"type": "mrkdwn", "text": "*Quota:*\nRequestsPerMonth"},
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{"type": "mrkdwn", "text": fmt.Sprintf("*Limit:*\n%s", printer.Sprint(number.Decimal(e.Quota.RequestsPerMonth)))},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Used:*\n%s", printer.Sprint(number.Decimal(e.Used)))},
				},
			},
		},
	}

	return postSlack(webhookURL, payload)
}

func sendSummaryNotification(webhookURL, billingPeriod string, notified map[string]bool) error {
	var workspaceIDs []string
	for wsID := range notified {
		workspaceIDs = append(workspaceIDs, wsID)
	}

	payload := map[string]any{
		"text": fmt.Sprintf("Monthly Quota Summary: %s", billingPeriod),
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]any{
					"type":  "plain_text",
					"text":  fmt.Sprintf("Monthly Quota Summary: %s", billingPeriod),
					"emoji": true,
				},
			},
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*%d workspaces* exceeded their quota this month.", len(notified)),
				},
			},
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": fmt.Sprintf("```%s```", strings.Join(workspaceIDs, "\n")),
				},
			},
		},
	}

	return postSlack(webhookURL, payload)
}

func postSlack(webhookURL string, payload map[string]any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack returned status %d", resp.StatusCode)
	}

	return nil
}
