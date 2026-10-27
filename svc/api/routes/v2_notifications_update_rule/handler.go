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
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/notificationapi"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2NotificationsUpdateRuleRequestBody
type Response = openapi.NotificationRuleResponse

type Handler struct {
	Notifications ctrlv1connect.NotificationServiceClient
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/notifications.updateRule" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := notificationapi.Authorize(s, permissions.Write, false)
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	targets, err := notificationapi.Targets(req.Targets)
	if err != nil {
		return err
	}
	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}
	res, err := h.Notifications.UpdateRule(ctx, connect.NewRequest(&ctrlv1.NotificationUpdateRuleRequest{
		WorkspaceId:      principal.AuthorizedWorkspaceID,
		RuleId:           req.RuleId,
		Name:             req.Name,
		NotificationType: req.NotificationType,
		ProjectId:        ptr.SafeDeref(req.ProjectId, ""),
		AppId:            ptr.SafeDeref(req.AppId, ""),
		EnvironmentId:    ptr.SafeDeref(req.EnvironmentId, ""),
		Enabled:          req.Enabled,
		Targets:          targets,
		Actor:            actor,
	}))
	if err != nil {
		return notificationapi.HandleError(err, "update notification rule")
	}
	data, err := notificationapi.Rule(res.Msg.GetRule())
	if err != nil {
		return err
	}
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
