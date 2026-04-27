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
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
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
// current deployment, used when redeploying a non-git project.
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
// apps deploy HEAD of their default branch, non-git apps reuse the live
// deployment's Docker image.
//
// The workflow runs asynchronously keyed by {app, environment}, so different
// environments (e.g. prod vs preview) for the same app deploy in parallel while
// lifecycle operations within one environment remain serialized. Workspace-wide
// build concurrency is enforced separately via BuildSlotService. Returns the
// deployment ID and initial status.
func (s *Service) CreateDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateDeploymentRequest],
) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	if req.Msg.GetProjectId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("project_id is required"))
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

	appID := req.Msg.GetAppId()
	if appID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("app_id is required"))
	}

	// Verify the environment exists
	env, err := db.Query.FindEnvironmentByAppIdAndSlug(ctx, s.db.RO(), db.FindEnvironmentByAppIdAndSlugParams{
		AppID: appID,
		Slug:  req.Msg.GetEnvironmentSlug(),
	})
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("environment '%s' not found for app '%s'",
					req.Msg.GetEnvironmentSlug(), appID))
		}
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup environment: %w", err))
	}

	// Lookup app with build and runtime settings
	appWithSettings, err := db.Query.FindAppWithSettings(ctx, s.db.RO(), db.FindAppWithSettingsParams{
		ID:            appID,
		EnvironmentID: env.Environment.ID,
	})
	if err != nil && db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("app '%s' not found or missing settings", appID))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup app: %w", err))
	}

	app := appWithSettings.App
	appBuildSettings := appWithSettings.AppBuildSetting
	appRuntimeSettings := appWithSettings.AppRuntimeSetting

	// Fetch app-scoped environment variables
	appEnvVars, err := db.Query.FindAppEnvVarsByAppAndEnv(ctx, s.db.RO(), db.FindAppEnvVarsByAppAndEnvParams{
		AppID:         app.ID,
		EnvironmentID: env.Environment.ID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to fetch app environment variables: %w", err))
	}

	secretsBlob := []byte{}
	if len(appEnvVars) > 0 {
		secretsConfig := &ctrlv1.SecretsConfig{
			Secrets: make(map[string]string, len(appEnvVars)),
		}
		for _, ev := range appEnvVars {
			if !validation.IsValidEnvVarKey(ev.Key) {
				return nil, connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("environment variable key %q is invalid: %s", ev.Key, validation.ErrMsgInvalidEnvVarKey))
			}
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

	var gitCommitSha, gitBranch, gitCommitMessage, gitCommitAuthorHandle, gitCommitAuthorAvatarURL, gitForkRepository string
	var gitCommitTimestamp int64
	var deployReq *hydrav1.DeployRequest

	// Resolve request-level overrides once so all branches can use them.
	keyspaceID := req.Msg.GetKeyspaceId()
	var keyAuthID *string
	if keyspaceID != "" {
		keyAuthID = &keyspaceID
	}
	command := req.Msg.GetCommand()

	if dockerImage := req.Msg.GetDockerImage(); dockerImage != "" {
		// Explicit docker image (CLI, REST API)
		gitBranch = branchFromGitCommit(req.Msg.GetGitCommit(), appWithSettings.App.DefaultBranch)

		if gitCommit := req.Msg.GetGitCommit(); gitCommit != nil {
			gitCommitSha = gitCommit.GetCommitSha()
			gitCommitMessage = trimLength(gitCommit.GetCommitMessage(), maxCommitMessageLength)
			gitCommitAuthorHandle = trimLength(strings.TrimSpace(gitCommit.GetAuthorHandle()), maxCommitAuthorHandleLength)
			gitCommitAuthorAvatarURL = trimLength(strings.TrimSpace(gitCommit.GetAuthorAvatarUrl()), maxCommitAuthorAvatarLength)
			gitCommitTimestamp = gitCommit.GetTimestamp()
		}

		logger.Info("deployment will use prebuilt image",
			"deployment_id", deploymentID,
			"app_id", app.ID,
			"image", dockerImage)

		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			KeyAuthId:    keyAuthID,
			Command:      command,
			Source: &hydrav1.DeployRequest_DockerImage{
				DockerImage: &hydrav1.DockerImage{
					Image: dockerImage,
				},
			},
		}
	} else {
		// Source omitted (dashboard redeploy w/o commit SHA): auto-detect from app config
		repoConn, repoErr := db.Query.FindGithubRepoConnectionByAppId(ctx, s.db.RO(), app.ID)
		hasRepoConnection := repoErr == nil
		if repoErr != nil && !db.IsNotFound(repoErr) {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to lookup github repo connection: %w", repoErr))
		}

		if hasRepoConnection {
			// Has github_repo_connections row: build from git source.
			gitBranch = branchFromGitCommit(req.Msg.GetGitCommit(), appWithSettings.App.DefaultBranch)

			if gitCommit := req.Msg.GetGitCommit(); gitCommit != nil {
				gitCommitSha = gitCommit.GetCommitSha()
				gitCommitMessage = trimLength(gitCommit.GetCommitMessage(), maxCommitMessageLength)
				gitCommitAuthorHandle = trimLength(strings.TrimSpace(gitCommit.GetAuthorHandle()), maxCommitAuthorHandleLength)
				gitCommitAuthorAvatarURL = trimLength(strings.TrimSpace(gitCommit.GetAuthorAvatarUrl()), maxCommitAuthorAvatarLength)
				gitCommitTimestamp = gitCommit.GetTimestamp()
				gitForkRepository = gitCommit.GetForkRepository()
			}

			deployReq = &hydrav1.DeployRequest{
				DeploymentId: deploymentID,
				KeyAuthId:    keyAuthID,
				Command:      command,
				Source: &hydrav1.DeployRequest_Git{
					Git: &hydrav1.GitSource{
						InstallationId: repoConn.InstallationID,
						Repository:     repoConn.RepositoryFullName,
						CommitSha:      gitCommitSha,
						ContextPath:    appBuildSettings.DockerContext,
						DockerfilePath: appBuildSettings.Dockerfile,
						Branch:         gitBranch,
						ForkRepository: gitForkRepository,
					},
				},
			}
		} else {
			// No repo connection: redeploy the current deployment's Docker image
			dockerInfo, dockerErr := buildDockerSource(ctx, s.db, app, deploymentID)
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
				KeyAuthId:    keyAuthID,
				Command:      command,
				Source: &hydrav1.DeployRequest_DockerImage{
					DockerImage: &hydrav1.DockerImage{
						Image: dockerInfo.dockerImage,
					},
				},
			}
		}
	}

	// Insert deployment into database, snapshotting settings from app
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   workspaceID,
		ProjectID:                     project.ID,
		AppID:                         app.ID,
		EnvironmentID:                 env.Environment.ID,
		SentinelConfig:                appRuntimeSettings.SentinelConfig,
		EncryptedEnvironmentVariables: secretsBlob,
		Command:                       appRuntimeSettings.Command,
		Status:                        db.DeploymentsStatusPending,
		CreatedAt:                     now,
		UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
		GitCommitSha:                  sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
		GitBranch:                     sql.NullString{String: gitBranch, Valid: gitBranch != ""},
		GitCommitMessage:              sql.NullString{String: gitCommitMessage, Valid: gitCommitMessage != ""},
		GitCommitAuthorHandle:         sql.NullString{String: gitCommitAuthorHandle, Valid: gitCommitAuthorHandle != ""},
		GitCommitAuthorAvatarUrl:      sql.NullString{String: gitCommitAuthorAvatarURL, Valid: gitCommitAuthorAvatarURL != ""},
		GitCommitTimestamp:            sql.NullInt64{Int64: gitCommitTimestamp, Valid: gitCommitTimestamp != 0},
		CpuMillicores:                 appRuntimeSettings.CpuMillicores,
		MemoryMib:                     appRuntimeSettings.MemoryMib,
		StorageMib:                    appRuntimeSettings.StorageMib,
		Port:                          appRuntimeSettings.Port,
		ShutdownSignal:                db.DeploymentsShutdownSignal(appRuntimeSettings.ShutdownSignal),
		UpstreamProtocol:              db.DeploymentsUpstreamProtocol(appRuntimeSettings.UpstreamProtocol),
		Healthcheck:                   appRuntimeSettings.Healthcheck,
		PrNumber:                      sql.NullInt64{Int64: 0, Valid: false},
		ForkRepositoryFullName:        sql.NullString{String: gitForkRepository, Valid: gitForkRepository != ""},
	})
	if err != nil {
		logger.Error("failed to insert deployment", "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	logger.Info("starting deployment workflow",
		"deployment_id", deploymentID,
		"workspace_id", workspaceID,
		"project_id", project.ID,
		"app_id", app.ID,
		"environment", env.Environment.ID,
	)

	// Send deployment request asynchronously, keyed by deployment_id —
	// each deployment runs as its own isolated workflow.
	invocation, err := s.deploymentClient(deploymentID).
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

	// Persist the invocation ID so the deployment can be cancelled later.
	invocationID := invocation.Id()
	if updateErr := db.Query.UpdateDeploymentInvocationID(ctx, s.db.RW(), db.UpdateDeploymentInvocationIDParams{
		ID:           deploymentID,
		InvocationID: sql.NullString{Valid: true, String: invocationID},
		UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}); updateErr != nil {
		logger.Error("failed to persist invocation id",
			"deployment_id", deploymentID,
			"invocation_id", invocationID,
			"error", updateErr,
		)
		// Non-fatal: the deployment is already running. Cancellation may
		// fail for this one deployment, but the workflow itself is fine.
	}

	logger.Info("deployment workflow started",
		"deployment_id", deploymentID,
		"invocation_id", invocationID,
	)

	if cancelErr := s.dedup.CancelOlderSiblings(ctx, dedup.Newer{
		ID:            deploymentID,
		AppID:         app.ID,
		EnvironmentID: env.Environment.ID,
		GitBranch:     gitBranch,
		CreatedAt:     now,
	}); cancelErr != nil {
		logger.Error("failed to cancel superseded siblings",
			"deployment_id", deploymentID,
			"error", cancelErr,
		)
	}

	return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: deploymentID,
		Status:       ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}

