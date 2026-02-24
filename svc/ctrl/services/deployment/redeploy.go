package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
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

// Redeploy triggers a fresh deployment for a project+environment pair.
//
// For git-connected projects the API sends a GitSource with a branch (and
// optionally a commit SHA) to the deploy worker. The worker resolves the branch
// HEAD via GitHub when no commit SHA is provided
//
// For non-git projects it reuses the live deployment's Docker image with
// refreshed env vars and settings.
func (s *Service) Redeploy(
	ctx context.Context,
	req *connect.Request[ctrlv1.RedeployRequest],
) (*connect.Response[ctrlv1.RedeployResponse], error) {
	if err := assert.All(
		assert.NotEmpty(req.Msg.GetProjectId()),
		assert.NotEmpty(req.Msg.GetEnvironmentSlug()),
	); err != nil {
		return nil, err
	}

	// Lookup project, environment, build/runtime settings, and env vars in one query
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

	// Try to find a GitHub repo connection for this project
	repoConn, err := db.Query.FindGithubRepoConnectionByProjectId(ctx, s.db.RO(), req.Msg.GetProjectId())
	hasGitConnection := err == nil
	if err != nil && !db.IsNotFound(err) {
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup github repo connection: %w", err))
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	var gitCommitSha, gitBranch, gitCommitMessage, gitCommitAuthorHandle, gitCommitAuthorAvatarURL string
	var gitCommitTimestamp int64
	var deploySource *hydrav1.DeployRequest

	if hasGitConnection {
		// Determine target branch and optional commit SHA from the request.
		// Branch resolution (looking up HEAD) happens in the deploy worker
		branch, commitSHA := resolveGitTarget(req.Msg, project)
		gitBranch = branch
		gitCommitSha = commitSHA

		deploySource = &hydrav1.DeployRequest{
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
	} else {
		// Non-git project: reuse the live deployment's Docker image
		source, dockerErr := buildDockerSource(ctx, s.db, project, deploymentID)
		if dockerErr != nil {
			return nil, dockerErr
		}
		gitCommitSha = source.commitSHA
		gitBranch = source.branch
		gitCommitMessage = source.commitMessage
		gitCommitAuthorHandle = source.authorHandle
		gitCommitAuthorAvatarURL = source.authorAvatarURL
		gitCommitTimestamp = source.commitTimestamp

		deploySource = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			KeyAuthId:    nil,
			Command:      nil,
			Source: &hydrav1.DeployRequest_DockerImage{
				DockerImage: &hydrav1.DockerImage{
					Image: source.dockerImage,
				},
			},
		}
	}

	// Insert deployment record
	err = db.Query.InsertDeployment(ctx, s.db.RW(), db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   project.WorkspaceID,
		ProjectID:                     project.ID,
		EnvironmentID:                 env.ID,
		OpenapiSpec:                   sql.NullString{Valid: false},
		SentinelConfig:                envSettings.EnvironmentRuntimeSetting.SentinelConfig,
		EncryptedEnvironmentVariables: secretsBlob,
		Command:                       envSettings.EnvironmentRuntimeSetting.Command,
		Status:                        db.DeploymentsStatusPending,
		CreatedAt:                     now,
		UpdatedAt:                     sql.NullInt64{Valid: false},
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

	logger.Info("starting redeploy workflow",
		"deployment_id", deploymentID,
		"project_id", project.ID,
		"environment", env.ID,
		"has_git_connection", hasGitConnection,
	)

	invocation, err := s.deploymentClient(project.ID).
		Deploy().
		Send(ctx, deploySource)
	if err != nil {
		logger.Error("failed to start redeploy workflow", "error", err)
		return nil, connect.NewError(connect.CodeInternal,
			fmt.Errorf("unable to start workflow: %w", err))
	}

	logger.Info("redeploy workflow started",
		"deployment_id", deploymentID,
		"invocation_id", invocation.Id,
	)

	return connect.NewResponse(&ctrlv1.RedeployResponse{
		DeploymentId: deploymentID,
		Status:       ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}

// resolveGitTarget extracts branch and commit SHA from the redeploy request.
// When a commit_sha is provided, branch defaults to the project's default
// branch for metadata. When only a branch is provided, commit SHA is left empty
// for the worker to resolve. When neither is provided, the project's default
// branch is used.
func resolveGitTarget(msg *ctrlv1.RedeployRequest, project db.Project) (branch string, commitSHA string) {
	defaultBranch := project.DefaultBranch.String
	if defaultBranch == "" {
		defaultBranch = "main"
	}

	switch target := msg.GetTarget().(type) {
	case *ctrlv1.RedeployRequest_CommitSha:
		return defaultBranch, target.CommitSha
	case *ctrlv1.RedeployRequest_Branch:
		return target.Branch, ""
	default:
		return defaultBranch, ""
	}
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

	logger.Info("redeploy will reuse live deployment image",
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
