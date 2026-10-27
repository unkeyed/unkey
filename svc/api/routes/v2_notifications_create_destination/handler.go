package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2NotificationsCreateDestinationRequestBody
type Response = openapi.NotificationDestinationResponse

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.createDestination" }

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
	if req.Channel == openapi.Webhook && req.SigningSecret == nil {
		return fault.New("webhook signing secret is required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Public("Webhook destinations require a signing secret of at least 32 characters."),
		)
	}
	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}
	res, err := h.Notifications.CreateDestination(ctx, connect.NewRequest(&ctrlv1.NotificationCreateDestinationRequest{
		WorkspaceId:    principal.AuthorizedWorkspaceID,
		Name:           req.Name,
		Channel:        string(req.Channel),
		Url:            ptr.SafeDeref(req.Url, ""),
		Headers:        notificationapi.Headers(req.Headers).GetItems(),
		SigningSecret:  ptr.SafeDeref(req.SigningSecret, ""),
		Enabled:        ptr.SafeDeref(req.Enabled, true),
		IdempotencyKey: req.IdempotencyKey,
		Actor:          actor,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "create notification destination")
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
