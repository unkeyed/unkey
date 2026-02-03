package quotacheck

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const stateKeyNotifiedWorkspaces = "notified_workspaces"

// followUpInterval is the minimum time between follow-up notifications.
// First notification is sent immediately, subsequent ones are sent weekly.
// We use 6 days 20 hours instead of exactly 7 days to account for timing drift
// in the daily scheduled job (e.g., 16:03 one week vs 16:00 the next).
const followUpInterval = 6*24*time.Hour + 20*time.Hour

// exceededWorkspace holds info about a workspace that exceeded its quota.
type exceededWorkspace struct {
	Workspace  db.ListWorkspacesForQuotaCheckRow
	Used       int64
	IsFollowUp bool
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

	// Load notification timestamps from state (workspace ID -> Unix timestamp of last notification)
	notifiedAt, err := restate.Get[map[string]int64](ctx, stateKeyNotifiedWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("get notified state: %w", err)
	}

	if notifiedAt == nil {
		notifiedAt = make(map[string]int64)
	}

	// Get current time deterministically for Restate replay
	now, err := restate.Run(ctx, func(restate.RunContext) (int64, error) {
		return time.Now().Unix(), nil
	}, restate.WithName("get current time"))
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
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

			// Skip workspaces with no quota set
			if !ws.RequestsPerMonth.Valid {
				continue
			}

			usage := usedVerifications + usedRatelimits

			if usage < ws.RequestsPerMonth.Int64 {
				continue
			}

			// Check if we should send a notification:
			// - First time: always notify
			// - Follow-up: only if 7+ days since last notification
			lastNotified := notifiedAt[ws.ID]
			isFollowUp := lastNotified > 0
			if isFollowUp {
				timeSinceLastNotification := time.Duration(now-lastNotified) * time.Second
				if timeSinceLastNotification < followUpInterval {
					continue
				}
			}

			e := exceededWorkspace{
				Workspace:  ws,
				Used:       usage,
				IsFollowUp: isFollowUp,
			}

			// Send notification
			if s.slackWebhookURL != "" {
				_, notifyErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
					return restate.Void{}, sendSlackNotification(rc, s.slackWebhookURL, e)
				}, restate.WithName("notify "+ws.ID))
				if notifyErr != nil {
					return nil, fmt.Errorf("failed to send notification: %w", notifyErr)
				}
			}

			exceeded = append(exceeded, e)
			notifiedAt[ws.ID] = now
			newlyNotified = append(newlyNotified, ws.ID)
		}
	}

	// Update state with notification timestamps
	if len(newlyNotified) > 0 {
		restate.Set(ctx, stateKeyNotifiedWorkspaces, notifiedAt)
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

func sendSlackNotification(ctx context.Context, webhookURL string, e exceededWorkspace) error {
	printer := message.NewPrinter(language.English)

	title := fmt.Sprintf("Quota Exceeded: %s", e.Workspace.Name)
	if e.IsFollowUp {
		title = fmt.Sprintf("Quota Still Exceeded (Weekly Reminder): %s", e.Workspace.Name)
	}

	payload := slack.Payload{
		Text: title,
		Blocks: []slack.Block{
			slack.NewHeaderBlock(title),
			slack.NewSectionBlock(
				slack.NewMarkdownField(fmt.Sprintf("*Workspace ID:*\n`%s`", e.Workspace.ID)),
				slack.NewMarkdownField(fmt.Sprintf("*Workspace Name:*\n%s", e.Workspace.Name)),
				slack.NewMarkdownField(fmt.Sprintf("*Organisation ID:*\n`%s`", e.Workspace.OrgID)),
				slack.NewMarkdownField(fmt.Sprintf("*Stripe ID:*\n`%s`", e.Workspace.StripeCustomerID.String)),
			),
			slack.NewSectionBlock(
				slack.NewMarkdownField(fmt.Sprintf("*Tier:*\n%s", e.Workspace.Tier.String)),
				slack.NewMarkdownField("*Quota:*\nRequestsPerMonth"),
			),
			slack.NewSectionBlock(
				slack.NewMarkdownField(fmt.Sprintf("*Limit:*\n%s", printer.Sprint(number.Decimal(e.Workspace.RequestsPerMonth.Int64)))),
				slack.NewMarkdownField(fmt.Sprintf("*Used:*\n%s", printer.Sprint(number.Decimal(e.Used)))),
			),
		},
	}

	return slack.NewClient().Send(ctx, webhookURL, payload)
}
