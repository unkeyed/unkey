package deployment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"

	deploymentworkflow "github.com/unkeyed/unkey/go/apps/ctrl/workflows/deployment"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

func trimLength(s string, characters int) string {
	if len(s) > characters {
		return s[:characters]
	}
	return s
}

func (s *Service) CreateDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateDeploymentRequest],
) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
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

	env, err := db.Query.FindEnvironmentByProjectIdAndSlug(ctx, s.db.RO(), db.FindEnvironmentByProjectIdAndSlugParams{
		WorkspaceID: req.Msg.GetWorkspaceId(),
		ProjectID:   project.ID,
		Slug:        req.Msg.GetEnvironmentSlug(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("environment '%s' not found in workspace '%s'",
					req.Msg.GetEnvironmentSlug(), req.Msg.GetWorkspaceId()))
		}
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup environment: %w", err))
	}

	// Get git branch name for the deployment
	gitBranch := req.Msg.GetBranch()
	if gitBranch == "" {
		gitBranch = project.DefaultBranch.String
		if gitBranch == "" {
			gitBranch = "main" // fallback default
		}
	}

	// Validate git commit timestamp if provided (must be Unix epoch milliseconds)
	if req.Msg.GetGitCommitTimestamp() != 0 {
		timestamp := req.Msg.GetGitCommitTimestamp()
		// Reject timestamps that are clearly in seconds format (< 1_000_000_000_000)
		// This corresponds to January 1, 2001 in milliseconds
		if timestamp < 1_000_000_000_000 {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("git_commit_timestamp must be Unix epoch milliseconds, got %d (appears to be seconds format)", timestamp))
		}

		// Also reject future timestamps more than 1 hour ahead (likely invalid)
		maxValidTimestamp := time.Now().Add(1 * time.Hour).UnixMilli()
		if timestamp > maxValidTimestamp {
			return nil,
				connect.NewError(
					connect.CodeInvalidArgument,
					fmt.Errorf("git_commit_timestamp %d is too far in the future (must be Unix epoch milliseconds)", timestamp),
				)
		}
	}

	// Generate deployment ID
	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	// Sanitize input values before persisting
	gitCommitSha := req.Msg.GetGitCommitSha()
	gitCommitMessage := trimLength(req.Msg.GetGitCommitMessage(), 10240)
	gitCommitAuthorName := trimLength(strings.TrimSpace(req.Msg.GetGitCommitAuthorName()), 256)
	gitCommitAuthorUsername := trimLength(strings.TrimSpace(req.Msg.GetGitCommitAuthorUsername()), 256)
	gitCommitAuthorAvatarUrl := trimLength(strings.TrimSpace(req.Msg.GetGitCommitAuthorAvatarUrl()), 512)

	// Insert deployment into database
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:            deploymentID,
		WorkspaceID:   req.Msg.GetWorkspaceId(),
		ProjectID:     req.Msg.GetProjectId(),
		EnvironmentID: env.ID,
		RuntimeConfig: json.RawMessage(`{
		"regions": [{"region":"us-east-1", "vmCount": 1}],
		"cpus": 2,
		"memory": 2048
		}`),
		OpenapiSpec:         sql.NullString{String: "", Valid: false},
		Status:              db.DeploymentsStatusPending,
		CreatedAt:           now,
		UpdatedAt:           sql.NullInt64{Int64: now, Valid: true},
		GitCommitSha:        sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
		GitBranch:           sql.NullString{String: gitBranch, Valid: true},
		GitCommitMessage:    sql.NullString{String: gitCommitMessage, Valid: req.Msg.GetGitCommitMessage() != ""},
		GitCommitAuthorName: sql.NullString{String: gitCommitAuthorName, Valid: req.Msg.GetGitCommitAuthorName() != ""},
		// TODO: Use email to lookup GitHub username/avatar via GitHub API instead of persisting PII
		GitCommitAuthorUsername:  sql.NullString{String: gitCommitAuthorUsername, Valid: req.Msg.GetGitCommitAuthorUsername() != ""},
		GitCommitAuthorAvatarUrl: sql.NullString{String: gitCommitAuthorAvatarUrl, Valid: req.Msg.GetGitCommitAuthorAvatarUrl() != ""},
		GitCommitTimestamp:       sql.NullInt64{Int64: req.Msg.GetGitCommitTimestamp(), Valid: req.Msg.GetGitCommitTimestamp() != 0},
	})
	if err != nil {
		s.logger.Error("failed to insert deployment", "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.logger.Info("starting deployment workflow for deployment",
		"deployment_id", deploymentID,
		"workspace_id", req.Msg.GetWorkspaceId(),
		"project_id", req.Msg.GetProjectId(),
		"environment", env.ID,
	)

	// Start the deployment workflow directly
	deployReq := deploymentworkflow.DeployRequest{
		WorkspaceID:   req.Msg.GetWorkspaceId(),
		ProjectID:     req.Msg.GetProjectId(),
		EnvironmentID: env.ID,
		DeploymentID:  deploymentID,
		DockerImage:   req.Msg.GetDockerImage(),
		KeyspaceID:    req.Msg.GetKeyspaceId(),
	}

	err = s.deployWorkflow.Start(ctx, s.restate.Raw(), deployReq)
	if err != nil {
		s.logger.Error("failed to start deployment workflow",
			"deployment_id", deploymentID,
			"error", err)
		// Don't fail deployment creation - workflow can be retried
	} else {
		s.logger.Info("deployment workflow started",
			"deployment_id", deploymentID)
	}

	res := connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: deploymentID,
		Status:       ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	})

	return res, nil
}
