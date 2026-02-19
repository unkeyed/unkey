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
	"github.com/unkeyed/unkey/svc/ctrl/internal/envresolve"
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
// The method looks up the project and app to infer the workspace, validates the
// environment exists, fetches app-scoped environment variables with template
// resolution, and persists the deployment with status "pending" before triggering
// the workflow. Git commit metadata is optional but validated when provided:
// timestamps must be Unix epoch milliseconds and cannot be more than one hour
// in the future.
//
// The workflow runs asynchronously keyed by app ID, so only one deployment
// per app executes at a time. Returns the deployment ID and initial status.
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

	// Default app_slug to "default" for backwards compatibility
	appSlug := req.Msg.GetAppSlug()
	if appSlug == "" {
		appSlug = "default"
	}

	// Lookup app with build and runtime settings for this environment
	appWithSettings, err := db.Query.FindAppWithSettings(ctx, s.db.RO(), db.FindAppWithSettingsParams{
		ProjectID:     project.ID,
		Slug:          appSlug,
		EnvironmentID: req.Msg.GetEnvironmentSlug(), // will be resolved below
	})
	// If app not found, fall back to looking up app and environment separately
	// This handles the case where settings haven't been created yet
	if err != nil && db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("app '%s' not found in project '%s' or missing settings for environment '%s'",
				appSlug, req.Msg.GetProjectId(), req.Msg.GetEnvironmentSlug()))
	}
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup app: %w", err))
	}

	app := appWithSettings.App
	appBuildSettings := appWithSettings.AppBuildSetting
	appRuntimeSettings := appWithSettings.AppRuntimeSetting
	_ = appBuildSettings // build settings used by the workflow, not here

	// Verify the environment exists
	envSettings, err := db.Query.FindEnvironmentWithSettingsByProjectIdAndSlug(ctx, s.db.RO(), db.FindEnvironmentWithSettingsByProjectIdAndSlugParams{
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
	env := envSettings.Environment

	// Fetch app-scoped environment variables
	appEnvVars, err := db.Query.FindAppEnvVarsByAppAndEnv(ctx, s.db.RO(), db.FindAppEnvVarsByAppAndEnvParams{
		AppID:         app.ID,
		EnvironmentID: env.ID,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to fetch app environment variables: %w", err))
	}

	// Convert to envresolve types
	appVars := make([]envresolve.AppVar, len(appEnvVars))
	for i, ev := range appEnvVars {
		appVars[i] = envresolve.AppVar{Key: ev.Key, Value: ev.Value}
	}

	// Check if we need to resolve template references
	needsShared, needsSiblings := false, false
	for _, v := range appVars {
		if strings.Contains(v.Value, "${{") {
			if strings.Contains(v.Value, "shared.") {
				needsShared = true
			}
			// Check for sibling refs (any ${{ that's not shared. and not a bare ref)
			needsSiblings = true
		}
	}

	var sharedVars []envresolve.AppVar
	if needsShared {
		envVars, err := db.Query.FindEnvironmentVariablesByEnvironmentId(ctx, s.db.RO(), env.ID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to fetch shared environment variables: %w", err))
		}
		sharedVars = make([]envresolve.AppVar, len(envVars))
		for i, ev := range envVars {
			sharedVars[i] = envresolve.AppVar{Key: ev.Key, Value: ev.Value}
		}
	}

	var siblingVars []envresolve.SiblingVar
	if needsSiblings {
		sibVars, err := db.Query.FindSiblingAppVarsByProjectAndEnv(ctx, s.db.RO(), db.FindSiblingAppVarsByProjectAndEnvParams{
			ProjectID:     project.ID,
			EnvironmentID: env.ID,
		})
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to fetch sibling app variables: %w", err))
		}
		siblingVars = make([]envresolve.SiblingVar, len(sibVars))
		for i, sv := range sibVars {
			siblingVars[i] = envresolve.SiblingVar{
				AppSlug: sv.AppSlug,
				Key:     sv.Key,
				Value:   sv.Value,
			}
		}
	}

	// Resolve templates
	resolvedVars, err := envresolve.Resolve(appVars, sharedVars, siblingVars)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("failed to resolve environment variable templates: %w", err))
	}

	// Build secrets blob from resolved vars
	secretsBlob := []byte{}
	if len(resolvedVars) > 0 {
		secretsConfig := &ctrlv1.SecretsConfig{
			Secrets: resolvedVars,
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

	logger.Info("deployment will use prebuilt image",
		"deployment_id", deploymentID,
		"app_id", app.ID,
		"image", dockerImage)

	// Insert deployment into database, snapshotting settings from app
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   workspaceID,
		ProjectID:                     req.Msg.GetProjectId(),
		AppID:                         app.ID,
		EnvironmentID:                 env.ID,
		OpenapiSpec:                   sql.NullString{String: "", Valid: false},
		SentinelConfig:                appRuntimeSettings.SentinelConfig,
		EncryptedEnvironmentVariables: secretsBlob,
		Command:                       appRuntimeSettings.Command,
		Status:                        db.DeploymentsStatusPending,
		CreatedAt:                     now,
		UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
		GitCommitSha:                  sql.NullString{String: gitCommitSha, Valid: gitCommitSha != ""},
		GitBranch:                     sql.NullString{String: gitBranch, Valid: true},
		GitCommitMessage:              sql.NullString{String: gitCommitMessage, Valid: gitCommitMessage != ""},
		GitCommitAuthorHandle:         sql.NullString{String: gitCommitAuthorHandle, Valid: gitCommitAuthorHandle != ""},
		GitCommitAuthorAvatarUrl:      sql.NullString{String: gitCommitAuthorAvatarURL, Valid: gitCommitAuthorAvatarURL != ""},
		GitCommitTimestamp:            sql.NullInt64{Int64: gitCommitTimestamp, Valid: gitCommitTimestamp != 0},
		CpuMillicores:                 appRuntimeSettings.CpuMillicores,
		MemoryMib:                     appRuntimeSettings.MemoryMib,
		Port:                          appRuntimeSettings.Port,
		ShutdownSignal:                db.DeploymentsShutdownSignal(appRuntimeSettings.ShutdownSignal),
		Healthcheck:                   appRuntimeSettings.Healthcheck,
	})
	if err != nil {
		logger.Error("failed to insert deployment", "error", err.Error())
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	logger.Info("starting deployment workflow",
		"deployment_id", deploymentID,
		"workspace_id", workspaceID,
		"project_id", req.Msg.GetProjectId(),
		"app_id", app.ID,
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

	// Send deployment request asynchronously (fire-and-forget), keyed by app ID
	invocation, err := s.deploymentClient(app.ID).
		Deploy().
		Send(ctx, deployReq)
	if err != nil {
		logger.Error("failed to start deployment workflow", "error", err)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("unable to start workflow: %w", err))
	}

	logger.Info("deployment workflow started",
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
