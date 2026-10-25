package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	restate "github.com/restatedev/sdk-go"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/proto"
)

const maxAttempts = 8

var errDeliveryChanged = errors.New("notification delivery changed")

type attemptResult struct {
	Status     db.NotificationDeliveriesStatus `json:"status"`
	RetryAfter time.Duration                   `json:"retryAfter"`
}

func (s *Service) attempt(ctx context.Context, ref deliveryRef) (attemptResult, error) {
	for {
		row, err := s.db.FindNotificationDelivery(ctx, db.FindNotificationDeliveryParams{WorkspaceID: ref.WorkspaceID, ID: ref.ID})
		if err != nil {
			return attemptResult{}, err
		}
		result, err := s.attemptDelivery(ctx, row)
		if !errors.Is(err, errDeliveryChanged) {
			return result, err
		}
	}
}

func (s *Service) attemptDelivery(ctx context.Context, row db.NotificationDelivery) (attemptResult, error) {
	if row.Status != db.NotificationDeliveriesStatusPending && row.Status != db.NotificationDeliveriesStatusSending {
		return attemptResult{Status: row.Status, RetryAfter: 0}, nil
	}
	uncertain := row.Status == db.NotificationDeliveriesStatusSending
	stop := func(reason string) (attemptResult, error) {
		status := db.NotificationDeliveriesStatusSuppressed
		if uncertain {
			status = db.NotificationDeliveriesStatusUnknown
		}
		return s.record(ctx, row, status, reason, "", 0)
	}
	provider := s.providers[row.Provider.String]
	if provider == nil || string(provider.Channel()) != string(row.Channel) {
		return stop("provider_unavailable")
	}
	now := s.clock.Now()
	if uncertain && !canRetryAmbiguous(provider, row.FirstAttemptAt, now) {
		return stop("unsafe_to_retry")
	}
	if row.AttemptCount >= maxAttempts {
		status := db.NotificationDeliveriesStatusFailed
		if uncertain {
			status = db.NotificationDeliveriesStatusUnknown
		}
		return s.record(ctx, row, status, "attempts_exhausted", "", 0)
	}
	if row.NextAttemptAt.Valid && row.NextAttemptAt.Int64 > now.UnixMilli() {
		return attemptResult{Status: row.Status, RetryAfter: time.UnixMilli(row.NextAttemptAt.Int64).Sub(now)}, nil
	}
	plaintext, err := s.decrypt(ctx, row.WorkspaceID, row.EncryptedTargetSnapshot)
	if err != nil {
		return attemptResult{}, err
	}
	var frozen snapshot
	if json.Unmarshal([]byte(plaintext), &frozen) != nil || frozen.Event.WorkspaceID != row.WorkspaceID || frozen.Event.ID() != row.NotificationID {
		return stop("invalid_snapshot")
	}
	if frozen.Event.ExpiresAt != 0 && frozen.Event.ExpiresAt <= now.UnixMilli() {
		return stop("expired")
	}
	allowed, err := s.allowed(ctx, frozen, notification.Channel(row.Channel))
	if err != nil {
		return attemptResult{}, err
	}
	if !allowed {
		return stop("recipient_disabled")
	}
	target, err := s.target(ctx, row.WorkspaceID, frozen, notification.Channel(row.Channel))
	if err != nil {
		if restate.IsTerminalError(err) {
			return stop("invalid_snapshot")
		}
		return attemptResult{}, err
	}
	now = s.clock.Now()
	if uncertain && !canRetryAmbiguous(provider, row.FirstAttemptAt, now) {
		return stop("unsafe_to_retry")
	}
	if frozen.Event.ExpiresAt != 0 && frozen.Event.ExpiresAt <= now.UnixMilli() {
		return stop("expired")
	}
	claimed, err := s.db.StartNotificationDeliveryAttempt(ctx, db.StartNotificationDeliveryAttemptParams{
		WorkspaceID: row.WorkspaceID, ID: row.ID, Now: sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ExpectedStatus: row.Status, ExpectedAttemptCount: row.AttemptCount,
	})
	if err != nil {
		return attemptResult{}, err
	}
	if claimed != 1 {
		return attemptResult{}, errDeliveryChanged
	}
	row.Status = db.NotificationDeliveriesStatusSending
	row.AttemptCount++
	if !row.FirstAttemptAt.Valid {
		row.FirstAttemptAt = sql.NullInt64{Int64: now.UnixMilli(), Valid: true}
	}
	receipt, sendErr := provider.Send(ctx, notification.Request{DeliveryID: row.ID, Event: frozen.Event, Content: frozen.Content, Target: target})
	if sendErr == nil {
		if len(receipt.MessageID) > 256 {
			return s.record(ctx, row, db.NotificationDeliveriesStatusUnknown, "invalid_provider_receipt", "", 0)
		}
		return s.record(ctx, row, db.NotificationDeliveriesStatusAccepted, "", receipt.MessageID, 0)
	}
	return s.recordSendFailure(ctx, row, provider, uncertain, sendErr)
}

