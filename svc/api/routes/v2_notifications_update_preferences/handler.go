package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2NotificationsUpdatePreferencesRequestBody
type Response = openapi.NotificationPreferencesResponse

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.updatePreferences" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.PersonalUser(s)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}
	preferences := make([]*ctrlv1.NotificationPreference, 0, len(req.Preferences))
	for _, preference := range req.Preferences {
		preferences = append(preferences, &ctrlv1.NotificationPreference{
			NotificationType: preference.NotificationType,
			Enabled:          preference.Enabled,
		})
	}
	res, err := h.Notifications.UpdatePreferences(ctx, connect.NewRequest(&ctrlv1.NotificationUpdatePreferencesRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
		UserId:      principal.Subject.ID,
		Preferences: preferences,
		Actor:       actor,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "update personal notification preferences")
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: notificationapi.Preferences(res.Msg.GetPreferences()),
	})
}
