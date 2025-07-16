package quotacheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/cmd/cli/cli"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

var Command = &cli.Command{
	Name:  "quotacheck",
	Usage: "Check for workspaces that have exceeded their quotas",
	Description: `Check all workspaces for quota violations and optionally send Slack notifications.
This command scans all enabled workspaces and compares their monthly usage against their quota limits.

EXAMPLES:
    # Check quotas with just database connections
    unkey quotacheck --database-dsn="postgres://..." --clickhouse-url="http://..."
    
    # Check quotas and send Slack notifications for violations  
    unkey quotacheck --database-dsn="postgres://..." --clickhouse-url="http://..." --slack-webhook-url="https://hooks.slack.com/..."`,
	Flags: []cli.Flag{
		cli.String("clickhouse-url", "URL for the ClickHouse database", "", "CLICKHOUSE_URL", true),
		cli.String("database-dsn", "DSN for the primary database", "", "DATABASE_DSN", true),
		cli.String("slack-webhook-url", "Slack webhook URL to send notifications", "", "SLACK_WEBHOOK_URL", false),
	},
	Action: run,
}

func run(ctx context.Context, cmd *cli.Command) error {
	year, month, _ := time.Now().Date()
	logger := logging.New()
	slackWebhookURL := cmd.String("slack-webhook-url")

	database, err := db.New(db.Config{
		PrimaryDSN:  cmd.String("database-dsn"),
		ReadOnlyDSN: "",
		Logger:      logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}

	ch, err := clickhouse.New(clickhouse.Config{
		URL:    cmd.String("clickhouse-url"),
		Logger: logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}

	logger.Info("Starting quota check for all workspaces")
	counter := 0
	violationsFound := 0
	cursor := ""

	for {
		list, err := db.Query.ListWorkspaces(ctx, database.RO(), cursor)
		if err != nil {
			return fmt.Errorf("failed to list workspaces: %w", err)
		}

		if len(list) == 0 {
			break
		}

		cursor = list[len(list)-1].Workspace.ID

		for _, e := range list {
			counter++
			if counter%100 == 0 {
				logger.Info("progress", "count", counter)
			}

			if !e.Workspace.Enabled {
				continue
			}

			usedVerifications, err := ch.GetBillableVerifications(ctx, e.Workspace.ID, year, int(month))
			if err != nil {
				return fmt.Errorf("failed to get billable verifications for workspace %s: %w", e.Workspace.ID, err)
			}

			usedRatelimits, err := ch.GetBillableRatelimits(ctx, e.Workspace.ID, year, int(month))
			if err != nil {
				return fmt.Errorf("failed to get billable ratelimits for workspace %s: %w", e.Workspace.ID, err)
			}

			usage := usedVerifications + usedRatelimits

			if usage > e.Quotas.RequestsPerMonth {
				violationsFound++
				logger.Warn("workspace has exceeded request quota",
					"workspace_id", e.Workspace.ID,
					"workspace_name", e.Workspace.Name,
					"usage", usage,
					"limit", e.Quotas.RequestsPerMonth,
				)

				if slackWebhookURL != "" {
					err = sendSlackNotification(slackWebhookURL, e, usage)
					if err != nil {
						return fmt.Errorf("failed to send slack notification for workspace %s: %w", e.Workspace.ID, err)
					}
					logger.Info("sent Slack notification", "workspace_id", e.Workspace.ID)
				}
			}
		}
	}

	logger.Info("quota check completed",
		"total_workspaces_checked", counter,
		"violations_found", violationsFound,
	)

	if violationsFound > 0 {
		fmt.Printf("Found %d workspace(s) exceeding quotas out of %d checked\n", violationsFound, counter)
	} else {
		fmt.Printf("All %d workspaces are within their quota limits\n", counter)
	}

	return nil
}

func sendSlackNotification(webhookURL string, e db.ListWorkspacesRow, used int64) error {
	payload := map[string]any{
		"text": fmt.Sprintf("Quota Exceeded: %s", e.Workspace.Name),
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]any{
					"type":  "plain_text",
					"text":  fmt.Sprintf("🚨 Quota Exceeded: %s", e.Workspace.Name),
					"emoji": true,
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Workspace ID:*\n`%s`", e.Workspace.ID),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Workspace Name:*\n%s", e.Workspace.Name),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Organisation ID:*\n`%s`", e.Workspace.OrgID),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Stripe ID:*\n`%s`", e.Workspace.StripeCustomerID.String),
					},
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Workspace Tier:*\n%s", e.Workspace.Tier.String),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Quota Type:*\nRequests Per Month"),
					},
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Limit:*\n%s", message.NewPrinter(language.English).Sprint(number.Decimal(e.Quotas.RequestsPerMonth))),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Used:*\n%s", message.NewPrinter(language.English).Sprint(number.Decimal(used))),
					},
				},
			},
			{
				"type": "section",
				"text": map[string]any{
					"type": "mrkdwn",
					"text": fmt.Sprintf("*Overage:* %s requests (%.1f%% over limit)",
						message.NewPrinter(language.English).Sprint(number.Decimal(used-e.Quotas.RequestsPerMonth)),
						float64(used-e.Quotas.RequestsPerMonth)/float64(e.Quotas.RequestsPerMonth)*100,
					),
				},
			},
		},
	}

	slackBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, webhookURL, bytes.NewBuffer(slackBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status code: %d", resp.StatusCode)
	}

	return nil
}
