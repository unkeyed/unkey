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

type Request = openapi.V2NotificationsPreviewRecipientsRequestBody
type Response = openapi.V2NotificationsPreviewRecipientsResponseBody

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.previewRecipients" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.Authorize(s, permissions.Read, false)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	res, err := h.Notifications.PreviewRecipients(ctx, connect.NewRequest(&ctrlv1.NotificationPreviewRecipientsRequest{
		WorkspaceId:      principal.AuthorizedWorkspaceID,
		NotificationType: req.NotificationType,
		ProjectId:        req.ProjectId,
		AppId:            req.AppId,
		EnvironmentId:    req.EnvironmentId,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "preview notification recipients")
	}
	data := make([]openapi.NotificationRecipient, 0, len(res.Msg.GetRecipients()))
	for _, recipient := range res.Msg.GetRecipients() {
		data = append(data, openapi.NotificationRecipient{
			Key:           recipient.GetKey(),
			Channel:       recipient.GetChannel(),
			UserId:        notificationapi.OptionalString(recipient.GetUserId()),
			DestinationId: notificationapi.OptionalString(recipient.GetDestinationId()),
			Email:         notificationapi.OptionalString(recipient.GetEmail()),
			Reason:        notificationapi.OptionalString(recipient.GetReason()),
		})
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
