package app

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// DeleteApp enqueues a durable Restate workflow that cascades through
// the app's environments and cleans up all associated resources.
// Returns immediately after the workflow is enqueued; actual deletion
// is eventually consistent.
func (s *Service) DeleteApp(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteAppRequest],
) (*connect.Response[ctrlv1.DeleteAppResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetAppId(), "app_id is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	app, err := db.Query.FindAppById(ctx, s.db.RO(), req.Msg.GetAppId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("app not found: %s", req.Msg.GetAppId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find app: %w", err))
	}

	client := hydrav1.NewAppServiceIngressClient(s.restate, req.Msg.GetAppId())
	_, err = client.Delete().Send(ctx, &hydrav1.DeleteAppRequest{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger app deletion: %w", err))
	}

	a := req.Msg.GetActor()
	err = s.auditlogs.Insert(ctx, nil, []auditlog.AuditLog{
		{
			WorkspaceID:   app.WorkspaceID,
			Event:         auditlog.AppDeleteEvent,
			Display:       fmt.Sprintf("Deleted app %s", app.ID),
			ActorID:       a.GetId(),
			ActorName:     a.GetName(),
			ActorType:     actor.AuditType(a.GetType()),
			ActorMeta:     actor.Meta(a.GetMeta()),
			RemoteIP:      a.GetRemoteIp(),
			UserAgent:     a.GetUserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					ID:          app.ID,
					Type:        auditlog.AppResourceType,
					Meta:        map[string]any{"name": app.Name, "slug": app.Slug, "projectId": app.ProjectID},
					Name:        app.Name,
					DisplayName: app.Name,
				},
			},
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert audit log: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteAppResponse{}), nil
}
