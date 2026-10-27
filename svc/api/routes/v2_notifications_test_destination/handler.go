package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2NotificationsTestDestinationRequestBody
type Response = openapi.V2NotificationsTestDestinationResponseBody

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.testDestination" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.Authorize(s, permissions.Write, false)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	if err := notificationapi.ValidateIdempotencyKey(req.IdempotencyKey); err != nil {
		return err
	}
	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}
	res, err := h.Notifications.TestDestination(ctx, connect.NewRequest(&ctrlv1.NotificationTestDestinationRequest{
		WorkspaceId:    principal.AuthorizedWorkspaceID,
		DestinationId:  req.DestinationId,
		IdempotencyKey: req.IdempotencyKey,
		Actor:          actor,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "test notification destination")
	}
	var response Response
	response.Meta = openapi.Meta{RequestId: s.RequestID()}
	response.Data.DeliveryId = res.Msg.GetDeliveryId()
	response.Data.Status = openapi.V2NotificationsTestDestinationResponseBodyDataStatus(res.Msg.GetStatus())
	if !response.Data.Status.Valid() {
		return fault.New("invalid notification delivery status",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("Unable to read the notification delivery status."),
		)
	}
	response.Data.Reason = notificationapi.OptionalString(res.Msg.GetReason())
	return s.JSON(http.StatusOK, response)
}
