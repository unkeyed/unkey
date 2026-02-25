package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
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

// dockerSourceInfo holds the Docker image and inherited git metadata from a
// live deployment, used when redeploying a non-git project.
type dockerSourceInfo struct {
	commitSHA       string
	branch          string
	commitMessage   string
	authorHandle    string
	authorAvatarURL string
	commitTimestamp int64
	dockerImage     string
}

// CreateDeployment creates a new deployment record and initiates an async Restate
// workflow. When source is omitted, the handler auto-detects: git-connected
// projects deploy HEAD of their default branch, non-git projects reuse the live
// deployment's Docker image.
//
// The workflow runs asynchronously keyed by project ID, so only one deployment
// per project executes at a time.
func (s *Service) CreateDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateDeploymentRequest],
) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
	if req.Msg.GetProjectId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("project_id is required"))
	}

	// Lookup project, environment, build/runtime settings, and env vars
	row, err := db.Query.FindProjectWithEnvironmentSettingsAndVars(ctx, s.db.RO(),
		db.FindProjectWithEnvironmentSettingsAndVarsParams{
			ProjectID: req.Msg.GetProjectId(),
			Slug:      req.Msg.GetEnvironmentSlug(),
		})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("project %q or environment %q not found",
					req.Msg.GetProjectId(), req.Msg.GetEnvironmentSlug()))
		}
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup project and environment: %w", err))
	}
	project := row.Project
	workspaceID := project.WorkspaceID
	envSettings := row
	env := row.Environment

	envVars, err := db.UnmarshalNullableJSONTo[[]db.EnvVarInfo](row.EnvironmentVariables)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to unmarshal environment variables: %w", err))
	}

	secretsBlob := []byte{}
	if len(envVars) > 0 {
		secretsConfig := &ctrlv1.SecretsConfig{
			Secrets: make(map[string]string, len(envVars)),
		}
		for _, ev := range envVars {
			secretsConfig.Secrets[ev.Key] = ev.Value
		}

		secretsBlob, err = protojson.Marshal(secretsConfig)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to marshal secrets config: %w", err))
		}
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	var gitCommitSha, gitBranch, gitCommitMessage, gitCommitAuthorHandle, gitCommitAuthorAvatarURL string
	var gitCommitTimestamp int64
	var deployReq *hydrav1.DeployRequest

	switch source := req.Msg.GetSource().(type) {
	case *ctrlv1.CreateDeploymentRequest_DockerImage:
		// Docker image source: deploy a prebuilt image directly
		dockerImage := source.DockerImage
		if dockerImage == "" {
			return nil, connect.NewError(connect.CodeInvalidArgument,
				fmt.Errorf("docker_image is required when source is docker_image"))
		}

		// Resolve branch from git_commit metadata or project default
		gitBranch = branchFromGitCommit(req.Msg.GetGitCommit(), project)

		// Validate git commit timestamp if provided
		if tsErr := validateGitCommitTimestamp(req.Msg.GetGitCommit()); tsErr != nil {
			return nil, tsErr
		}

		// Extract git metadata
		if gitCommit := req.Msg.GetGitCommit(); gitCommit != nil {
			gitCommitSha = gitCommit.GetCommitSha()
			gitCommitMessage = trimLength(gitCommit.GetCommitMessage(), maxCommitMessageLength)
			gitCommitAuthorHandle = trimLength(strings.TrimSpace(gitCommit.GetAuthorHandle()), maxCommitAuthorHandleLength)
			gitCommitAuthorAvatarURL = trimLength(strings.TrimSpace(gitCommit.GetAuthorAvatarUrl()), maxCommitAuthorAvatarLength)
			gitCommitTimestamp = gitCommit.GetTimestamp()
		}

		logger.Info("deployment will use prebuilt image",
			"deployment_id", deploymentID,
			"image", dockerImage)

		keyspaceID := req.Msg.GetKeyspaceId()
		var keySpaceID *string
		if keyspaceID != "" {
			keySpaceID = &keyspaceID
		}

		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			KeyAuthId:    keySpaceID,
			Command:      req.Msg.GetCommand(),
			Source: &hydrav1.DeployRequest_DockerImage{
				DockerImage: &hydrav1.DockerImage{
					Image: dockerImage,
				},
			},
		}

	case *ctrlv1.CreateDeploymentRequest_Git:
		// Git source: build from a GitHub repo connection
		repoConn, repoErr := db.Query.FindGithubRepoConnectionByProjectId(ctx, s.db.RO(), req.Msg.GetProjectId())
		if repoErr != nil {
			if db.IsNotFound(repoErr) {
				return nil, connect.NewError(connect.CodeFailedPrecondition,
					fmt.Errorf("project %q has no git connection; cannot deploy from git source", req.Msg.GetProjectId()))
			}
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to lookup github repo connection: %w", repoErr))
		}

		branch, commitSHA := resolveGitTarget(source.Git, project)
		gitBranch = branch
		gitCommitSha = commitSHA

		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			KeyAuthId:    nil,
			Command:      nil,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repoConn.InstallationID,
					Repository:     repoConn.RepositoryFullName,
					CommitSha:      commitSHA,
					ContextPath:    envSettings.EnvironmentBuildSetting.DockerContext,
					DockerfilePath: envSettings.EnvironmentBuildSetting.Dockerfile,
					Branch:         branch,
				},
			},
		}

	default:
		// No source specified: auto-detect based on project configuration
		repoConn, repoErr := db.Query.FindGithubRepoConnectionByProjectId(ctx, s.db.RO(), req.Msg.GetProjectId())
		hasRepoConnection := repoErr == nil
		if repoErr != nil && !db.IsNotFound(repoErr) {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to lookup github repo connection: %w", repoErr))
		}

		if hasRepoConnection {
			// Git-connected: deploy HEAD of default branch
			defaultBranch := project.DefaultBranch.String
			if defaultBranch == "" {
				defaultBranch = "main"
			}
			gitBranch = defaultBranch

			deployReq = &hydrav1.DeployRequest{
				DeploymentId: deploymentID,
				KeyAuthId:    nil,
				Command:      nil,
				Source: &hydrav1.DeployRequest_Git{
					Git: &hydrav1.GitSource{
						InstallationId: repoConn.InstallationID,
						Repository:     repoConn.RepositoryFullName,
						CommitSha:      "",
						ContextPath:    envSettings.EnvironmentBuildSetting.DockerContext,
						DockerfilePath: envSettings.EnvironmentBuildSetting.Dockerfile,
						Branch:         defaultBranch,
					},
				},
			}
		} else {
			// Non-git: reuse the live deployment's Docker image
			dockerInfo, dockerErr := buildDockerSource(ctx, s.db, project, deploymentID)
			if dockerErr != nil {
				return nil, dockerErr
			}
			gitCommitSha = dockerInfo.commitSHA
			gitBranch = dockerInfo.branch
			gitCommitMessage = dockerInfo.commitMessage
			gitCommitAuthorHandle = dockerInfo.authorHandle
			gitCommitAuthorAvatarURL = dockerInfo.authorAvatarURL
			gitCommitTimestamp = dockerInfo.commitTimestamp

			deployReq = &hydrav1.DeployRequest{
				DeploymentId: deploymentID,
				KeyAuthId:    nil,
				Command:      nil,
				Source: &hydrav1.DeployRequest_DockerImage{
					DockerImage: &hydrav1.DockerImage{
						Image: dockerInfo.dockerImage,
					},
				},
			}
		}
	}

	// Insert deployment into database, snapshotting settings from environment
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   workspaceID,
		ProjectID:                     project.ID,
		EnvironmentID:                 env.ID,
		OpenapiSpec:                   sql.NullString{String: "", Valid: false},
		SentinelConfig:                envSettings.EnvironmentRuntimeSetting.SentinelConfig,
		EncryptedEnvironmentVariables: secretsBlob,
		Command:                       envSettings.EnvironmentRuntimeSetting.Command,
		Status:                        db.DeploymentsStatusPending,
		CreatedAt:                     now,
		UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
		GitCommitSha:                  sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
		GitBranch:                     sql.NullString{String: gitBranch, Valid: gitBranch != ""},
		GitCommitMessage:              sql.NullString{String: gitCommitMessage, Valid: gitCommitMessage != ""},
		GitCommitAuthorHandle:         sql.NullString{String: gitCommitAuthorHandle, Valid: gitCommitAuthorHandle != ""},
		GitCommitAuthorAvatarUrl:      sql.NullString{String: gitCommitAuthorAvatarURL, Valid: gitCommitAuthorAvatarURL != ""},
		GitCommitTimestamp:            sql.NullInt64{Int64: gitCommitTimestamp, Valid: gitCommitTimestamp != 0},
		CpuMillicores:                 envSettings.EnvironmentRuntimeSetting.CpuMillicores,
		MemoryMib:                     envSettings.EnvironmentRuntimeSetting.MemoryMib,
		Port:                          envSettings.EnvironmentRuntimeSetting.Port,
		ShutdownSignal:                db.DeploymentsShutdownSignal(envSettings.EnvironmentRuntimeSetting.ShutdownSignal),
		Healthcheck:                   envSettings.EnvironmentRuntimeSetting.Healthcheck,
	})
	if err != nil {
		logger.Error("failed to insert deployment", "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	logger.Info("starting deployment workflow",
		"deployment_id", deploymentID,
		"workspace_id", workspaceID,
		"project_id", project.ID,
		"environment", env.ID,
	)

	invocation, err := s.deploymentClient(project.ID).
		Deploy().
		Send(ctx, deployReq)
	if err != nil {
		logger.Error("failed to start deployment workflow", "error", err)

		updateErr := db.Query.UpdateDeploymentStatus(ctx, s.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    db.DeploymentsStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			logger.Error("failed to mark deployment as failed", "deployment_id", deploymentID, "error", updateErr)
		}

		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unable to start workflow: %w", err))
	}

	logger.Info("deployment workflow started",
		"deployment_id", deploymentID,
		"invocation_id", invocation.Id,
	)

	return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: deploymentID,
		Status:       ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}

