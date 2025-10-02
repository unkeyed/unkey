package deployment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
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
	// Lookup project and infer workspace from it
	project, err := db.Query.FindProjectById(ctx, s.db.RO(), req.Msg.GetProjectId())
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, 
				fmt.Errorf("project not found: %s", req.Msg.GetProjectId()))
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	workspaceID := project.WorkspaceID

	env, err := db.Query.FindEnvironmentByProjectIdAndSlug(ctx, s.db.RO(), db.FindEnvironmentByProjectIdAndSlugParams{
		WorkspaceID: workspaceID,
		ProjectID:   project.ID,
		Slug:        req.Msg.GetEnvironmentSlug(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("environment '%s' not found in workspace '%s'",
					req.Msg.GetEnvironmentSlug(), workspaceID))
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
	gitCommitAuthorHandle := trimLength(strings.TrimSpace(req.Msg.GetGitCommitAuthorHandle()), 256)
	gitCommitAuthorAvatarUrl := trimLength(strings.TrimSpace(req.Msg.GetGitCommitAuthorAvatarUrl()), 512)

	// Insert deployment into database
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:            deploymentID,
		WorkspaceID:   workspaceID,
		ProjectID:     req.Msg.GetProjectId(),
		EnvironmentID: env.ID,
		RuntimeConfig: json.RawMessage(`{
		"regions": [{"region":"us-east-1", "vmCount": 1}],
		"cpus": 2,
		"memory": 2048
		}`),
		OpenapiSpec:              sql.NullString{String: "", Valid: false},
		Status:                   db.DeploymentsStatusPending,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Int64: now, Valid: true},
		GitCommitSha:             sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
		GitBranch:                sql.NullString{String: gitBranch, Valid: true},
		GitCommitMessage:         sql.NullString{String: gitCommitMessage, Valid: req.Msg.GetGitCommitMessage() != ""},
		GitCommitAuthorHandle:    sql.NullString{String: gitCommitAuthorHandle, Valid: req.Msg.GetGitCommitAuthorHandle() != ""},
		GitCommitAuthorAvatarUrl: sql.NullString{String: gitCommitAuthorAvatarUrl, Valid: req.Msg.GetGitCommitAuthorAvatarUrl() != ""},
		GitCommitTimestamp:       sql.NullInt64{Int64: req.Msg.GetGitCommitTimestamp(), Valid: req.Msg.GetGitCommitTimestamp() != 0},
	})
	if err != nil {
		s.logger.Error("failed to insert deployment", "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.logger.Info("starting deployment workflow for deployment",
		"deployment_id", deploymentID,
		"workspace_id", workspaceID,
		"project_id", req.Msg.GetProjectId(),
		"environment", env.ID,
	)

	// Start the deployment workflow directly
	deployReq := &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		DockerImage:  req.Msg.GetDockerImage(),
		KeyAuthId:    req.Msg.GetKeyspaceId(),
	}
	// this is ugly, but we're waiting for
	// https://github.com/restatedev/sdk-go/issues/103
	invocation := restateingress.WorkflowSend[*hydrav1.DeployRequest](
		s.restate,
		"hydra.v1.DeploymentService",
		project.ID,
		"Deploy",
	).Send(ctx, deployReq)
	if invocation.Error != nil {
		s.logger.Error("failed to start deployment workflow", "error", invocation.Error.Error())
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unable to start workflow: %w", invocation.Error))
	}
	s.logger.Info("deployment workflow started",
		"deployment_id", deploymentID,
		"invocation_id", invocation.Id,
	)

	res := connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: deploymentID,
		Status:       ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	})

	return res, nil
}
