package version

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

	// Create or find branch record
	branchName := req.Msg.GetBranch()
	if branchName == "" {
		branchName = project.DefaultBranch.String
		if branchName == "" {
			branchName = "main" // fallback default
		}
	}

	var branchID string
	branch, err := db.Query.FindBranchByProjectName(ctx, s.db.RO(), db.FindBranchByProjectNameParams{
		ProjectID: req.Msg.GetProjectId(),
		Name:      branchName,
	})
	if err != nil {
		if db.IsNotFound(err) {
			// Branch doesn't exist, create it
			branchID = uid.New("branch")
			err = db.Query.InsertBranch(ctx, s.db.RW(), db.InsertBranchParams{
				ID:          branchID,
				WorkspaceID: req.Msg.GetWorkspaceId(),
				ProjectID:   req.Msg.GetProjectId(),
				Name:        branchName,
				CreatedAt:   time.Now().UnixMilli(),
				UpdatedAt:   sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
			})
			if err != nil {
				return nil, connect.NewError(connect.CodeInternal,
					fmt.Errorf("failed to create branch: %w", err))
			}
			s.logger.Info("created new branch", "branch_id", branchID, "name", branchName, "project_id", req.Msg.GetProjectId())
		} else {
			return nil, connect.NewError(connect.CodeInternal, err)
		}
	} else {
		// Branch exists, use it
		branchID = branch.ID
		s.logger.Info("using existing branch", "branch_id", branchID, "name", branchName, "project_id", req.Msg.GetProjectId())
	}

	// Generate version ID
	versionID := uid.New(uid.VersionPrefix, 4)
	now := time.Now().UnixMilli()

	// Insert version into database
	err = db.Query.InsertVersion(ctx, s.db.RW(), db.InsertVersionParams{
		ID:             versionID,
		WorkspaceID:    req.Msg.GetWorkspaceId(),
		ProjectID:      req.Msg.GetProjectId(),
		BranchID:       sql.NullString{String: branchID, Valid: true},
		BuildID:        sql.NullString{String: "", Valid: false}, // Build creation handled separately
		RootfsImageID:  "",                                       // Image handling not implemented yet
		GitCommitSha:   sql.NullString{String: req.Msg.GetGitCommitSha(), Valid: req.Msg.GetGitCommitSha() != ""},
		GitBranch:      sql.NullString{String: branchName, Valid: true},
		ConfigSnapshot: []byte("{}"), // Configuration snapshot placeholder
		Status:         db.VersionsStatusPending,
		CreatedAt:      now,
		UpdatedAt:      sql.NullInt64{Int64: now, Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.logger.Info("starting deployment workflow for version",
		"version_id", versionID,
		"workspace_id", req.Msg.GetWorkspaceId(),
		"project_id", req.Msg.GetProjectId(),
		"docker_image", req.Msg.GetDockerImageTag())

	// Start the deployment workflow directly
	deployReq := &DeployRequest{
		WorkspaceID: req.Msg.GetWorkspaceId(),
		ProjectID:   req.Msg.GetProjectId(),
		VersionID:   versionID,
		DockerImage: req.Msg.GetDockerImageTag(),
	}

	executionID, err := s.hydraEngine.StartWorkflow(ctx, "deployment", deployReq,
		hydra.WithMaxAttempts(3),
		hydra.WithTimeout(25*time.Minute),
		hydra.WithRetryBackoff(1*time.Minute),
	)
	if err != nil {
		s.logger.Error("failed to start deployment workflow",
			"version_id", versionID,
			"error", err)
		// Don't fail version creation - workflow can be retried
	} else {
		s.logger.Info("deployment workflow started",
			"version_id", versionID,
			"execution_id", executionID)
	}

	res := connect.NewResponse(&ctrlv1.CreateVersionResponse{
		VersionId: versionID,
		Status:    ctrlv1.VersionStatus_VERSION_STATUS_PENDING,
	})

	return res, nil
}