// branchFromGitCommit extracts the branch from GitCommitInfo, falling back
// to the project's default branch or "main".
func branchFromGitCommit(gitCommit *ctrlv1.GitCommitInfo, project db.Project) string {
	if gitCommit != nil && gitCommit.GetBranch() != "" {
		return gitCommit.GetBranch()
	}
	if project.DefaultBranch.Valid && project.DefaultBranch.String != "" {
		return project.DefaultBranch.String
	}
	return "main"
}

// validateGitCommitTimestamp validates the timestamp in GitCommitInfo if present.
func validateGitCommitTimestamp(gitCommit *ctrlv1.GitCommitInfo) error {
	if gitCommit == nil || gitCommit.GetTimestamp() == 0 {
		return nil
	}
	timestamp := gitCommit.GetTimestamp()

	// Reject timestamps that are clearly in seconds format (< 1_000_000_000_000)
	if timestamp < 1_000_000_000_000 {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("git_commit_timestamp must be Unix epoch milliseconds, got %d (appears to be seconds format)", timestamp))
	}

	// Reject future timestamps more than 1 hour ahead
	maxValidTimestamp := time.Now().Add(1 * time.Hour).UnixMilli()
	if timestamp > maxValidTimestamp {
		return connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("git_commit_timestamp %d is too far in the future (must be Unix epoch milliseconds)", timestamp))
	}
	return nil
}

