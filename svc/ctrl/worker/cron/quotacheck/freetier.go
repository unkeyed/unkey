package quotacheck

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/restate/restateutil"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
	"golang.org/x/text/number"
)

const (
	freeTierExceededTemplate  = "free-tier-exceeded"
	freeTierThrottledTemplate = "free-tier-throttled"
)

const stateKeyFreeTierWarnedAt = "warned_at"

const freeTierName = "Free"

const throttleRequestsPerMinute = 5

const throttleAfter = 5 * 24 * time.Hour

const throttleStallTimeout = 24 * time.Hour

const warningStaleAfter = 7 * 24 * time.Hour

const throttleRetryDelay = time.Hour

const runMaxAttempts = 5

type freeTierAction int

const (
	freeTierActionNone freeTierAction = iota
	freeTierActionWarn
	freeTierActionRescheduleThrottle
)

type sendResult int

const (
	sendResultFailed sendResult = iota
	sendResultSent
	sendResultNoRecipients
	sendResultDeliveryOff
)

func freeTierKey(workspaceID string) string {
	return "free-tier-" + workspaceID
}

func isFreeTier(tier sql.NullString) bool {
	return tier.Valid && tier.String == freeTierName
}

func isThrottled(requestsPerMinute sql.NullInt32) bool {
	return requestsPerMinute.Valid && requestsPerMinute.Int32 <= throttleRequestsPerMinute
}

func nextFreeTierAction(warnedAt int64, now time.Time) freeTierAction {
	if warnedAt == 0 {
		return freeTierActionWarn
	}
	if now.Sub(time.Unix(warnedAt, 0)) > throttleAfter+warningStaleAfter {
		return freeTierActionWarn
	}
	if now.Sub(time.Unix(warnedAt, 0)) > throttleAfter+throttleStallTimeout {
		return freeTierActionRescheduleThrottle
	}
	return freeTierActionNone
}

func (h *Handler) HandleEnforceFreeTier(
	ctx restate.ObjectContext,
	req *hydrav1.EnforceFreeTierRequest,
) (*hydrav1.EnforceFreeTierResponse, error) {
	resp := &hydrav1.EnforceFreeTierResponse{}

	ws, found, err := h.findWorkspace(ctx, req.GetWorkspaceId())
	if err != nil {
		return nil, err
	}
	if !found || !isFreeTier(ws.Tier) || isThrottled(ws.RequestsPerMinute) {
		restate.Clear(ctx, stateKeyFreeTierWarnedAt)
		return resp, nil
	}

	warnedAt, err := restate.Get[int64](ctx, stateKeyFreeTierWarnedAt)
	if err != nil {
		return nil, fmt.Errorf("get warned_at: %w", err)
	}
	now, err := restateutil.Now(ctx)
	if err != nil {
		return nil, fmt.Errorf("get current time: %w", err)
	}

	switch nextFreeTierAction(warnedAt, now) {
	case freeTierActionNone:
		resp.WarnedAt = warnedAt
		return resp, nil

	case freeTierActionWarn:
		warnedAt = now.Unix()
		vars := h.freeTierVariables(ws, now)
		vars["USED"] = message.NewPrinter(language.English).Sprint(number.Decimal(req.GetUsed()))
		vars["THROTTLE_DATE"] = now.Add(throttleAfter).UTC().Format("January 2, 2006")
		warning, sendErr := h.emailAdmins(ctx, ws, freeTierExceededTemplate,
			fmt.Sprintf("free-tier-exceeded/%s/%d", ws.ID, warnedAt), vars)
		if sendErr != nil {
			return nil, sendErr
		}
		switch warning {
		case sendResultFailed:
			return nil, fmt.Errorf("free tier warning send failed without an error for workspace %s", ws.ID)
		case sendResultNoRecipients, sendResultDeliveryOff:
			return resp, nil
		case sendResultSent:
			h.startThrottleCountdownAfterDeliveredWarning(ctx, ws.ID, warnedAt)
			resp.WarnedAt = warnedAt
			return resp, nil
		}
		return nil, fmt.Errorf("unknown free tier warning send result %d for workspace %s", warning, ws.ID)

	case freeTierActionRescheduleThrottle:
		logger.Warn("free tier throttle overdue, rescheduling", "workspace_id", ws.ID)
		h.scheduleThrottle(ctx, ws.ID, warnedAt, 0)
		resp.WarnedAt = warnedAt
		return resp, nil
	}

	return nil, fmt.Errorf("unknown free tier action for workspace %s", ws.ID)
}

