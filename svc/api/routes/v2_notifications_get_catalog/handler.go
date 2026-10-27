package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	notificationv1 "github.com/unkeyed/unkey/gen/proto/notification/v1"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.EmptyResponse
type Response = openapi.V2NotificationsGetCatalogResponseBody

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.getCatalog" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	res, err := h.Notifications.GetCatalog(ctx, connect.NewRequest(&ctrlv1.NotificationGetCatalogRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "get notification catalog")
	}
	var response Response
	response.Meta = openapi.Meta{RequestId: s.RequestID()}
	response.Data.EventTypes = append([]string{}, res.Msg.GetEventTypes()...)
	response.Data.Channels = append([]string{}, res.Msg.GetChannels()...)
	response.Data.Audiences = []openapi.V2NotificationsGetCatalogResponseBodyDataAudiences{}
	for _, audience := range res.Msg.GetAudiences() {
		if audience == notificationv1.Audience_AUDIENCE_WORKSPACE_ADMINS {
			response.Data.Audiences = append(response.Data.Audiences, openapi.V2NotificationsGetCatalogResponseBodyDataAudiencesWorkspaceAdmins)
		}
	}
	return s.JSON(http.StatusOK, response)
}
