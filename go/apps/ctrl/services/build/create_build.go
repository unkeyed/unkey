package build

import (
	"context"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/ctrl/workflows"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) CreateBuild(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateBuildRequest],
) (*connect.Response[ctrlv1.CreateBuildResponse], error) {
	buildID := uid.New(uid.BuildPrefix)
	now := time.Now().UnixMilli()

	// Insert build into database
	err := db.Query.InsertBuild(ctx, s.db.RW(), db.InsertBuildParams{
		ID:          buildID,
		WorkspaceID: req.Msg.GetWorkspaceId(),
		ProjectID:   req.Msg.GetProjectId(),
		VersionID:   req.Msg.GetVersionId(),
		CreatedAt:   now,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Start the build workflow asynchronously
	go func() {
		// Create a new background context for the async workflow
		// This prevents cancellation when the HTTP request completes
		workflowCtx := context.Background()

		buildWorkflow := workflows.Build{
			Logger:         s.logger,
			DB:             s.db,
			BuilderService: s.builderService,
		}

		buildReq := workflows.BuildRequest{
			WorkspaceID: req.Msg.GetWorkspaceId(),
			ProjectID:   req.Msg.GetProjectId(),
			VersionID:   req.Msg.GetVersionId(),
			DockerImage: req.Msg.GetDockerImage(),
		}

		// Run the build workflow with background context
		if err := buildWorkflow.Run(workflowCtx, buildReq); err != nil {
			s.logger.Error("Build workflow failed", "build_id", buildID, "error", err)
		}
	}()

	res := connect.NewResponse(&ctrlv1.CreateBuildResponse{
		BuildId: buildID,
	})

	return res, nil
}
