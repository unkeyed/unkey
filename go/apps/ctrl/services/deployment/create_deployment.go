package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/go/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/proto"
)

const (
	maxCommitMessageLength      = 10240
	maxCommitAuthorHandleLength = 256
	maxCommitAuthorAvatarLength = 512
)

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
	if req.Msg.GetGitCommit() != nil && req.Msg.GetGitCommit().GetTimestamp() != 0 {
		timestamp := req.Msg.GetGitCommit().GetTimestamp()
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

	var gitCommitSha, gitCommitMessage, gitCommitAuthorHandle, gitCommitAuthorAvatarURL string
	var gitCommitTimestamp int64

	if gitCommit := req.Msg.GetGitCommit(); gitCommit != nil {
		gitCommitSha = gitCommit.GetCommitSha()
		gitCommitMessage = trimLength(gitCommit.GetCommitMessage(), maxCommitMessageLength)
		gitCommitAuthorHandle = trimLength(strings.TrimSpace(gitCommit.GetAuthorHandle()), maxCommitAuthorHandleLength)
		gitCommitAuthorAvatarURL = trimLength(strings.TrimSpace(gitCommit.GetAuthorAvatarUrl()), maxCommitAuthorAvatarLength)
		gitCommitTimestamp = gitCommit.GetTimestamp()
	}

	var buildContextKey string
	var dockerfilePath string
	var dockerImage *string

	switch source := req.Msg.GetSource().(type) {
	case *ctrlv1.CreateDeploymentRequest_BuildContext:
		buildContextKey = source.BuildContext.GetBuildContextPath()
		dockerfilePath = source.BuildContext.GetDockerfilePath()
		if dockerfilePath == "" {
			dockerfilePath = "./Dockerfile"
		}

	case *ctrlv1.CreateDeploymentRequest_DockerImage:
		image := source.DockerImage
		dockerImage = &image

	default:
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("source must be specified (either build_context or docker_image)"))
	}

	// Log deployment source
	if buildContextKey != "" {
		s.logger.Info("deployment will build from source",
			"deployment_id", deploymentID,
			"context_key", buildContextKey,
			"dockerfile", dockerfilePath)
	} else {
		s.logger.Info("deployment will use prebuilt image",
			"deployment_id", deploymentID,
			"image", *dockerImage)
	}

	// Insert deployment into database
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:                       deploymentID,
		WorkspaceID:              workspaceID,
		ProjectID:                req.Msg.GetProjectId(),
		EnvironmentID:            env.ID,
		OpenapiSpec:              sql.NullString{String: "", Valid: false},
		SentinelConfig:           env.SentinelConfig,
		Status:                   db.DeploymentsStatusPending,
		CreatedAt:                now,
		GitCommitSha:             sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
		GitBranch:                sql.NullString{String: gitBranch, Valid: true},
		GitCommitMessage:         sql.NullString{String: gitCommitMessage, Valid: gitCommitMessage != ""},
		GitCommitAuthorHandle:    sql.NullString{String: gitCommitAuthorHandle, Valid: gitCommitAuthorHandle != ""},
		GitCommitAuthorAvatarUrl: sql.NullString{String: gitCommitAuthorAvatarURL, Valid: gitCommitAuthorAvatarURL != ""},
		GitCommitTimestamp:       sql.NullInt64{Int64: gitCommitTimestamp, Valid: gitCommitTimestamp != 0},
		CpuMillicores:            256,
		MemoryMib:                256,
	})
	if err != nil {
		s.logger.Error("failed to insert deployment", "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	s.logger.Info("starting deployment workflow",
		"deployment_id", deploymentID,
		"workspace_id", workspaceID,
		"project_id", req.Msg.GetProjectId(),
		"environment", env.ID,
		"context_key", buildContextKey,
		"docker_image", dockerImage,
	)

	// Start the deployment workflow
	keyspaceID := req.Msg.GetKeyspaceId()
	var keySpaceID *string
	if keyspaceID != "" {
		keySpaceID = &keyspaceID
	}

	deployReq := &hydrav1.DeployRequest{
		BuildContextPath: nil,
		DockerfilePath:   nil,
		DockerImage:      nil,
		DeploymentId:     deploymentID,
		KeyAuthId:        keySpaceID,
	}

	switch source := req.Msg.GetSource().(type) {
	case *ctrlv1.CreateDeploymentRequest_BuildContext:
		deployReq.BuildContextPath = proto.String(source.BuildContext.GetBuildContextPath())
		deployReq.DockerfilePath = source.BuildContext.DockerfilePath
	case *ctrlv1.CreateDeploymentRequest_DockerImage:
		deployReq.DockerImage = proto.String(source.DockerImage)
	}

	// Send deployment request asynchronously (fire-and-forget)
	invocation, err := s.deploymentClient(project.ID).
		Deploy().
		Send(ctx, deployReq)
	if err != nil {
		s.logger.Error("failed to start deployment workflow", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unable to start workflow: %w", err))
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

func trimLength(s string, characters int) string {
	if len(s) > characters {
		return s[:characters]
	}
	return s
}