func (s *Service) recordSendFailure(ctx context.Context, row db.NotificationDelivery, provider notification.Provider, uncertain bool, sendErr error) (attemptResult, error) {
	failure := &notification.SendError{Kind: notification.Ambiguous, Code: "provider_error", RetryAfter: 0}
	var classified *notification.SendError
	if errors.As(sendErr, &classified) {
		failure = classified
	}
	if failure.Kind == notification.Rejected {
		status := db.NotificationDeliveriesStatusFailed
		if uncertain {
			status = db.NotificationDeliveriesStatusUnknown
		}
		return s.record(ctx, row, status, "provider_rejected", "", 0)
	}
	uncertain = uncertain || failure.Kind != notification.Retryable
	delay := time.Second * (1 << (row.AttemptCount - 1))
	if failure.RetryAfter > delay {
		delay = failure.RetryAfter
	}
	if delay > time.Hour {
		delay = time.Hour
	}
	status := db.NotificationDeliveriesStatusPending
	if uncertain {
		if !canRetryAmbiguous(provider, row.FirstAttemptAt, s.clock.Now().Add(delay)) {
			return s.record(ctx, row, db.NotificationDeliveriesStatusUnknown, "ambiguous_provider_result", "", 0)
		}
		status = db.NotificationDeliveriesStatusSending
	}
	return s.record(ctx, row, status, "provider_retry", "", delay)
}

func canRetryAmbiguous(provider notification.Provider, first sql.NullInt64, at time.Time) bool {
	return first.Valid && provider.IdempotencyWindow() > time.Minute &&
		at.Before(time.UnixMilli(first.Int64).Add(provider.IdempotencyWindow()-time.Minute))
}

func (s *Service) record(ctx context.Context, row db.NotificationDelivery, status db.NotificationDeliveriesStatus, reason, messageID string, retryAfter time.Duration) (attemptResult, error) {
	now := s.clock.Now()
	updated, err := s.db.UpdateNotificationDelivery(ctx, db.UpdateNotificationDeliveryParams{
		WorkspaceID:          row.WorkspaceID,
		ID:                   row.ID,
		Status:               status,
		Reason:               nullable(reason),
		ProviderMessageID:    nullable(messageID),
		NextAttemptAt:        sql.NullInt64{Int64: now.Add(retryAfter).UnixMilli(), Valid: retryAfter > 0},
		Now:                  sql.NullInt64{Int64: now.UnixMilli(), Valid: true},
		ExpectedStatus:       row.Status,
		ExpectedAttemptCount: row.AttemptCount,
	})
	if err != nil {
		return attemptResult{}, err
	}
	if updated != 1 {
		return attemptResult{}, errDeliveryChanged
	}
	return attemptResult{Status: status, RetryAfter: retryAfter}, nil
}

func (s *Service) allowed(ctx context.Context, frozen snapshot, channel notification.Channel) (bool, error) {
	valid, err := s.db.NotificationResourceBelongsToWorkspace(ctx, db.NotificationResourceBelongsToWorkspaceParams{
		WorkspaceID: frozen.Event.WorkspaceID, ProjectID: frozen.Event.Resource.ProjectID,
		AppID: frozen.Event.Resource.AppID, EnvironmentID: frozen.Event.Resource.EnvironmentID,
	})
	if err != nil || !valid {
		return false, err
	}
	if channel != notification.Email {
		return s.db.NotificationDestinationEnabled(ctx, db.NotificationDestinationEnabledParams{
			WorkspaceID: frozen.Event.WorkspaceID, ID: frozen.DestinationID, Channel: db.NotificationDestinationsChannel(channel),
		})
	}
	enabled, err := s.db.NotificationEmailEnabled(ctx, db.NotificationEmailEnabledParams{
		WorkspaceID: frozen.Event.WorkspaceID, UserID: frozen.UserID, NotificationType: frozen.Event.Type,
	})
	if err != nil || !enabled {
		return false, err
	}
	if s.directory == nil {
		return false, errors.New("notification member directory unavailable")
	}
	workspace, err := s.db.FindWorkspaceByID(ctx, frozen.Event.WorkspaceID)
	if err != nil {
		return false, err
	}
	return s.directory.CanEmail(ctx, workspace.OrgID, frozen.UserID, frozen.Email)
}

func (s *Service) decrypt(ctx context.Context, workspaceID, ciphertext string) (string, error) {
	response, err := s.vault.Decrypt(ctx, &vaultv1.DecryptRequest{Keyring: workspaceID, Encrypted: ciphertext})
	if err != nil {
		return "", errors.New("decrypt notification target")
	}
	return response.GetPlaintext(), nil
}

func (s *Service) target(ctx context.Context, workspaceID string, frozen snapshot, channel notification.Channel) (notification.Target, error) {
	target := notification.Target{Email: frozen.Email, URL: "", Headers: nil, SigningSecret: ""}
	if channel == notification.Email {
		return target, nil
	}
	config := &notificationv1.DestinationConfig{}
	if proto.Unmarshal(frozen.Config, config) != nil || !validDestination(channel, frozen.Config) {
		return target, invalid("invalid destination snapshot")
	}
	var encryptedURL string
	switch channel {
	case notification.Slack:
		encryptedURL = config.GetSlack().GetEncryptedWebhookUrl()
	case notification.Discord:
		encryptedURL = config.GetDiscord().GetEncryptedWebhookUrl()
	case notification.Webhook:
		webhook := config.GetWebhook()
		encryptedURL = webhook.GetEncryptedUrl()
		var err error
		target.SigningSecret, err = s.decrypt(ctx, workspaceID, webhook.GetEncryptedSigningSecret())
		if err != nil {
			return target, err
		}
		target.Headers = make(map[string]string, len(webhook.GetHeaders()))
		for _, header := range webhook.GetHeaders() {
			value, err := s.decrypt(ctx, workspaceID, header.GetEncryptedValue())
			if err != nil {
				return target, err
			}
			target.Headers[header.GetName()] = value
		}
	case notification.Email:
		return target, invalid("email is not a destination")
	}
	var err error
	target.URL, err = s.decrypt(ctx, workspaceID, encryptedURL)
	return target, err
}
