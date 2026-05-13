package project

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
)

// CreateProject creates a project and a default app with environments
// by delegating app creation to [app.Service.CreateApp]. The caller's
// Authorization header is forwarded to the internal CreateApp call.
func (s *Service) CreateProject(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateProjectRequest],
) (*connect.Response[ctrlv1.CreateProjectResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetWorkspaceId(), "workspace_id is required"),
		assert.NotEmpty(req.Msg.GetName(), "name is required"),
		assert.NotEmpty(req.Msg.GetSlug(), "slug is required"),
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	workspaceID := req.Msg.GetWorkspaceId()
	projectID := uid.New(uid.ProjectPrefix)
	now := time.Now().UnixMilli()

	err := db.Query.InsertProject(ctx, s.db.RW(), db.InsertProjectParams{
		ID:               projectID,
		WorkspaceID:      workspaceID,
		Name:             req.Msg.GetName(),
		Slug:             req.Msg.GetSlug(),
		DeleteProtection: sql.NullBool{Valid: false},
		CreatedAt:        now,
		UpdatedAt:        sql.NullInt64{Valid: false},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to insert project: %w", err))
	}

	appReq := connect.NewRequest(&ctrlv1.CreateAppRequest{
		WorkspaceId: workspaceID,
		ProjectId:   projectID,
		Name:        req.Msg.GetName(),
		Slug:        "default",
	})
	// Forward the caller's Authorization header so the internal CreateApp
	// call passes authentication.
	appReq.Header().Set("Authorization", req.Header().Get("Authorization"))
	_, err = s.appService.CreateApp(ctx, appReq)
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.CreateProjectResponse{
		Id: projectID,
	}), nil
}
