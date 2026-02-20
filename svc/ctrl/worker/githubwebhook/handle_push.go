package githubwebhook

import (
	"database/sql"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

// HandlePush processes a GitHub push event durably via Restate. It looks up
// repo connections, resolves project/environment/app/settings, creates
// deployment records, and fires off DeployService.Deploy() for each deployment.
func (s *Service) HandlePush(ctx restate.ObjectContext, req *hydrav1.HandlePushRequest) (*hydrav1.HandlePushResponse, error) {
	logger.Info("handling GitHub push in Restate",
		"delivery_id", req.GetDeliveryId(),
		"repository", req.GetRepositoryFullName(),
		"branch", req.GetBranch(),
		"commit_sha", req.GetAfter(),
	)

	repoConnections, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.GithubRepoConnection, error) {
		return db.Query.ListGithubRepoConnections(runCtx, s.db.RO(), db.ListGithubRepoConnectionsParams{
			InstallationID: req.GetInstallationId(),
			RepositoryID:   req.GetRepositoryId(),
		})
	}, restate.WithName("list repo connections"))
	if err != nil {
		return nil, err
	}

	for _, repo := range repoConnections {
		project, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindProjectByIdRow, error) {
			return db.Query.FindProjectById(runCtx, s.db.RO(), repo.ProjectID)
		}, restate.WithName("find project"))
		if err != nil {
			if db.IsNotFound(err) {
				logger.Info("No project found for repo connection", "projectId", repo.ProjectID)
				continue
			}
			logger.Error("failed to find project for repo connection", "projectId", repo.ProjectID, "error", err)
			continue
		}

		defaultBranch := "main"
		if project.DefaultBranch.Valid && project.DefaultBranch.String != "" {
			defaultBranch = project.DefaultBranch.String
		}

		envSlug := "preview"
		if req.GetBranch() == defaultBranch {
			envSlug = "production"
		}

		env, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.Environment, error) {
			return db.Query.FindEnvironmentByProjectIdAndSlug(runCtx, s.db.RO(), db.FindEnvironmentByProjectIdAndSlugParams{
				WorkspaceID: project.WorkspaceID,
				ProjectID:   project.ID,
				Slug:        envSlug,
			})
		}, restate.WithName("find environment"))
		if err != nil {
			logger.Error("failed to find environment for repo connection", "projectId", repo.ProjectID, "appId", repo.AppID, "envSlug", envSlug, "error", err)
			continue
		}

		appRow, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindAppByIdRow, error) {
			return db.Query.FindAppById(runCtx, s.db.RO(), repo.AppID)
		}, restate.WithName("find app"))
		if err != nil {
			logger.Error("failed to find app for repo connection", "appId", repo.AppID, "error", err)
			continue
		}
		app := appRow.App

		runtimeRow, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindAppRuntimeSettingsByAppAndEnvRow, error) {
			return db.Query.FindAppRuntimeSettingsByAppAndEnv(runCtx, s.db.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
				AppID:         app.ID,
				EnvironmentID: env.ID,
			})
		}, restate.WithName("find runtime settings"))
		if err != nil {
			logger.Error("failed to find runtime settings", "appId", app.ID, "envId", env.ID, "error", err)
			continue
		}
		runtimeSettings := runtimeRow.AppRuntimeSetting

		buildRow, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindAppBuildSettingsByAppAndEnvRow, error) {
			return db.Query.FindAppBuildSettingsByAppAndEnv(runCtx, s.db.RO(), db.FindAppBuildSettingsByAppAndEnvParams{
				AppID:         app.ID,
				EnvironmentID: env.ID,
			})
		}, restate.WithName("find build settings"))
		if err != nil {
			logger.Error("failed to find build settings", "appId", app.ID, "envId", env.ID, "error", err)
			continue
		}
		buildSettings := buildRow.AppBuildSetting

		envVars, err := restate.Run(ctx, func(runCtx restate.RunContext) ([]db.FindAppEnvVarsByAppAndEnvRow, error) {
			return db.Query.FindAppEnvVarsByAppAndEnv(runCtx, s.db.RO(), db.FindAppEnvVarsByAppAndEnvParams{
				AppID:         app.ID,
				EnvironmentID: env.ID,
			})
		}, restate.WithName("find env vars"))
		if err != nil {
			logger.Error("failed to find env vars", "appId", app.ID, "envId", env.ID, "error", err)
			continue
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
				logger.Error("failed to marshal secrets config", "appId", app.ID, "error", err)
				continue
			}
		}

		// Create deployment record
		deploymentID := uid.New(uid.DeploymentPrefix)
		now := time.Now().UnixMilli()

		commitMessage := req.GetCommitMessage()
		authorHandle := req.GetCommitAuthorHandle()
		authorAvatarURL := req.GetCommitAuthorAvatarUrl()
		commitTimestamp := req.GetCommitTimestamp()

		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			return db.Query.InsertDeployment(runCtx, s.db.RW(), db.InsertDeploymentParams{
				ID:                            deploymentID,
				K8sName:                       uid.DNS1035(12),
				WorkspaceID:                   project.WorkspaceID,
				ProjectID:                     project.ID,
				AppID:                         app.ID,
				EnvironmentID:                 env.ID,
				SentinelConfig:                runtimeSettings.SentinelConfig,
				EncryptedEnvironmentVariables: secretsBlob,
				Command:                       runtimeSettings.Command,
				Status:                        db.DeploymentsStatusPending,
				CreatedAt:                     now,
				UpdatedAt:                     sql.NullInt64{Valid: false},
				GitCommitSha:                  sql.NullString{String: req.GetAfter(), Valid: req.GetAfter() != ""},
				GitBranch:                     sql.NullString{String: req.GetBranch(), Valid: req.GetBranch() != ""},
				GitCommitMessage:              sql.NullString{String: commitMessage, Valid: commitMessage != ""},
				GitCommitAuthorHandle:         sql.NullString{String: authorHandle, Valid: authorHandle != ""},
				GitCommitAuthorAvatarUrl:      sql.NullString{String: authorAvatarURL, Valid: authorAvatarURL != ""},
				GitCommitTimestamp:            sql.NullInt64{Int64: commitTimestamp, Valid: commitTimestamp != 0},
				OpenapiSpec:                   sql.NullString{Valid: false},
				CpuMillicores:                 runtimeSettings.CpuMillicores,
				MemoryMib:                     runtimeSettings.MemoryMib,
				Port:                          runtimeSettings.Port,
				ShutdownSignal:                db.DeploymentsShutdownSignal(runtimeSettings.ShutdownSignal),
				Healthcheck:                   runtimeSettings.Healthcheck,
			})
		}, restate.WithName("insert deployment"))
		if err != nil {
			logger.Error("failed to insert deployment", "appId", app.ID, "error", err)
			continue
		}

		logger.Info("Created deployment record",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", project.ID,
			"app_id", app.ID,
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
			"branch", req.GetBranch(),
			"environment", envSlug,
		)

		// Start deploy workflow with GitSource, keyed by app ID
		deployClient := hydrav1.NewDeployServiceClient(ctx, app.ID)
		deployClient.Deploy().Send(&hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repo.InstallationID,
					Repository:     req.GetRepositoryFullName(),
					CommitSha:      req.GetAfter(),
					ContextPath:    buildSettings.DockerContext,
					DockerfilePath: buildSettings.Dockerfile,
				},
			},
		})

		logger.Info("Deployment workflow started",
			"deployment_id", deploymentID,
			"delivery_id", req.GetDeliveryId(),
			"project_id", project.ID,
			"app_id", app.ID,
			"repository", req.GetRepositoryFullName(),
			"commit_sha", req.GetAfter(),
		)
	}

	return &hydrav1.HandlePushResponse{}, nil
}