func (h *Handler) HandleThrottleFreeTierWorkspace(
	ctx restate.ObjectContext,
	req *hydrav1.ThrottleFreeTierWorkspaceRequest,
) (*hydrav1.ThrottleFreeTierWorkspaceResponse, error) {
	resp := &hydrav1.ThrottleFreeTierWorkspaceResponse{}

	warnedAt, getErr := restate.Get[int64](ctx, stateKeyFreeTierWarnedAt)
	if getErr != nil {
		return nil, fmt.Errorf("get warned_at: %w", getErr)
	}
	if warnedAt == 0 || warnedAt != req.GetWarnedAt() {
		return resp, nil
	}

	ws, found, findErr := h.findWorkspace(ctx, req.GetWorkspaceId())
	if findErr != nil {
		logger.Error("free tier throttle lookup failed, retrying later",
			"workspace_id", req.GetWorkspaceId(),
			"error", findErr.Error(),
		)
		h.scheduleThrottle(ctx, req.GetWorkspaceId(), warnedAt, throttleRetryDelay)
		return resp, nil
	}
	if !found || !isFreeTier(ws.Tier) {
		restate.Clear(ctx, stateKeyFreeTierWarnedAt)
		return resp, nil
	}

	throttleNoticeAccurate, err := restate.Run(ctx, func(rc restate.RunContext) (bool, error) {
		writeErr := h.db.TightenFreeTierApiThrottle(rc, db.TightenFreeTierApiThrottleParams{
			RequestsPerMinute: sql.NullInt32{Int32: throttleRequestsPerMinute, Valid: true},
			WorkspaceID:       ws.ID,
		})
		if writeErr != nil {
			return false, writeErr
		}
		return h.throttleNoticeIsAccurate(rc, ws.ID)
	}, restate.WithName("throttle api requests"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		logger.Error("free tier throttle failed, retrying later",
			"workspace_id", ws.ID,
			"error", err.Error(),
		)
		h.scheduleThrottle(ctx, ws.ID, warnedAt, throttleRetryDelay)
		return resp, nil
	}
	restate.Clear(ctx, stateKeyFreeTierWarnedAt)
	if !throttleNoticeAccurate {
		return resp, nil
	}
	logger.Info("throttled free tier workspace",
		"workspace_id", ws.ID,
		"requests_per_minute", throttleRequestsPerMinute,
	)

	now, nowErr := restateutil.Now(ctx)
	if nowErr != nil {
		return nil, fmt.Errorf("get current time: %w", nowErr)
	}
	h.notifyThrottledBestEffort(ctx, ws, warnedAt, now)
	return resp, nil
}

func (h *Handler) startThrottleCountdownAfterDeliveredWarning(ctx restate.ObjectContext, workspaceID string, warnedAt int64) {
	restate.Set(ctx, stateKeyFreeTierWarnedAt, warnedAt)
	h.scheduleThrottle(ctx, workspaceID, warnedAt, throttleAfter)
}

func (h *Handler) throttleNoticeIsAccurate(ctx context.Context, workspaceID string) (bool, error) {
	rows, err := h.db.GetWorkspacesForQuotaCheckByIDs(ctx, []string{workspaceID})
	if err != nil {
		return false, err
	}
	return len(rows) == 1 && isFreeTier(rows[0].Tier) && isThrottled(rows[0].RequestsPerMinute), nil
}

func (h *Handler) notifyThrottledBestEffort(
	ctx restate.ObjectContext,
	ws db.GetWorkspacesForQuotaCheckByIDsRow,
	warnedAt int64,
	now time.Time,
) {
	_, err := h.emailAdmins(ctx, ws, freeTierThrottledTemplate,
		fmt.Sprintf("free-tier-throttled/%s/%d", ws.ID, warnedAt), h.freeTierVariables(ws, now))
	if err != nil {
		logger.Error("free tier throttled email failed",
			"workspace_id", ws.ID,
			"error", err.Error(),
		)
	}
}

func (h *Handler) scheduleThrottle(ctx restate.ObjectContext, workspaceID string, warnedAt int64, delay time.Duration) {
	hydrav1.NewCronServiceClient(ctx, freeTierKey(workspaceID)).
		ThrottleFreeTierWorkspace().
		Send(&hydrav1.ThrottleFreeTierWorkspaceRequest{
			WorkspaceId: workspaceID,
			WarnedAt:    warnedAt,
		}, restate.WithDelay(delay))
}

func (h *Handler) findWorkspace(
	ctx restate.ObjectContext,
	workspaceID string,
) (db.GetWorkspacesForQuotaCheckByIDsRow, bool, error) {
	var none db.GetWorkspacesForQuotaCheckByIDsRow
	rows, err := restate.Run(ctx, func(rc restate.RunContext) ([]db.GetWorkspacesForQuotaCheckByIDsRow, error) {
		return h.db.GetWorkspacesForQuotaCheckByIDs(rc, []string{workspaceID})
	}, restate.WithName("find workspace"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return none, false, fmt.Errorf("find workspace %s: %w", workspaceID, err)
	}
	if len(rows) == 0 {
		return none, false, nil
	}
	return rows[0], true, nil
}

func (h *Handler) freeTierVariables(ws db.GetWorkspacesForQuotaCheckByIDsRow, now time.Time) map[string]string {
	printer := message.NewPrinter(language.English)
	return map[string]string{
		"WORKSPACE_NAME":      ws.Name,
		"LIMIT":               printer.Sprint(number.Decimal(ws.RequestsPerMonth.Int64)),
		"REQUESTS_PER_MINUTE": strconv.Itoa(throttleRequestsPerMinute),
		"BILLING_URL":         fmt.Sprintf("%s/%s/settings/billing", h.billingBaseURL, ws.Slug),
		"YEAR":                strconv.Itoa(now.UTC().Year()),
	}
}

func (h *Handler) emailAdmins(
	ctx restate.ObjectContext,
	ws db.GetWorkspacesForQuotaCheckByIDsRow,
	templateID string,
	idempotencyKey string,
	variables map[string]string,
) (sendResult, error) {
	if h.delivery == nil {
		return sendResultDeliveryOff, nil
	}
	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		return h.delivery.Admins.AdminEmails(rc, ws.OrgID)
	}, restate.WithName("resolve org admins"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return sendResultFailed, fmt.Errorf("resolve org admins for %s: %w", ws.ID, err)
	}
	if len(recipients) == 0 {
		logger.Warn("free tier email has no recipients",
			"workspace_id", ws.ID,
			"org_id", ws.OrgID,
			"template", templateID,
		)
		return sendResultNoRecipients, nil
	}

	err = restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.delivery.Email.Send(rc, email.Email{
			To:             recipients,
			TemplateID:     templateID,
			Variables:      variables,
			From:           "",
			Subject:        "",
			IdempotencyKey: idempotencyKey,
		})
	}, restate.WithName("send "+templateID), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return sendResultFailed, fmt.Errorf("send %s to %s: %w", templateID, ws.ID, err)
	}
	return sendResultSent, nil
}
