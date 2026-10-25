package notification

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sort"

	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/workos"
	"google.golang.org/protobuf/proto"
)

const maxRecipients = 1000

type snapshot struct {
	Event         notification.Event   `json:"event"`
	Content       notification.Content `json:"content"`
	Email         string               `json:"email,omitempty"`
	Config        []byte               `json:"config,omitempty"`
	UserID        string               `json:"userId,omitempty"`
	DestinationID string               `json:"destinationId,omitempty"`
}

type recipient struct {
	key     string
	channel notification.Channel
	email   string
	config  []byte
	reason  string
}

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
		WorkspaceID: event.WorkspaceID, ProjectID: event.Resource.ProjectID, AppID: event.Resource.AppID, EnvironmentID: event.Resource.EnvironmentID,
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
			ID: event.ID(), WorkspaceID: event.WorkspaceID, SourceEventID: event.SourceEventID, NotificationType: event.Type,
			RequestHash: hash, SchemaVersion: event.SchemaVersion, ResourceContext: resource, Payload: event.Payload,
			OccurredAt: event.OccurredAt, ExpiresAt: sql.NullInt64{Int64: event.ExpiresAt, Valid: event.ExpiresAt != 0}, CreatedAt: now,
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

func (s *Service) resolve(ctx context.Context, event notification.Event) ([]recipient, error) {
	targets, err := s.db.ListNotificationTargets(ctx, db.ListNotificationTargetsParams{
		WorkspaceID: event.WorkspaceID, NotificationType: event.Type, ProjectID: nullable(event.Resource.ProjectID),
		AppID: nullable(event.Resource.AppID), EnvironmentID: nullable(event.Resource.EnvironmentID),
	})
	if err != nil {
		return nil, err
	}
	if len(targets) > maxRecipients {
		return nil, invalid("notification target limit exceeded")
	}
	needsMembers := false
	needsAudience := false
	var userIDs []string
	for _, target := range targets {
		if target.Channel != db.NotificationSubscriptionTargetsChannelEmail {
			continue
		}
		needsMembers = true
		needsAudience = needsAudience || target.Audience.Valid
		if target.UserID.Valid {
			userIDs = append(userIDs, target.UserID.String)
		}
	}
	var members []workos.Member
	var preferences []db.ListNotificationPreferencesRow
	if needsMembers {
		if s.directory == nil {
			return nil, invalid("notification member directory is not configured")
		}
		workspace, err := s.db.FindWorkspaceByID(ctx, event.WorkspaceID)
		if err != nil {
			return nil, err
		}
		if needsAudience {
			members, err = s.directory.Members(ctx, workspace.OrgID)
		} else {
			members, err = s.directory.LookupMembers(ctx, workspace.OrgID, userIDs)
		}
		if errors.Is(err, workos.ErrDirectoryLimit) {
			return nil, invalid("notification member directory limit exceeded")
		}
		if err != nil {
			return nil, err
		}
		preferences, err = s.db.ListNotificationPreferences(ctx, db.ListNotificationPreferencesParams{WorkspaceID: event.WorkspaceID, NotificationType: event.Type})
		if err != nil {
			return nil, err
		}
		if len(preferences) > 10000 {
			return nil, invalid("notification preference limit exceeded")
		}
	}
	audiences := make(map[notificationv1.Audience][]string)
	for _, target := range targets {
		if !target.Audience.Valid {
			continue
		}
		audience := notificationv1.Audience(target.Audience.Int32)
		if _, found := audiences[audience]; found {
			continue
		}
		resolver := s.audiences[audience]
		if resolver == nil {
			return nil, invalid("notification audience has no resolver")
		}
		ids, err := resolver(ctx, event.WorkspaceID, members)
		if err != nil {
			return nil, err
		}
		audiences[audience] = ids
	}
	return resolveTargets(targets, members, preferences, audiences)
}

func resolveTargets(targets []db.ListNotificationTargetsRow, members []workos.Member, preferences []db.ListNotificationPreferencesRow, audiences map[notificationv1.Audience][]string) ([]recipient, error) {
	memberByID := make(map[string]workos.Member, len(members))
	for _, member := range members {
		memberByID[member.UserID] = member
	}
	optedOut := make(map[string]bool, len(preferences))
	for _, preference := range preferences {
		optedOut[preference.UserID] = !preference.Enabled
	}
	unique := make(map[string]recipient)
	for _, target := range targets {
		channel := notification.Channel(target.Channel)
		if channel != notification.Email {
			key := "destination:" + target.DestinationID.String
			reason := ""
			if !target.DestinationEnabled.Valid {
				reason = "destination_unavailable"
			} else if !target.DestinationEnabled.Bool {
				reason = "destination_paused"
			} else if !validDestination(channel, []byte(target.DestinationConfig.String)) {
				reason = "invalid_destination_config"
			}
			unique[string(channel)+":"+key] = recipient{key: key, channel: channel, email: "", config: []byte(target.DestinationConfig.String), reason: reason}
			continue
		}
		ids := []string{target.UserID.String}
		if target.Audience.Valid {
			ids = audiences[notificationv1.Audience(target.Audience.Int32)]
		}
		for _, userID := range ids {
			member, found := memberByID[userID]
			reason := ""
			if !found {
				reason = "member_unavailable"
			} else if optedOut[userID] {
				reason = "email_opted_out"
			}
			key := "user:" + userID
			unique["email:"+key] = recipient{key: key, channel: channel, email: member.Email, config: nil, reason: reason}
		}
	}
	if len(unique) > maxRecipients {
		return nil, invalid("notification recipient limit exceeded")
	}
	result := make([]recipient, 0, len(unique))
	for _, target := range unique {
		result = append(result, target)
	}
	sort.Slice(result, func(i, j int) bool {
		return string(result[i].channel)+result[i].key < string(result[j].channel)+result[j].key
	})
	return result, nil
}

func validDestination(channel notification.Channel, data []byte) bool {
	if len(data) > 64*1024 {
		return false
	}
	config := &notificationv1.DestinationConfig{}
	if proto.Unmarshal(data, config) != nil {
		return false
	}
	switch channel {
	case notification.Discord:
		return config.GetDiscord().GetEncryptedWebhookUrl() != ""
	case notification.Slack:
		return config.GetSlack().GetEncryptedWebhookUrl() != ""
	case notification.Webhook:
		return config.GetWebhook().GetEncryptedUrl() != "" && config.GetWebhook().GetEncryptedSigningSecret() != ""
	case notification.Email:
		return false
	}
	return false
}

func WorkspaceAdmins(_ context.Context, _ string, members []workos.Member) ([]string, error) {
	ids := make([]string, 0)
	for _, member := range members {
		if member.Admin {
			ids = append(ids, member.UserID)
		}
	}
	return ids, nil
}

func nullable(value string) sql.NullString {
	return sql.NullString{String: value, Valid: value != ""}
}
