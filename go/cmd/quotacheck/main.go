package quotacheck

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"context"

	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"

	"github.com/urfave/cli/v3"
)

var Cmd = &cli.Command{
	Name:        "quotacheck",
	Description: "Check for exceeded quotas",
	Flags: []cli.Flag{
		&cli.IntFlag{
			Name:    "parallel",
			Value:   16,
			Usage:   "Number of parallel workers to process workspace quota checks",
			Aliases: []string{"p"},
		},
		&cli.StringFlag{
			Name:     "clickhouse-url",
			Usage:    "URL for the ClickHouse database",
			Sources:  cli.EnvVars("CLICKHOUSE_URL"),
			Required: true,
		},
		&cli.StringFlag{
			Name:     "database-dsn",
			Usage:    "DSN for the primary database",
			Sources:  cli.EnvVars("DATABASE_DSN"),
			Required: true,
		},
		&cli.StringFlag{
			Name:    "slack-webhook-url",
			Usage:   "Slack webhook URL to send notifications",
			Sources: cli.EnvVars("SLACK_WEBHOOK_URL"),
		},
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
	queue := make(chan db.ListWorkspacesRow, 1)

	wg := sync.WaitGroup{}

	workers := cmd.Int("parallel")
	logger.Info("Creating workers", "goroutines", workers)
	for range workers {
		go func() {
			for e := range queue {
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
				wg.Done()
			}

		}()
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
				logger.Info("progress", "n", counter)
			}
			wg.Add(1)
			queue <- e
		}

	}

	wg.Wait()
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
						"text": fmt.Sprintf("*Clerk ID:*\n`%s`", e.Workspace.TenantID),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Organisation ID:*\n`%s`", e.Workspace.OrgID.String),
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status code: %d", resp.StatusCode)
	}

	return nil
}
