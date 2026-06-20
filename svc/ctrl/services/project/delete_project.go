package project

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

// DeleteProject enqueues a durable Restate workflow that cascades through
// the project's apps and environments, cleaning up all associated resources.
// Returns immediately after the workflow is enqueued; actual deletion
// is eventually consistent.
func (s *Service) DeleteProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.DeleteProjectRequest],
) (*connect.Response[ctrlv1.DeleteProjectResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetProjectId(), "project_id is required"),
		assert.NotNil(req.Msg.GetActor(), "actor is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	project, err := db.Query.FindProjectById(ctx, s.db.RO(), req.Msg.GetProjectId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("project not found: %s", req.Msg.GetProjectId()))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find project: %w", err))
	}

	client := hydrav1.NewProjectServiceIngressClient(s.restate, req.Msg.GetProjectId())
	_, err = client.Delete().Send(ctx, &hydrav1.DeleteProjectRequest{})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger project deletion: %w", err))
	}

	a := req.Msg.GetActor()
	err = s.auditlogs.Insert(ctx, nil, []auditlog.AuditLog{
		{
			WorkspaceID:   project.WorkspaceID,
			Event:         auditlog.ProjectDeleteEvent,
			Display:       fmt.Sprintf("Deleted project %s", project.ID),
			ActorID:       a.GetId(),
			ActorName:     a.GetName(),
			ActorType:     actor.AuditType(a.GetType()),
			ActorMeta:     actor.Meta(a.GetMeta()),
			RemoteIP:      a.GetRemoteIp(),
			UserAgent:     a.GetUserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					ID:          project.ID,
					Type:        auditlog.ProjectResourceType,
					Meta:        map[string]any{"name": project.Name, "slug": project.Slug},
					Name:        project.Name,
					DisplayName: project.Name,
				},
			},
		},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert audit log: %w", err))
	}

	return connect.NewResponse(&ctrlv1.DeleteProjectResponse{}), nil
}
