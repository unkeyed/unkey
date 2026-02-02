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

const stateKeyNotifiedWorkspaces = "notified_workspaces"

// exceededWorkspace holds info about a workspace that exceeded its quota.
type exceededWorkspace struct {
	Workspace db.ListWorkspacesForQuotaCheckRow
	Used      int64
}

// RunCheck queries all workspace usage and sends Slack notifications for newly exceeded quotas.
// This handler is intended to be called on a schedule via GitHub Actions.
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

	// Load already-notified workspaces from state
	notified, err := restate.Get[map[string]bool](ctx, stateKeyNotifiedWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("get notified state: %w", err)
	}

	if notified == nil {
		notified = make(map[string]bool)
	}

	var exceeded []exceededWorkspace
	var newlyNotified []string
	workspacesChecked := 0
	cursor := ""

	// Iterate through all workspaces with cursor-based pagination
	for {
		// Fetch next batch of workspaces
		currentCursor := cursor
		batch, fetchErr := restate.Run(ctx, func(rc restate.RunContext) ([]db.ListWorkspacesForQuotaCheckRow, error) {
			return db.Query.ListWorkspacesForQuotaCheck(rc, s.db.RO(), currentCursor)
		}, restate.WithName("fetch workspaces after "+currentCursor))
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch workspaces: %w", fetchErr)
		}

		if len(batch) == 0 {
			break
		}
		cursor = batch[len(batch)-1].ID

		// Process each workspace in the batch
		for _, ws := range batch {
			workspacesChecked++
			if workspacesChecked%100 == 0 {
				s.logger.Info("progress", "count", workspacesChecked)
			}

			if !ws.Enabled {
				continue
			}

			// Skip workspaces we've already notified this billing period
			if notified[ws.ID] {
				continue
			}

			// Get usage for this workspace from ClickHouse
			usedVerifications, verErr := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
				return s.clickhouse.GetBillableVerifications(rc, ws.ID, year, month)
			}, restate.WithName("get verifications "+ws.ID))
			if verErr != nil {
				return nil, fmt.Errorf("failed to get verifications: %w", verErr)
			}

			usedRatelimits, rlErr := restate.Run(ctx, func(rc restate.RunContext) (int64, error) {
				return s.clickhouse.GetBillableRatelimits(rc, ws.ID, year, month)
			}, restate.WithName("get ratelimits "+ws.ID))
			if rlErr != nil {
				return nil, fmt.Errorf("failed to get ratelimits: %w", rlErr)
			}

			usage := usedVerifications + usedRatelimits

			if usage < ws.RequestsPerMonth.Int64 {
				continue
			}

			e := exceededWorkspace{
				Workspace: ws,
				Used:      usage,
			}

			// Send notification
			if req.GetSlackWebhookUrl() != "" {
				_, notifyErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
					return restate.Void{}, sendSlackNotification(req.GetSlackWebhookUrl(), e)
				}, restate.WithName("notify "+ws.ID))
				if notifyErr != nil {
					return nil, fmt.Errorf("failed to send notification: %w", notifyErr)
				}
			}

			exceeded = append(exceeded, e)
			notified[ws.ID] = true
			newlyNotified = append(newlyNotified, ws.ID)
		}
	}

	// Update state with newly notified workspaces
	if len(newlyNotified) > 0 {
		restate.Set(ctx, stateKeyNotifiedWorkspaces, notified)
	}

	s.logger.Info("quota check complete",
		"billing_period", billingPeriod,
		"workspaces_checked", workspacesChecked,
		"workspaces_exceeded", len(exceeded),
		"notifications_sent", len(newlyNotified),
	)

	// Send heartbeat to indicate successful completion
	_, err = restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, s.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat"))
	if err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunCheckResponse{
		WorkspacesChecked:  int32(workspacesChecked),
		WorkspacesExceeded: int32(len(exceeded)),
		NotificationsSent:  int32(len(newlyNotified)),
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
					{"type": "mrkdwn", "text": fmt.Sprintf("*Limit:*\n%s", printer.Sprint(number.Decimal(e.Workspace.RequestsPerMonth.Int64)))},
					{"type": "mrkdwn", "text": fmt.Sprintf("*Used:*\n%s", printer.Sprint(number.Decimal(e.Used)))},
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
