package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func (s *Service) CreateVersion(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateVersionRequest],
) (*connect.Response[ctrlv1.CreateVersionResponse], error) {
	// Validate workspace exists
	_, err := db.Query.FindWorkspaceByID(ctx, s.db.RO(), req.Msg.GetWorkspaceId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("workspace not found: %s", req.Msg.GetWorkspaceId()))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Validate project exists and belongs to workspace
	project, err := db.Query.FindProjectById(ctx, s.db.RO(), req.Msg.GetProjectId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("project not found: %s", req.Msg.GetProjectId()))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	// Verify project belongs to the specified workspace
	if project.WorkspaceID != req.Msg.GetWorkspaceId() {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("project %s does not belong to workspace %s",
				req.Msg.GetProjectId(), req.Msg.GetWorkspaceId()))
	}

	// Get git branch name for the deployment
	gitBranch := req.Msg.GetBranch()
	if gitBranch == "" {
		gitBranch = project.DefaultBranch.String
		if gitBranch == "" {
			gitBranch = "main" // fallback default
		}
	}

	// Determine environment (default to preview)
	// TODO: Add environment field to CreateVersionRequest proto
	environment := db.DeploymentsEnvironmentPreview

	// Generate deployment ID
	deploymentID := uid.New("deployment")
	now := time.Now().UnixMilli()

	// Insert deployment into database
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:             deploymentID,
		WorkspaceID:    req.Msg.GetWorkspaceId(),
		ProjectID:      req.Msg.GetProjectId(),
		Environment:    environment,
		BuildID:        sql.NullString{String: "", Valid: false}, // Build creation handled separately
		RootfsImageID:  "",                                       // Image handling not implemented yet
		GitCommitSha:   sql.NullString{String: req.Msg.GetGitCommitSha(), Valid: req.Msg.GetGitCommitSha() != ""},
		GitBranch:      sql.NullString{String: gitBranch, Valid: true},
		ConfigSnapshot: []byte("{}"), // Configuration snapshot placeholder
		OpenapiSpec:    sql.NullString{String: "", Valid: false},
		Status:         "pending",
		CreatedAt:      now,
		UpdatedAt:      sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.logger.Info("starting deployment workflow for deployment",
		"deployment_id", deploymentID,
		"workspace_id", req.Msg.GetWorkspaceId(),
		"project_id", req.Msg.GetProjectId(),
		"environment", environment,
		"docker_image", req.Msg.GetDockerImageTag())

	// Start the deployment workflow directly
	deployReq := &DeployRequest{
		WorkspaceID:  req.Msg.GetWorkspaceId(),
		ProjectID:    req.Msg.GetProjectId(),
		DeploymentID: deploymentID,
		DockerImage:  req.Msg.GetDockerImageTag(),
		KeyspaceID:   req.Msg.GetKeyspaceId(),
		Hostname:     req.Msg.GetHostname(),
	}

	executionID, err := s.hydraEngine.StartWorkflow(ctx, "deployment", deployReq,
		hydra.WithMaxAttempts(3),
		hydra.WithTimeout(25*time.Minute),
		hydra.WithRetryBackoff(1*time.Minute),
	)
	if err != nil {
		s.logger.Error("failed to start deployment workflow",
			"deployment_id", deploymentID,
			"error", err)
		// Don't fail deployment creation - workflow can be retried
	} else {
		s.logger.Info("deployment workflow started",
			"deployment_id", deploymentID,
			"execution_id", executionID)
	}

	res := connect.NewResponse(&ctrlv1.CreateVersionResponse{
		VersionId: deploymentID,
		Status:    ctrlv1.VersionStatus_VERSION_STATUS_PENDING,
	})

	return res, nil
}
