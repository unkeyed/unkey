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

type Request = openapi.NotificationDestinationRequest
type Response = openapi.NotificationEmptyResponse

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.deleteDestination" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.Authorize(s, permissions.Write, false)
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
	_, err = h.Notifications.DeleteDestination(ctx, connect.NewRequest(&ctrlv1.NotificationDeleteDestinationRequest{
		WorkspaceId:   principal.AuthorizedWorkspaceID,
		DestinationId: req.DestinationId,
		Actor:         actor,
	}))
	if err != nil {
		if connect.CodeOf(err) == connect.CodeFailedPrecondition {
			return fault.New("notification destination is referenced by a rule",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Public("Remove this destination from all notification rules before deleting it."),
			)
		}
		return notificationapi.HandleError(err, "delete notification destination")
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.EmptyResponse{},
	})
}
