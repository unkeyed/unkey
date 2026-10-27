package handler

import (
	"context"
	"net/http"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.NotificationListRequest
type Response = openapi.V2NotificationsListDestinationsResponseBody

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.listDestinations" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.Authorize(s, permissions.Read, false)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	res, err := h.Notifications.ListDestinations(ctx, connect.NewRequest(&ctrlv1.NotificationListDestinationsRequest{
		WorkspaceId: principal.AuthorizedWorkspaceID,
		Cursor:      ptr.SafeDeref(req.Cursor, ""),
		Limit:       int32(ptr.SafeDeref(req.Limit, 50)),
	}))
	if err != nil {
		return notificationapi.HandleError(err, "list notification destinations")
	}
	data := make([]openapi.NotificationDestination, 0, len(res.Msg.GetDestinations()))
	for _, value := range res.Msg.GetDestinations() {
		destination, err := notificationapi.Destination(value)
		if err != nil {
			return err
		}
		data = append(data, destination)
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
		Pagination: openapi.NotificationPagination{
			NextCursor: notificationapi.OptionalString(res.Msg.GetNextCursor()),
			HasMore:    res.Msg.GetHasMore(),
		},
	})
}
