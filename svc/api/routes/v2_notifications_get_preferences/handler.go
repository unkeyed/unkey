package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.EmptyResponse
type Response = openapi.NotificationPreferencesResponse

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.getPreferences" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.PersonalUser(s)
	if err != nil {
		return err
	}
	res, err := h.Notifications.GetPreferences(ctx, connect.NewRequest(&ctrlv1.NotificationGetPreferencesRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
		UserId:      principal.Subject.ID,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "get personal notification preferences")
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: notificationapi.Preferences(res.Msg.GetPreferences()),
	})
}
