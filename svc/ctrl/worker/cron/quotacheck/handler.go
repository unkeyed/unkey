// Package quotacheck implements the CronService.RunQuotaCheck handler.
// The handler queries workspace usage from ClickHouse and sends Slack
// notifications for newly exceeded quotas. Keyed by billing period
// (e.g. "2026-01"); state tracks notified workspaces so a daily
// re-trigger doesn't spam.
package quotacheck

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// stateKeyNotifiedWorkspaces tracks per-workspace last-notified
// timestamps within a billing period (VO state).
const stateKeyNotifiedWorkspaces = "notified_workspaces"

// minUsageThreshold is the minimum usage to consider for quota checks.
// Workspaces below this threshold are skipped since the minimum paid
// plan starts at 150k.
const minUsageThreshold = 150_000

// followUpInterval is the minimum time between follow-up notifications.
// First notification is sent immediately, subsequent ones are sent
// weekly. We use 6 days 20 hours instead of exactly 7 days to account
// for timing drift in the daily scheduled job (e.g., 16:03 one week vs
// 16:00 the next).
const followUpInterval = 6*24*time.Hour + 20*time.Hour

// batchSize is the number of workspace IDs to fetch from the database
// in a single query.
const batchSize = 100

// exceededWorkspace holds info about a workspace that exceeded its quota.
type exceededWorkspace struct {
	Workspace  db.GetWorkspacesForQuotaCheckByIDsRow
	Used       int64
	IsFollowUp bool
}

// Config holds the handler's dependencies.
type Config struct {
	// DB is the primary application database. Must not be nil.
	DB db.Database
	// Clickhouse is the analytics database. Must not be nil — pass
	// clickhouse.NewNoop() if unavailable.
	Clickhouse clickhouse.ClickHouse
	// Heartbeat is pinged on successful completion. Must not be nil; use
	// healthcheck.NewNoop() if monitoring is not configured.
	Heartbeat healthcheck.Heartbeat
	// SlackWebhookURL is the Slack webhook for quota-exceeded
	// notifications. Empty disables Slack notifications.
	SlackWebhookURL string
}

// Handler executes RunQuotaCheck.
type Handler struct {
	db              db.Database
	clickhouse      clickhouse.ClickHouse
	heartbeat       healthcheck.Heartbeat
	slackWebhookURL string
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil; use clickhouse.NewNoop() if unavailable"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
	); err != nil {
		return nil, err
	}
	return &Handler{
		db:              cfg.DB,
		clickhouse:      cfg.Clickhouse,
		heartbeat:       cfg.Heartbeat,
		slackWebhookURL: cfg.SlackWebhookURL,
	}, nil
}

// Handle queries all workspace usage and sends Slack notifications for
// newly exceeded quotas.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunQuotaCheckRequest,
) (*hydrav1.RunQuotaCheckResponse, error) {
	billingPeriod := restate.Key(ctx)
	logger.Info("running quota check", "billing_period", billingPeriod)

	year, month, err := parseBillingPeriod(billingPeriod)
	if err != nil {
		return nil, fmt.Errorf("invalid billing period %q: %w", billingPeriod, err)
	}

	notifiedAt, err := restate.Get[map[string]int64](ctx, stateKeyNotifiedWorkspaces)
	if err != nil {
		return nil, fmt.Errorf("get notified state: %w", err)
	}
	if notifiedAt == nil {
		notifiedAt = make(map[string]int64)
	}

	now, err := restate.Run(ctx, func(restate.RunContext) (int64, error) {
		return time.Now().Unix(), nil
	}, restate.WithName("get current time"))
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}

	usageAboveThreshold, err := restate.Run(ctx, func(rc restate.RunContext) (map[string]int64, error) {
		return h.clickhouse.GetBillableUsageAboveThreshold(rc, year, month, minUsageThreshold)
	}, restate.WithName("get billable usage above threshold"))
	if err != nil {
		return nil, fmt.Errorf("failed to get billable usage: %w", err)
	}

	logger.Info("fetched usage data", "workspaces_above_threshold", len(usageAboveThreshold))

	workspaceIDs := make([]string, 0, len(usageAboveThreshold))
	for wsID := range usageAboveThreshold {
		workspaceIDs = append(workspaceIDs, wsID)
	}
	// Sort so batch contents are stable across replays. The downstream
	// restate.Run calls are journaled by batch index; if the iteration
	// order differs on replay, the same "fetch workspaces batch N" entry
	// resolves with a different batchIDs slice and the journal diverges.
	sort.Strings(workspaceIDs)

	var exceeded []exceededWorkspace
	var newlyNotified []string
	workspacesChecked := 0

	for i := 0; i < len(workspaceIDs); i += batchSize {
		batchIDs := workspaceIDs[i:min(i+batchSize, len(workspaceIDs))]

		batch, fetchErr := restate.Run(ctx, func(rc restate.RunContext) ([]db.GetWorkspacesForQuotaCheckByIDsRow, error) {
			return db.Query.GetWorkspacesForQuotaCheckByIDs(rc, h.db.RO(), batchIDs)
		}, restate.WithName(fmt.Sprintf("fetch workspaces batch %d", i/batchSize)))
		if fetchErr != nil {
			return nil, fmt.Errorf("fetch workspaces: %w", fetchErr)
		}

		for _, ws := range batch {
			workspacesChecked++
			if workspacesChecked%1000 == 0 {
				logger.Info("progress", "count", workspacesChecked)
			}

			if !ws.Enabled {
				continue
			}
			if !ws.RequestsPerMonth.Valid {
				continue
			}

			usage := usageAboveThreshold[ws.ID]
			if usage < ws.RequestsPerMonth.Int64 {
				continue
			}

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

			if h.slackWebhookURL != "" {
				_, notifyErr := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
					return restate.Void{}, sendSlackNotification(rc, h.slackWebhookURL, e)
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

	if len(newlyNotified) > 0 {
		restate.Set(ctx, stateKeyNotifiedWorkspaces, notifiedAt)
	}

	logger.Info("quota check complete",
		"billing_period", billingPeriod,
		"workspaces_checked", workspacesChecked,
		"workspaces_exceeded", len(exceeded),
		"notifications_sent", len(newlyNotified),
	)

	if _, err := restate.Run(ctx, func(rc restate.RunContext) (restate.Void, error) {
		return restate.Void{}, h.heartbeat.Ping(rc)
	}, restate.WithName("send heartbeat")); err != nil {
		return nil, fmt.Errorf("send heartbeat: %w", err)
	}

	return &hydrav1.RunQuotaCheckResponse{
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
