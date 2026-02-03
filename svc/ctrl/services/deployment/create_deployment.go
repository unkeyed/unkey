package deployment

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// maxCommitMessageLength limits commit messages to prevent oversized database entries.
	maxCommitMessageLength = 10240
	// maxCommitAuthorHandleLength limits author handles (e.g., GitHub usernames).
	maxCommitAuthorHandleLength = 256
	// maxCommitAuthorAvatarLength limits avatar URL length.
	maxCommitAuthorAvatarLength = 512
)

// CreateDeployment creates a new deployment record and initiates an async Restate
// workflow. The deployment source must be a prebuilt Docker image.
//
// The method looks up the project to infer the workspace, validates the
// environment exists, fetches environment variables, and persists the deployment
// with status "pending" before triggering the workflow. Git commit metadata is
// optional but validated when provided: timestamps must be Unix epoch milliseconds
// and cannot be more than one hour in the future.
//
// The workflow runs asynchronously keyed by project ID, so only one deployment
// per project executes at a time. Returns the deployment ID and initial status.
func (s *Service) CreateDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateDeploymentRequest],
) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
	if req.Msg.GetProjectId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("project_id is required"))
	}

	dockerImage := req.Msg.GetDockerImage()
	if dockerImage == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("docker_image is required"))
	}

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

	// Fetch environment variables and build secrets blob
	envVars, err := db.Query.FindEnvironmentVariablesByEnvironmentId(ctx, s.db.RO(), env.ID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to fetch environment variables: %w", err))
	}

	secretsBlob := []byte{}
	if len(envVars) > 0 {
		secretsConfig := &ctrlv1.SecretsConfig{
			Secrets: make(map[string]string, len(envVars)),
		}
		for _, ev := range envVars {
			secretsConfig.Secrets[ev.Key] = ev.Value
		}

		var err error
		secretsBlob, err = protojson.Marshal(secretsConfig)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to marshal secrets config: %w", err))
		}
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

	s.logger.Info("deployment will use prebuilt image",
		"deployment_id", deploymentID,
		"image", dockerImage)

	// Determine command: CLI override > project default > empty array
	var commandJSON []byte
	if len(req.Msg.GetCommand()) > 0 {
		// CLI provided command override
		commandJSON, err = json.Marshal(req.Msg.GetCommand())
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to serialize command: %w", err))
		}
	} else if len(project.Command) > 0 && string(project.Command) != "[]" {
		// Use project's default command
		commandJSON = project.Command
	} else {
		// No command specified, use empty array
		commandJSON = []byte("[]")
	}

	// Insert deployment into database
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   workspaceID,
		ProjectID:                     req.Msg.GetProjectId(),
		EnvironmentID:                 env.ID,
		OpenapiSpec:                   sql.NullString{String: "", Valid: false},
		SentinelConfig:                env.SentinelConfig,
		EncryptedEnvironmentVariables: secretsBlob,
		Command:                       commandJSON,
		Status:                        db.DeploymentsStatusPending,
		CreatedAt:                     now,
		UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
		GitCommitSha:                  sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
		GitBranch:                     sql.NullString{String: gitBranch, Valid: true},
		GitCommitMessage:              sql.NullString{String: gitCommitMessage, Valid: gitCommitMessage != ""},
		GitCommitAuthorHandle:         sql.NullString{String: gitCommitAuthorHandle, Valid: gitCommitAuthorHandle != ""},
		GitCommitAuthorAvatarUrl:      sql.NullString{String: gitCommitAuthorAvatarURL, Valid: gitCommitAuthorAvatarURL != ""},
		GitCommitTimestamp:            sql.NullInt64{Int64: gitCommitTimestamp, Valid: gitCommitTimestamp != 0},
		CpuMillicores:                 256,
		MemoryMib:                     256,
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
		"docker_image", dockerImage,
	)

	// Start the deployment workflow
	keyspaceID := req.Msg.GetKeyspaceId()
	var keySpaceID *string
	if keyspaceID != "" {
		keySpaceID = &keyspaceID
	}

	deployReq := &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		KeyAuthId:    keySpaceID,
		Command:      req.Msg.GetCommand(),
		Source: &hydrav1.DeployRequest_DockerImage{
			DockerImage: &hydrav1.DockerImage{
				Image: dockerImage,
			},
		},
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

// trimLength truncates s to the specified number of characters. Note this
// operates on bytes, not runes, so multi-byte UTF-8 characters may be split.
func trimLength(s string, characters int) string {
	if len(s) > characters {
		return s[:characters]
	}
	return s
}
