package quotacheck

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

// emailMode is the free-tier warning switch. off is Slack only.
type emailMode string

const (
	emailModeOff    emailMode = "off"
	emailModeDryRun emailMode = "dry_run"
	emailModeLive   emailMode = "live"
)

const (
	skipReasonDisabled            = "disabled"
	skipReasonMissingCap          = "missing_cap"
	skipReasonDeleted             = "deleted"
	skipReasonPaidTier            = "paid_tier"
	skipReasonAPISubscription     = "api_subscription"
	skipReasonLegacySubscriptions = "legacy_subscriptions"
	skipReasonRateLimited         = "rate_limited"
	skipReasonAlreadySent         = "already_sent"
	skipReasonNoAdmins            = "no_admins"
)

func parseEmailMode(raw string) emailMode {
	switch emailMode(raw) {
	case emailModeOff, emailModeDryRun, emailModeLive:
		return emailMode(raw)
	default:
		return emailModeOff
	}
}

// activeEmailMode is the mode this run actually applies. live without a
// Resend key is off so a noop send cannot be recorded as delivered.
func (h *Handler) activeEmailMode() emailMode {
	if h.emailMode == emailModeLive && !h.resendConfigured {
		return emailModeOff
	}
	return h.emailMode
}

// noteSkip logs a workspace the warning will not email. off stays quiet so
// the Slack-only run does not grow a new log stream. Returns 1 when it logs.
func (h *Handler) noteSkip(mode emailMode, ws db.GetWorkspacesForQuotaCheckByIDsRow, usage int64, reason string) int {
	if mode != emailModeDryRun && mode != emailModeLive {
		return 0
	}
	attrs := []any{
		"workspace_id", ws.ID,
		"name", ws.Name,
		"slug", ws.Slug,
		"usage", usage,
		"reason", reason,
	}
	if ws.RequestsPerMonth.Valid {
		attrs = append(attrs, "cap", ws.RequestsPerMonth.Int64)
	}
	if reason == skipReasonRateLimited && ws.ApiRequestsCountMaxPerMinute.Valid {
		attrs = append(attrs, "requests_per_minute", ws.ApiRequestsCountMaxPerMinute.Int32)
	}
	logger.Info("free quota warning skipped", attrs...)
	return 1
}

// considerFreeQuotaWarning emails a free workspace that is over its own cap,
// or logs what it would do. Failures are logged and not marked, so the next
// daily run retries that workspace without failing the Slack report.
// Counts are sent, dry-run, skipped.
func (h *Handler) considerFreeQuotaWarning(
	ctx restate.ObjectContext,
	mode emailMode,
	billingPeriod string,
	now time.Time,
	sentAt map[string]int64,
	ws db.GetWorkspacesForQuotaCheckByIDsRow,
	usage int64,
) (int, int, int) {
	if mode == emailModeOff {
		return 0, 0, 0
	}
	if reason := freeQuotaSkipReason(ws); reason != "" {
		return 0, 0, h.noteSkip(mode, ws, usage, reason)
	}
	if _, ok := sentAt[ws.ID]; ok {
		return 0, 0, h.noteSkip(mode, ws, usage, skipReasonAlreadySent)
	}

	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		return h.admins.AdminEmails(rc, ws.OrgID)
	}, restate.WithName("quota warning admins "+ws.ID), restate.WithMaxRetryAttempts(emailRunMaxAttempts))
	if err != nil {
		logger.Error("free quota warning admin lookup failed",
			"workspace_id", ws.ID,
			"name", ws.Name,
			"slug", ws.Slug,
			"error", err.Error(),
		)
		return 0, 0, 0
	}
	if len(recipients) == 0 {
		return 0, 0, h.noteSkip(mode, ws, usage, skipReasonNoAdmins)
	}

	if mode == emailModeDryRun {
		logger.Info("free quota warning dry run",
			"workspace_id", ws.ID,
			"name", ws.Name,
			"slug", ws.Slug,
			"usage", usage,
			"cap", ws.RequestsPerMonth.Int64,
			"admin_count", len(recipients),
		)
		return 0, 1, 0
	}

	sent, err := restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
		sendErr := h.email.Send(rc, email.Email{
			From:           "",
			To:             recipients,
			TemplateID:     freeQuotaWarningTemplate,
			Variables:      quotaWarningVariables(h.billingBaseURL, ws, usage, now),
			Subject:        "",
			IdempotencyKey: freeQuotaWarningIdempotencyKey(ws.ID, billingPeriod),
		})
		if sendErr != nil {
			return false, sendErr
		}
		return true, nil
	}, restate.WithName("quota warning send "+ws.ID), restate.WithMaxRetryAttempts(emailRunMaxAttempts))
	if err != nil || !sent {
		attrs := []any{
			"workspace_id", ws.ID,
			"name", ws.Name,
			"slug", ws.Slug,
		}
		if err != nil {
			attrs = append(attrs, "error", err.Error())
		}
		logger.Error("free quota warning send failed", attrs...)
		return 0, 0, 0
	}

	sentAt[ws.ID] = now.Unix()
	restate.Set(ctx, stateKeyFreeQuotaWarningSent, sentAt)
	logger.Info("free quota warning sent",
		"workspace_id", ws.ID,
		"name", ws.Name,
		"slug", ws.Slug,
		"usage", usage,
		"cap", ws.RequestsPerMonth.Int64,
		"admin_count", len(recipients),
	)
	return 1, 0, 0
}

