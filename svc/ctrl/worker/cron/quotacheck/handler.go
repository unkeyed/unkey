// Package quotacheck implements the CronService.RunQuotaCheck handler.
// The handler queries workspace usage from ClickHouse, sends internal Slack
// notifications, and emails Free-tier workspace admins on the first two
// notifications. Keyed by billing period (e.g. "2026-01"); state tracks
// notified workspaces so a daily re-trigger doesn't spam.
package quotacheck

import (
	"context"
	"fmt"
	"sort"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/billingperiod"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/healthcheck"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/slack"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// stateKeyNotifiedWorkspaces tracks per-workspace last-notified
// timestamps within a billing period (VO state).
const stateKeyNotifiedWorkspaces = "notified_workspaces"

// stateKeyCustomerEmailCounts tracks how many customer emails each workspace
// received in this billing period. It is separate from the legacy notification
// timestamp state so deploying customer email does not invalidate existing VO
// state.
const stateKeyCustomerEmailCounts = "customer_email_counts"

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

// slackNotifyMaxAttempts bounds retries on the internal Slack notification so
// a persistently failing webhook eventually yields a terminal error instead of
// retrying forever. The Slack alert is best-effort: on failure the check logs
// and continues to the customer email and remaining workspaces rather than
// aborting the whole run.
const slackNotifyMaxAttempts uint = 5

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
	// Admins resolves the workspace org's admin email addresses. Must not be
	// nil; use workos.NewNoop() when recipient lookup is disabled.
	Admins workos.Resolver
	// Email sends customer notifications. Must not be nil; use email.NewNoop()
	// when delivery is disabled.
	Email email.Sender
	// CustomerEmailEnabled distinguishes real delivery from the noop sender so
	// disabled checks do not consume the first and second customer email slots.
	CustomerEmailEnabled bool
	// BillingBaseURL is the dashboard origin used for customer action links.
	BillingBaseURL string
	// FollowUpInterval overrides the default weekly reminder cadence when set.
	// This is used by integration tests to exercise the notification sequence.
	FollowUpInterval *time.Duration
}

// Handler executes RunQuotaCheck.
type Handler struct {
	db                   db.Database
	clickhouse           clickhouse.ClickHouse
	heartbeat            healthcheck.Heartbeat
	slackWebhookURL      string
	admins               workos.Resolver
	email                email.Sender
	customerEmailEnabled bool
	billingBaseURL       string
	followUp             time.Duration
}

// New constructs a Handler.
func New(cfg Config) (*Handler, error) {
	if err := assert.All(
		assert.NotNil(cfg.DB, "DB must not be nil"),
		assert.NotNil(cfg.Clickhouse, "Clickhouse must not be nil; use clickhouse.NewNoop() if unavailable"),
		assert.NotNil(cfg.Heartbeat, "Heartbeat must not be nil; use healthcheck.NewNoop()"),
		assert.NotNil(cfg.Admins, "Admins must not be nil; use workos.NewNoop()"),
		assert.NotNil(cfg.Email, "Email must not be nil; use email.NewNoop()"),
	); err != nil {
		return nil, err
	}
	reminderInterval := followUpInterval
	if cfg.FollowUpInterval != nil {
		reminderInterval = *cfg.FollowUpInterval
	}
	return &Handler{
		db:                   cfg.DB,
		clickhouse:           cfg.Clickhouse,
		heartbeat:            cfg.Heartbeat,
		slackWebhookURL:      cfg.SlackWebhookURL,
		admins:               cfg.Admins,
		email:                cfg.Email,
		customerEmailEnabled: cfg.CustomerEmailEnabled,
		billingBaseURL:       cfg.BillingBaseURL,
		followUp:             reminderInterval,
	}, nil
}

// Handle queries all workspace usage and sends internal and customer
// notifications for newly exceeded quotas.
func (h *Handler) Handle(
	ctx restate.ObjectContext,
	_ *hydrav1.RunQuotaCheckRequest,
) (*hydrav1.RunQuotaCheckResponse, error) {
	billingPeriod := restate.Key(ctx)
	logger.Info("running quota check", "billing_period", billingPeriod)

	p, err := billingperiod.Parse(billingPeriod)
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
	emailCounts, err := restate.Get[map[string]int64](ctx, stateKeyCustomerEmailCounts)
	if err != nil {
		return nil, fmt.Errorf("get customer email state: %w", err)
	}
	if emailCounts == nil {
		emailCounts = make(map[string]int64)
	}

	nowTime, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}
	now := nowTime.Unix()

	usageAboveThreshold, err := restate.Run(ctx, func(rc restate.RunContext) (map[string]int64, error) {
		return h.clickhouse.GetBillableUsageAboveThreshold(rc, p.Year, int(p.Month), minUsageThreshold)
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
	emailCountsChanged := false
	workspacesChecked := 0

	for i := 0; i < len(workspaceIDs); i += batchSize {
		batchIDs := workspaceIDs[i:min(i+batchSize, len(workspaceIDs))]

		batch, fetchErr := restate.Run(ctx, func(rc restate.RunContext) ([]db.GetWorkspacesForQuotaCheckByIDsRow, error) {
			return h.db.GetWorkspacesForQuotaCheckByIDs(rc, batchIDs)
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
				if timeSinceLastNotification < h.followUp {
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
				}, restate.WithName("notify "+ws.ID), restate.WithMaxRetryAttempts(slackNotifyMaxAttempts))
				if notifyErr != nil {
					// Best-effort: a failing Slack webhook must not abort the
					// run and starve later workspaces of their customer emails.
					logger.Error("failed to send quota slack notification",
						"error", notifyErr,
						"org_id", ws.OrgID,
						"workspace_id", ws.ID,
					)
				}
			}

			emailSent, emailErr := h.sendCustomerEmail(ctx, billingPeriod, p.Year, emailCounts[ws.ID], e)
			if emailErr != nil {
				logger.Error("failed to send quota customer email",
					"error", emailErr,
					"org_id", ws.OrgID,
					"workspace_id", ws.ID,
				)
			} else if emailSent {
				emailCounts[ws.ID]++
				emailCountsChanged = true
			}

			exceeded = append(exceeded, e)
			notifiedAt[ws.ID] = now
			newlyNotified = append(newlyNotified, ws.ID)
		}
	}

	if len(newlyNotified) > 0 {
		restate.Set(ctx, stateKeyNotifiedWorkspaces, notifiedAt)
	}
	if emailCountsChanged {
		restate.Set(ctx, stateKeyCustomerEmailCounts, emailCounts)
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
