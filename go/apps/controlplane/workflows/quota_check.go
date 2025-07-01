package workflows

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// QuotaCheckWorkflow checks for workspaces that have exceeded their quotas
// and sends notifications when necessary
type QuotaCheckWorkflow struct {
	Logger       logging.Logger
	Database     db.Database
	Clickhouse   clickhouse.ClickHouse
	SlackWebhook string
}

// Name returns the unique name for this workflow type
func (w *QuotaCheckWorkflow) Name() string {
	return "quota-check"
}

// QuotaCheckRequest represents the input data for the quota check workflow
type QuotaCheckRequest struct {
	Year  int `json:"year"`
	Month int `json:"month"`
}

// Run executes the quota check workflow logic
func (w *QuotaCheckWorkflow) Run(ctx hydra.WorkflowContext, req QuotaCheckRequest) error {
	w.Logger.Info("quota check workflow started",
		"execution_id", ctx.ExecutionID(),
		"year", req.Year,
		"month", req.Month,
	)

	// Step 1: Initialize and validate inputs
	_, err := hydra.Step(ctx, "initialize", func(stepCtx context.Context) (string, error) {
		if req.Year == 0 || req.Month == 0 {
			return "", fmt.Errorf("year and month must be provided")
		}

		w.Logger.Info("quota check initialized",
			"year", req.Year,
			"month", req.Month,
		)
		return "initialized", nil
	})
	if err != nil {
		return fmt.Errorf("initialization failed: %w", err)
	}

	// Step 2: Fetch and process workspaces
	processResult, err := hydra.Step(ctx, "process-workspaces", func(stepCtx context.Context) (map[string]interface{}, error) {
		return w.processWorkspaces(stepCtx, req.Year, req.Month)
	})
	if err != nil {
		return fmt.Errorf("workspace processing failed: %w", err)
	}

	// Step 3: Send summary notification if there were violations
	_, err = hydra.Step(ctx, "send-summary", func(stepCtx context.Context) (string, error) {
		violationsCount := processResult["violations_count"].(int)
		totalWorkspaces := processResult["total_workspaces"].(int)

		w.Logger.Info("quota check completed",
			"total_workspaces", totalWorkspaces,
			"violations_found", violationsCount,
		)

		if violationsCount > 0 && w.SlackWebhook != "" {
			return w.sendSummaryNotification(stepCtx, violationsCount, totalWorkspaces, req.Year, req.Month)
		}

		return "no_summary_needed", nil
	})
	if err != nil {
		return fmt.Errorf("summary notification failed: %w", err)
	}

	w.Logger.Info("quota check workflow completed successfully",
		"execution_id", ctx.ExecutionID(),
	)

	return nil
}

// processWorkspaces iterates through all workspaces and checks their quotas
func (w *QuotaCheckWorkflow) processWorkspaces(ctx context.Context, year, month int) (map[string]interface{}, error) {
	w.Logger.Info("starting workspace quota check")

	counter := 0
	violations := 0
	cursor := ""

	for {
		// Fetch workspaces in batches
		list, err := db.Query.ListWorkspaces(ctx, w.Database.RO(), cursor)
		if err != nil {
			return nil, fmt.Errorf("failed to list workspaces: %w", err)
		}

		if len(list) == 0 {
			break
		}
		cursor = list[len(list)-1].Workspace.ID

		for _, workspace := range list {
			counter++
			if counter%100 == 0 {
				w.Logger.Info("quota check progress", "processed", counter)
			}

			if !workspace.Workspace.Enabled {
				continue
			}

			// Check quota for this workspace
			violated, err := w.checkWorkspaceQuota(ctx, workspace, year, month)
			if err != nil {
				w.Logger.Error("failed to check workspace quota",
					"workspace_id", workspace.Workspace.ID,
					"error", err,
				)
				continue // Continue with other workspaces
			}

			if violated {
				violations++
			}
		}
	}

	return map[string]interface{}{
		"total_workspaces": counter,
		"violations_count": violations,
	}, nil
}