// freeQuotaSkipReason reports why this over-cap workspace is not a warning
// candidate. Empty means it is a free API workspace with no workspace throttle.
func freeQuotaSkipReason(ws db.GetWorkspacesForQuotaCheckByIDsRow) string {
	if ws.DeletedAtM.Valid {
		return skipReasonDeleted
	}
	if !isFreeAPITier(ws.Tier) {
		return skipReasonPaidTier
	}
	if ws.ApiSubscriptionID.Valid {
		return skipReasonAPISubscription
	}
	if hasLegacySubscriptions(ws.Subscriptions) {
		return skipReasonLegacySubscriptions
	}
	if hasWorkspaceRateLimit(ws.ApiRequestsCountMaxPerMinute) {
		return skipReasonRateLimited
	}
	return ""
}

func isFreeAPITier(tier sql.NullString) bool {
	return !tier.Valid || tier.String == "Free"
}

// hasWorkspaceRateLimit reports whether the API already throttles this
// workspace. limits.api_requests_count_max_per_minute NULL means there is no
// workspace throttle (the free-plan default, and the only value the verify
// path ignores). Any stored value is enforced, including zero, which blocks
// every request.
func hasWorkspaceRateLimit(limit sql.NullInt32) bool {
	return limit.Valid
}

// hasLegacySubscriptions reports a non-empty workspaces.subscriptions blob.
// NULL, JSON null, {}, and [] are the current billing path. Anything else is
// the leftover custom-pricing blob and must not get the free-tier warning.
func hasLegacySubscriptions(raw []byte) bool {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return false
	}
	var value any
	if err := json.Unmarshal(trimmed, &value); err != nil {
		return true
	}
	switch parsed := value.(type) {
	case nil:
		return false
	case map[string]any:
		return len(parsed) > 0
	case []any:
		return len(parsed) > 0
	default:
		return true
	}
}

func quotaWarningVariables(billingBaseURL string, ws db.GetWorkspacesForQuotaCheckByIDsRow, usage int64, now time.Time) map[string]string {
	printer := message.NewPrinter(language.English)
	return map[string]string{
		"WORKSPACE_NAME": ws.Name,
		"USED":           printer.Sprint(number.Decimal(usage)),
		"LIMIT":          printer.Sprint(number.Decimal(ws.RequestsPerMonth.Int64)),
		"BILLING_URL":    fmt.Sprintf("%s/%s/settings/billing", billingBaseURL, ws.Slug),
		"YEAR":           strconv.Itoa(now.Year()),
	}
}

func freeQuotaWarningIdempotencyKey(workspaceID, period string) string {
	return fmt.Sprintf("free-quota-warning/%s/%s", workspaceID, period)
}
