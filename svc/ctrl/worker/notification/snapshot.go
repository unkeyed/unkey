package notification

import (
	"context"
	"encoding/json"
	"errors"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type snapshot struct {
	Event         notification.Event   `json:"event"`
	Content       notification.Content `json:"content"`
	Email         string               `json:"email,omitempty"`
	Config        []byte               `json:"config,omitempty"`
	UserID        string               `json:"userId,omitempty"`
	DestinationID string               `json:"destinationId,omitempty"`
}

func (s *Service) freezeDelivery(ctx context.Context, event notification.Event, content notification.Content, recipient recipient, createdAtMillis int64) (db.InsertNotificationDeliveryParams, error) {
	row := db.InsertNotificationDeliveryParams{
		ID:                      notification.DeliveryID(event.ID(), recipient.channel, recipient.key),
		WorkspaceID:             event.WorkspaceID,
		NotificationID:          event.ID(),
		Channel:                 db.NotificationDeliveriesChannel(recipient.channel),
		TargetKey:               recipient.key,
		EncryptedTargetSnapshot: "",
		Provider:                nullable(s.defaults[recipient.channel]),
		Status:                  db.NotificationDeliveriesStatusPending,
		Reason:                  nullable(recipient.reason),
		CreatedAt:               createdAtMillis,
	}
	if event.ExpiresAt != 0 && event.ExpiresAt <= createdAtMillis {
		row.Reason = nullable("expired")
	}
	if row.Reason.Valid {
		row.Status = db.NotificationDeliveriesStatusSuppressed
		return row, nil
	}
	if !row.Provider.Valid {
		row.Status = db.NotificationDeliveriesStatusFailed
		row.Reason = nullable("provider_unavailable")
		return row, nil
	}

	frozen := snapshot{
		Event:         event,
		Content:       content,
		Email:         recipient.email,
		Config:        recipient.config,
		UserID:        "",
		DestinationID: "",
	}
	if recipient.channel == notification.Email {
		frozen.UserID = recipient.key[len("user:"):]
	} else {
		frozen.DestinationID = recipient.key[len("destination:"):]
	}
	encoded, err := json.Marshal(frozen)
	if err != nil {
		return db.InsertNotificationDeliveryParams{}, err
	}
	encrypted, err := s.vault.Encrypt(ctx, &vaultv1.EncryptRequest{Keyring: event.WorkspaceID, Data: string(encoded)})
	if err != nil {
		return db.InsertNotificationDeliveryParams{}, errors.New("encrypt notification snapshot")
	}
	row.EncryptedTargetSnapshot = encrypted.GetEncrypted()
	return row, nil
}
