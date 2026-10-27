package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.NotificationDestinationRequest
type Response = openapi.NotificationDestinationResponse

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.getDestination" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.Authorize(s, permissions.Read, true)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	res, err := h.Notifications.GetDestination(ctx, connect.NewRequest(&ctrlv1.NotificationGetDestinationRequest{
		WorkspaceId:   principal.AuthorizedWorkspaceID,
		DestinationId: req.DestinationId,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "get notification destination")
	}
	data, err := notificationapi.Destination(res.Msg.GetDestination())
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
