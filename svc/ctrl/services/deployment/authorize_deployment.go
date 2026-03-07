package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

// AuthorizeDeployment authorizes a deployment for an external contributor's push.
// It fetches the current HEAD of the branch from GitHub, re-derives all matching
// deploy contexts, creates deployment records, and fires deploy workflows.
func (s *Service) AuthorizeDeployment(ctx context.Context, req *connect.Request[ctrlv1.AuthorizeDeploymentRequest]) (*connect.Response[ctrlv1.AuthorizeDeploymentResponse], error) {
	projectID := req.Msg.GetProjectId()
	branch := req.Msg.GetBranch()

	if projectID == "" || branch == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("project_id and branch are required"))
	}

	// Find repo connection for this project
	repoConn, err := db.Query.FindGithubRepoConnectionByProjectId(ctx, s.db.RO(), projectID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no GitHub connection found for project %s", projectID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find repo connection: %w", err))
	}

	// Fetch current HEAD of the branch from GitHub — this is the SHA we deploy
	headCommit, err := s.github.GetBranchHeadCommit(repoConn.InstallationID, repoConn.RepositoryFullName, branch)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to fetch branch HEAD from GitHub: %w", err))
	}

	// Re-derive all matching deploy contexts (same query handle_push uses)
	contexts, err := db.Query.ListRepoConnectionDeployContexts(ctx, s.db.RO(), db.ListRepoConnectionDeployContextsParams{
		InstallationID: repoConn.InstallationID,
		RepositoryID:   repoConn.RepositoryID,
		Branch:         branch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list deploy contexts: %w", err))
	}

	if len(contexts) == 0 {
		return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("no deploy contexts found for project %s branch %s", projectID, branch))
	}

	// Fetch env vars for all matched apps
	allEnvVars, err := db.Query.ListEnvVarsForRepoConnections(ctx, s.db.RO(), db.ListEnvVarsForRepoConnectionsParams{
		InstallationID: repoConn.InstallationID,
		RepositoryID:   repoConn.RepositoryID,
		Branch:         branch,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to list env vars: %w", err))
	}

	envVarsByApp := make(map[string][]db.ListEnvVarsForRepoConnectionsRow)
	for _, ev := range allEnvVars {
		envVarsByApp[ev.AppID] = append(envVarsByApp[ev.AppID], ev)
	}

	commitTimestamp := headCommit.Timestamp.UnixMilli()

	for _, row := range contexts {
		project := row.Project
		env := row.Environment
		app := row.App
		runtimeSettings := row.AppRuntimeSetting
		buildSettings := row.AppBuildSetting
		repo := row.GithubRepoConnection

		// Build secrets blob from env vars
		appEnvVars := envVarsByApp[app.ID]
		secretsBlob := []byte{}
		if len(appEnvVars) > 0 {
			secretsConfig := &ctrlv1.SecretsConfig{
				Secrets: make(map[string]string, len(appEnvVars)),
			}
			for _, ev := range appEnvVars {
				secretsConfig.Secrets[ev.Key] = ev.Value
			}
			var marshalErr error
			secretsBlob, marshalErr = protojson.Marshal(secretsConfig)
			if marshalErr != nil {
				logger.Error("failed to marshal secrets config", "appId", app.ID, "error", marshalErr)
				continue
			}
		}

		deploymentID := uid.New(uid.DeploymentPrefix)
		now := time.Now().UnixMilli()

		err = db.Tx(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
			if txErr := db.Query.InsertDeployment(txCtx, tx, db.InsertDeploymentParams{
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
				GitCommitSha:                  sql.NullString{String: headCommit.SHA, Valid: true},
				GitBranch:                     sql.NullString{String: branch, Valid: true},
				GitCommitMessage:              sql.NullString{String: headCommit.Message, Valid: headCommit.Message != ""},
				GitCommitAuthorHandle:         sql.NullString{String: headCommit.AuthorHandle, Valid: headCommit.AuthorHandle != ""},
				GitCommitAuthorAvatarUrl:      sql.NullString{String: headCommit.AuthorAvatarURL, Valid: headCommit.AuthorAvatarURL != ""},
				GitCommitTimestamp:            sql.NullInt64{Int64: commitTimestamp, Valid: true},
				OpenapiSpec:                   sql.NullString{Valid: false},
				CpuMillicores:                 runtimeSettings.CpuMillicores,
				MemoryMib:                     runtimeSettings.MemoryMib,
				Port:                          runtimeSettings.Port,
				ShutdownSignal:                db.DeploymentsShutdownSignal(runtimeSettings.ShutdownSignal),
				Healthcheck:                   runtimeSettings.Healthcheck,
			}); txErr != nil {
				return txErr
			}

			return db.Query.InsertDeploymentStep(txCtx, tx, db.InsertDeploymentStepParams{
				WorkspaceID:   app.WorkspaceID,
				ProjectID:     app.ProjectID,
				AppID:         app.ID,
				EnvironmentID: env.ID,
				DeploymentID:  deploymentID,
				Step:          db.DeploymentStepsStepQueued,
				StartedAt:     uint64(now),
			})
		})
		if err != nil {
			logger.Error("failed to insert deployment for authorization", "appId", app.ID, "error", err)
			continue
		}

		// Fire deploy workflow via Restate
		_, sendErr := s.deploymentClient(project.ID).
			Deploy().
			Send(ctx, &hydrav1.DeployRequest{
				DeploymentId: deploymentID,
				Source: &hydrav1.DeployRequest_Git{
					Git: &hydrav1.GitSource{
						InstallationId: repo.InstallationID,
						Repository:     repo.RepositoryFullName,
						CommitSha:      headCommit.SHA,
						ContextPath:    buildSettings.DockerContext,
						DockerfilePath: buildSettings.Dockerfile,
					},
				},
			})
		if sendErr != nil {
			logger.Error("failed to trigger deploy workflow after authorization",
				"deployment_id", deploymentID,
				"error", sendErr,
			)
			continue
		}

		logger.Info("deployment authorized and workflow triggered",
			"deployment_id", deploymentID,
			"project_id", project.ID,
			"app_id", app.ID,
			"branch", branch,
			"commit_sha", headCommit.SHA,
		)
	}

	return connect.NewResponse(&ctrlv1.AuthorizeDeploymentResponse{}), nil
}
