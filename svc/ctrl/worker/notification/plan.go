package notification

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) plan(ctx context.Context, event notification.Event) ([]deliveryRef, error) {
	event, err := event.Normalize()
	if err != nil {
		return nil, invalid("invalid payload")
	}
	hash, err := event.Fingerprint()
	if err != nil {
		return nil, invalid("invalid payload")
	}
	existing, err := s.db.FindNotification(ctx, db.FindNotificationParams{WorkspaceID: event.WorkspaceID, ID: event.ID()})
	if err == nil {
		if existing.RequestHash != hash {
			return nil, invalid("event identity reused with different contents")
		}
		return s.pending(ctx, event)
	}
	if !db.IsNotFound(err) {
		return nil, err
	}

	definition, ok := s.definitions[event.Type]
	if !ok {
		return nil, invalid("unregistered notification type")
	}
	content, err := definition.Render(event)
	if err != nil {
		return nil, invalid("payload rejected by notification definition")
	}
	if len(content.Subject) == 0 || len(content.Subject) > 200 || len(content.Text) == 0 || len(content.Text)+len(content.HTML) > 64*1024 {
		return nil, invalid("invalid rendered notification")
	}
	valid, err := s.db.NotificationResourceBelongsToWorkspace(ctx, db.NotificationResourceBelongsToWorkspaceParams{
		WorkspaceID:   event.WorkspaceID,
		ProjectID:     event.Resource.ProjectID,
		AppID:         event.Resource.AppID,
		EnvironmentID: event.Resource.EnvironmentID,
	})
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, invalid("workspace or resource unavailable")
	}

	recipients, err := s.resolve(ctx, event)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now().UnixMilli()
	deliveries := make([]db.InsertNotificationDeliveryParams, 0, len(recipients))
	for _, recipient := range recipients {
		row, err := s.freezeDelivery(ctx, event, content, recipient, now)
		if err != nil {
			return nil, err
		}
		deliveries = append(deliveries, row)
	}

	resource, err := json.Marshal(event.Resource)
	if err != nil {
		return nil, err
	}
	err = db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		queries := db.NewQueries(tx)
		if err := queries.InsertNotification(txCtx, db.InsertNotificationParams{
			ID:               event.ID(),
			WorkspaceID:      event.WorkspaceID,
			SourceEventID:    event.SourceEventID,
			NotificationType: event.Type,
			RequestHash:      hash,
			SchemaVersion:    event.SchemaVersion,
			ResourceContext:  resource,
			Payload:          event.Payload,
			OccurredAt:       event.OccurredAt,
			ExpiresAt:        sql.NullInt64{Int64: event.ExpiresAt, Valid: event.ExpiresAt != 0},
			CreatedAt:        now,
		}); err != nil {
			return err
		}
		for _, delivery := range deliveries {
			if err := queries.InsertNotificationDelivery(txCtx, delivery); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.pending(ctx, event)
}

func (s *Service) pending(ctx context.Context, event notification.Event) ([]deliveryRef, error) {
	rows, err := s.db.ListPendingNotificationDeliveries(ctx, db.ListPendingNotificationDeliveriesParams{WorkspaceID: event.WorkspaceID, NotificationID: event.ID()})
	if err != nil {
		return nil, err
	}
	refs := make([]deliveryRef, 0, len(rows))
	for _, row := range rows {
		refs = append(refs, deliveryRef{WorkspaceID: row.WorkspaceID, ID: row.ID})
	}
	return refs, nil
}
