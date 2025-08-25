package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// limitString truncates a string to the specified maximum number of runes
func limitString(s string, maxRunes int) string {
	runes := []rune(s)
	if len(runes) > maxRunes {
		return string(runes[:maxRunes])
	}
	return s
}

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
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("git_commit_timestamp %d is too far in the future (must be Unix epoch milliseconds)", timestamp))
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
		ID:                       deploymentID,
		WorkspaceID:              req.Msg.GetWorkspaceId(),
		ProjectID:                req.Msg.GetProjectId(),
		Environment:              environment,
		BuildID:                  sql.NullString{String: "", Valid: false}, // Build creation handled separately
		RootfsImageID:            "",                                       // Image handling not implemented yet
		GitCommitSha:             sql.NullString{String: req.Msg.GetGitCommitSha(), Valid: req.Msg.GetGitCommitSha() != ""},
		GitBranch:                sql.NullString{String: gitBranch, Valid: true},
		GitCommitMessage:         sql.NullString{String: limitString(req.Msg.GetGitCommitMessage(), 10240), Valid: req.Msg.GetGitCommitMessage() != ""},
		GitCommitAuthorName:      sql.NullString{String: limitString(strings.TrimSpace(req.Msg.GetGitCommitAuthorName()), 256), Valid: req.Msg.GetGitCommitAuthorName() != ""},
		GitCommitAuthorEmail:     sql.NullString{String: limitString(strings.TrimSpace(strings.ToLower(req.Msg.GetGitCommitAuthorEmail())), 256), Valid: req.Msg.GetGitCommitAuthorEmail() != ""},
		GitCommitAuthorUsername:  sql.NullString{String: limitString(strings.TrimSpace(req.Msg.GetGitCommitAuthorUsername()), 256), Valid: req.Msg.GetGitCommitAuthorUsername() != ""},
		GitCommitAuthorAvatarUrl: sql.NullString{String: limitString(strings.TrimSpace(req.Msg.GetGitCommitAuthorAvatarUrl()), 512), Valid: req.Msg.GetGitCommitAuthorAvatarUrl() != ""},
		GitCommitTimestamp:       sql.NullInt64{Int64: req.Msg.GetGitCommitTimestamp(), Valid: req.Msg.GetGitCommitTimestamp() != 0},
		ConfigSnapshot:           []byte("{}"), // Configuration snapshot placeholder
		OpenapiSpec:              sql.NullString{String: "", Valid: false},
		Status:                   db.DeploymentsStatusPending,
		CreatedAt:                now,
		UpdatedAt:                sql.NullInt64{Int64: now, Valid: true},
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
