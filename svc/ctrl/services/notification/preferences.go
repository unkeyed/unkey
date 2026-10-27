package notification

import (
	"context"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (s *Service) GetPreferences(ctx context.Context, req *connect.Request[ctrlv1.NotificationGetPreferencesRequest]) (*connect.Response[ctrlv1.NotificationGetPreferencesResponse], error) {
	if err := s.authorize(ctx, req, req.Msg.GetWorkspaceId()); err != nil {
		return nil, err
	}
	if !validID(req.Msg.GetUserId(), 256) {
		return nil, rpcError(connect.CodeInvalidArgument, "user_id is required")
	}
	if err := s.requireMembers(ctx, s.db, req.Msg.GetWorkspaceId(), []string{req.Msg.GetUserId()}); err != nil {
		return nil, err
	}
	preferences, err := readPreferences(ctx, s.db, req.Msg.GetWorkspaceId(), req.Msg.GetUserId())
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationGetPreferencesResponse{Preferences: preferences}), nil
}

func (s *Service) UpdatePreferences(ctx context.Context, req *connect.Request[ctrlv1.NotificationUpdatePreferencesRequest]) (*connect.Response[ctrlv1.NotificationUpdatePreferencesResponse], error) {
	message := req.Msg
	if err := s.authorize(ctx, req, message.GetWorkspaceId()); err != nil {
		return nil, err
	}
	if !validID(message.GetUserId(), 256) || message.GetActor().GetType() != ctrlv1.ActorType_ACTOR_TYPE_USER || message.GetActor().GetId() != message.GetUserId() {
		return nil, rpcError(connect.CodeInvalidArgument, "preferences may only be changed by their user")
	}
	if len(message.GetPreferences()) == 0 || len(message.GetPreferences()) > len(eventTypes()) {
		return nil, rpcError(connect.CodeInvalidArgument, "invalid preference batch size")
	}
	seen := make(map[string]bool)
	for _, preference := range message.GetPreferences() {
		if err := validateType(preference.GetNotificationType()); err != nil {
			return nil, err
		}
		if seen[preference.GetNotificationType()] {
			return nil, rpcError(connect.CodeInvalidArgument, "duplicate notification preference")
		}
		seen[preference.GetNotificationType()] = true
	}
	var preferences []*ctrlv1.NotificationPreference
	err := s.mutate(ctx, message.GetWorkspaceId(), message.GetActor(), func(ctx context.Context, tx db.DBTX, queries *db.Queries) error {
		if err := s.requireMembers(ctx, queries, message.GetWorkspaceId(), []string{message.GetUserId()}); err != nil {
			return err
		}
		for _, preference := range message.GetPreferences() {
			if err := queries.UpsertNotificationPreference(ctx, db.UpsertNotificationPreferenceParams{
				ID:               stableID(message.GetWorkspaceId(), "preference", message.GetUserId()+"\x00"+preference.GetNotificationType()),
				WorkspaceID:      message.GetWorkspaceId(),
				UserID:           message.GetUserId(),
				NotificationType: preference.GetNotificationType(),
				Enabled:          preference.GetEnabled(),
				CreatedAt:        s.clock.Now().UnixMilli(),
			}); err != nil {
				return err
			}
		}
		if err := s.audit(ctx, tx, message.GetWorkspaceId(), message.GetActor(), "preference", "update", message.GetUserId()); err != nil {
			return err
		}
		var err error
		preferences, err = readPreferences(ctx, queries, message.GetWorkspaceId(), message.GetUserId())
		return err
	})
	if err != nil {
		return nil, err
	}
	return connect.NewResponse(&ctrlv1.NotificationUpdatePreferencesResponse{Preferences: preferences}), nil
}

func readPreferences(ctx context.Context, queries db.Querier, workspaceID, userID string) ([]*ctrlv1.NotificationPreference, error) {
	rows, err := queries.ListNotificationUserPreferences(ctx, db.ListNotificationUserPreferencesParams{WorkspaceID: workspaceID, UserID: userID})
	if err != nil {
		return nil, operationError(err)
	}
	if len(rows) > 100 {
		return nil, rpcError(connect.CodeFailedPrecondition, "notification preference limit exceeded")
	}
	enabled := make(map[string]bool, len(rows))
	for _, row := range rows {
		enabled[row.NotificationType] = row.Enabled
	}
	preferences := make([]*ctrlv1.NotificationPreference, 0, len(eventTypes()))
	for _, eventType := range eventTypes() {
		value, exists := enabled[eventType]
		preferences = append(preferences, &ctrlv1.NotificationPreference{NotificationType: eventType, Enabled: !exists || value})
	}
	return preferences, nil
}