// checkWorkspaceQuota checks if a single workspace has exceeded its quota
func (w *QuotaCheckWorkflow) checkWorkspaceQuota(ctx context.Context, workspace db.ListWorkspacesRow, year, month int) (bool, error) {
	// Get usage data from ClickHouse
	usedVerifications, err := w.Clickhouse.GetBillableVerifications(ctx, workspace.Workspace.ID, year, month)
	if err != nil {
		return false, fmt.Errorf("failed to get verifications: %w", err)
	}

	usedRatelimits, err := w.Clickhouse.GetBillableRatelimits(ctx, workspace.Workspace.ID, year, month)
	if err != nil {
		return false, fmt.Errorf("failed to get ratelimits: %w", err)
	}

	totalUsage := usedVerifications + usedRatelimits

	// Check if quota is exceeded
	if totalUsage > workspace.Quotas.RequestsPerMonth {
		w.Logger.Warn("workspace exceeded quota",
			"workspace_id", workspace.Workspace.ID,
			"workspace_name", workspace.Workspace.Name,
			"usage", totalUsage,
			"limit", workspace.Quotas.RequestsPerMonth,
		)

		// Send individual workspace notification
		if w.SlackWebhook != "" {
			err = w.sendWorkspaceViolationNotification(ctx, workspace, totalUsage)
			if err != nil {
				w.Logger.Error("failed to send slack notification",
					"workspace_id", workspace.Workspace.ID,
					"error", err,
				)
				// Don't fail the workflow for notification errors
			}
		}

		return true, nil
	}

	return false, nil
}

// sendWorkspaceViolationNotification sends a Slack notification for a specific workspace violation
func (w *QuotaCheckWorkflow) sendWorkspaceViolationNotification(ctx context.Context, workspace db.ListWorkspacesRow, used int64) error {
	payload := map[string]any{
		"text": fmt.Sprintf("Quota Exceeded: %s", workspace.Workspace.Name),
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]any{
					"type":  "plain_text",
					"text":  fmt.Sprintf("🚨 Quota Exceeded: %s", workspace.Workspace.Name),
					"emoji": true,
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Workspace ID:*\n`%s`", workspace.Workspace.ID),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Workspace Name:*\n%s", workspace.Workspace.Name),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Organisation ID:*\n`%s`", workspace.Workspace.OrgID),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Stripe ID:*\n`%s`", workspace.Workspace.StripeCustomerID.String),
					},
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Workspace Tier:*\n%s", workspace.Workspace.Tier.String),
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
						"text": fmt.Sprintf("*Limit:*\n%s", message.NewPrinter(language.English).Sprint(number.Decimal(workspace.Quotas.RequestsPerMonth))),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Used:*\n%s", message.NewPrinter(language.English).Sprint(number.Decimal(used))),
					},
				},
			},
		},
	}

	return w.sendSlackMessage(ctx, payload)
}

// sendSummaryNotification sends a summary of quota violations
func (w *QuotaCheckWorkflow) sendSummaryNotification(ctx context.Context, violations, total, year, month int) (string, error) {
	payload := map[string]any{
		"text": fmt.Sprintf("Quota Check Complete: %d violations found", violations),
		"blocks": []map[string]any{
			{
				"type": "header",
				"text": map[string]any{
					"type":  "plain_text",
					"text":  "📊 Quota Check Summary",
					"emoji": true,
				},
			},
			{
				"type": "section",
				"fields": []map[string]any{
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Period:*\n%d-%02d", year, month),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Workspaces Checked:*\n%s", message.NewPrinter(language.English).Sprint(number.Decimal(total))),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Violations Found:*\n%s", message.NewPrinter(language.English).Sprint(number.Decimal(violations))),
					},
					{
						"type": "mrkdwn",
						"text": fmt.Sprintf("*Compliance Rate:*\n%.2f%%", float64(total-violations)/float64(total)*100),
					},
				},
			},
		},
	}

	err := w.sendSlackMessage(ctx, payload)
	if err != nil {
		return "", err
	}

	return "summary_sent", nil
}

// sendSlackMessage sends a message to Slack webhook
func (w *QuotaCheckWorkflow) sendSlackMessage(ctx context.Context, payload map[string]any) error {
	slackBody, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal slack payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, w.SlackWebhook, bytes.NewBuffer(slackBody))
	if err != nil {
		return fmt.Errorf("failed to create slack request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send slack message: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("slack notification failed with status code: %d", resp.StatusCode)
	}

	return nil
}

// Start is a convenience method to start this workflow manually
func (w *QuotaCheckWorkflow) Start(ctx context.Context, engine *hydra.Engine, year, month int) (string, error) {
	req := QuotaCheckRequest{
		Year:  year,
		Month: month,
	}
	return engine.StartWorkflow(ctx, w.Name(), req)
}
