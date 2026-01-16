package quotacheck

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/cli"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// Cmd is the quotacheck command that monitors workspace quota usage by querying
// ClickHouse and sends Slack notifications when quotas are exceeded.
var Cmd = &cli.Command{
	Version:  "",
	Commands: []*cli.Command{},
	Aliases:  []string{},
	Name:     "quotacheck",
	Usage:    "Check for exceeded quotas",
	Description: `Check for exceeded quotas and optionally send Slack notifications.

This command monitors quota usage by querying ClickHouse for current usage metrics and comparing them against configured limits in the primary database. When quotas are exceeded, it can automatically send notifications via Slack webhook.

CONFIGURATION:
The command requires ClickHouse and database connections to function. Slack notifications are optional but recommended for production monitoring.

EXAMPLES:
unkey quotacheck --clickhouse-url clickhouse://localhost:9000 --database-dsn postgres://user:pass@localhost/db  # Check quotas without notifications
unkey quotacheck --clickhouse-url clickhouse://localhost:9000 --database-dsn postgres://user:pass@localhost/db --slack-webhook-url https://hooks.slack.com/services/...  # Check quotas with Slack notifications
CLICKHOUSE_URL=... DATABASE_DSN=... SLACK_WEBHOOK_URL=... unkey quotacheck  # Using environment variables`,
	Flags: []cli.Flag{
		cli.String("clickhouse-url", "URL for the ClickHouse database", cli.EnvVar("CLICKHOUSE_URL"), cli.Required()),
		cli.String("database-dsn", "DSN for the primary database", cli.EnvVar("DATABASE_DSN"), cli.Required()),
		cli.String("slack-webhook-url", "Slack webhook URL to send notifications", cli.EnvVar("SLACK_WEBHOOK_URL")),
	},
	Action: run,
}

// nolint:gocognit
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
		return err
	}

	ch, err := clickhouse.New(clickhouse.Config{
		URL:    cmd.String("clickhouse-url"),
		Logger: logger,
	})
	if err != nil {
		return err
	}

	logger.Info("Checking workspaces")
	counter := 0
	cursor := ""
	for {

		list, err := db.Query.ListWorkspaces(ctx, database.RO(), cursor)
		if err != nil {
			return err
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
				panic(err)
			}

			usedRatelimits, err := ch.GetBillableRatelimits(ctx, e.Workspace.ID, year, int(month))
			if err != nil {
				panic(err)
			}

			usage := usedVerifications + usedRatelimits

			if usage > e.Quotas.RequestsPerMonth {
				logger.Warn("workspace has exceeded request quota",
					"id", e.Workspace.ID,
				)
				if slackWebhookURL != "" {
					err = sendSlackNotification(slackWebhookURL, e, usage)
					if err != nil {
						panic(err)
					}
				}
			}

		}

	}

	return nil
}

// sendSlackNotification sends a message to a Slack webhook
func sendSlackNotification(webhookURL string, e db.ListWorkspacesRow, used int64) error {
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
						"text": fmt.Sprintf("*Quota:*\n%s", "RequestsPerMonth"),
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
		},
	}

	slackBody, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest(http.MethodPost, webhookURL, bytes.NewBuffer(slackBody))
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
		return fmt.Errorf("slack notification failed with status code: %d", resp.StatusCode)
	}

	return nil
}