// resolveGitTarget extracts branch and commit SHA from a GitTarget message.
// When a commit_sha is provided, branch defaults to the project's default
// branch for metadata. When only a branch is provided, commit SHA is left empty
// for the worker to resolve. When neither is provided, the project's default
// branch is used.
func resolveGitTarget(git *ctrlv1.GitTarget, project db.Project) (branch string, commitSHA string) {
	defaultBranch := project.DefaultBranch.String
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	if git == nil {
		return defaultBranch, ""
	}

	branch = git.GetBranch()
	if branch == "" {
		branch = defaultBranch
	}

	return branch, git.GetCommitSha()
}

// buildDockerSource looks up the live deployment's Docker image and carries
// over its git metadata for the new deployment record.
func buildDockerSource(
	ctx context.Context,
	database db.Database,
	project db.Project,
	deploymentID string,
) (dockerSourceInfo, error) {
	if !project.LiveDeploymentID.Valid || project.LiveDeploymentID.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("project %q has no live deployment and no git connection; cannot redeploy", project.ID))
	}

	liveDeployment, err := db.Query.FindDeploymentById(ctx, database.RO(), project.LiveDeploymentID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return dockerSourceInfo{}, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("live deployment %q not found", project.LiveDeploymentID.String))
		}
		return dockerSourceInfo{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup live deployment: %w", err))
	}

	if !liveDeployment.Image.Valid || liveDeployment.Image.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("live deployment %q has no Docker image; cannot redeploy without git connection",
				project.LiveDeploymentID.String))
	}

	logger.Info("deployment will reuse live deployment image",
		"deployment_id", deploymentID,
		"live_deployment_id", project.LiveDeploymentID.String,
		"image", liveDeployment.Image.String)

	return dockerSourceInfo{
		dockerImage:     liveDeployment.Image.String,
		commitSHA:       liveDeployment.GitCommitSha.String,
		branch:          liveDeployment.GitBranch.String,
		commitMessage:   liveDeployment.GitCommitMessage.String,
		authorHandle:    liveDeployment.GitCommitAuthorHandle.String,
		authorAvatarURL: liveDeployment.GitCommitAuthorAvatarUrl.String,
		commitTimestamp: liveDeployment.GitCommitTimestamp.Int64,
	}, nil
}

// trimLength truncates s to the specified number of characters. Note this
// operates on bytes, not runes, so multi-byte UTF-8 characters may be split.
func trimLength(s string, characters int) string {
	if len(s) > characters {
		return s[:characters]
	}
	return s
}