// branchFromGitCommit extracts the branch from GitCommitInfo, falling back
// to the app's default branch or "main".
func branchFromGitCommit(gitCommit *ctrlv1.GitCommitInfo, defaultBranch string) string {
	if gitCommit != nil && gitCommit.GetBranch() != "" {
		return gitCommit.GetBranch()
	}
	if defaultBranch != "" {
		return defaultBranch
	}
	return "main"
}

// buildDockerSource looks up the app's current deployment's Docker image and carries
// over its git metadata for the new deployment record.
func buildDockerSource(
	ctx context.Context,
	database db.Database,
	app db.App,
	deploymentID string,
) (dockerSourceInfo, error) {
	if !app.CurrentDeploymentID.Valid || app.CurrentDeploymentID.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has no current deployment and no git connection; cannot redeploy", app.ID))
	}

	currentDeployment, err := db.Query.FindDeploymentById(ctx, database.RO(), app.CurrentDeploymentID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return dockerSourceInfo{}, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("current deployment %q not found", app.CurrentDeploymentID.String))
		}
		return dockerSourceInfo{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup current deployment: %w", err))
	}

	if !currentDeployment.Image.Valid || currentDeployment.Image.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("current deployment %q has no Docker image; cannot redeploy without git connection",
				app.CurrentDeploymentID.String))
	}

	logger.Info("deployment will reuse current deployment image",
		"deployment_id", deploymentID,
		"current_deployment_id", app.CurrentDeploymentID.String,
		"image", currentDeployment.Image.String)

	return dockerSourceInfo{
		dockerImage:     currentDeployment.Image.String,
		commitSHA:       currentDeployment.GitCommitSha.String,
		branch:          currentDeployment.GitBranch.String,
		commitMessage:   currentDeployment.GitCommitMessage.String,
		authorHandle:    currentDeployment.GitCommitAuthorHandle.String,
		authorAvatarURL: currentDeployment.GitCommitAuthorAvatarUrl.String,
		commitTimestamp: currentDeployment.GitCommitTimestamp.Int64,
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
