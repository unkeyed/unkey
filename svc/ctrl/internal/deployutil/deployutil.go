package deployutil

import (
	"context"
	"database/sql"
	"time"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/uid"
	"google.golang.org/protobuf/encoding/protojson"
)

// GitCommitInfo holds git commit metadata needed for creating a deployment.
type GitCommitInfo struct {
	SHA             string
	Branch          string
	Message         string
	AuthorHandle    string
	AuthorAvatarURL string
	Timestamp       int64 // Unix milliseconds, 0 if unknown
}

// BuildSecretsBlob marshals environment variables into a protobuf SecretsConfig blob.
// Returns an empty byte slice if there are no env vars.
func BuildSecretsBlob(envVars []db.ListEnvVarsForRepoConnectionsRow) ([]byte, error) {
	if len(envVars) == 0 {
		return []byte{}, nil
	}

	secretsConfig := &ctrlv1.SecretsConfig{
		Secrets: make(map[string]string, len(envVars)),
	}
	for _, ev := range envVars {
		secretsConfig.Secrets[ev.Key] = ev.Value
	}
	return protojson.Marshal(secretsConfig)
}

// InsertDeploymentRecord creates a deployment and its initial queued step in a single transaction.
func InsertDeploymentRecord(
	ctx context.Context,
	rw *db.Replica,
	row db.ListRepoConnectionDeployContextsRow,
	commit GitCommitInfo,
	secretsBlob []byte,
	status db.DeploymentsStatus,
) (string, error) {
	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	project := row.Project
	env := row.Environment
	app := row.App
	runtimeSettings := row.AppRuntimeSetting

	err := db.Tx(ctx, rw, func(txCtx context.Context, tx db.DBTX) error {
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
			Status:                        status,
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: false},
			GitCommitSha:                  sql.NullString{String: commit.SHA, Valid: commit.SHA != ""},
			GitBranch:                     sql.NullString{String: commit.Branch, Valid: commit.Branch != ""},
			GitCommitMessage:              sql.NullString{String: commit.Message, Valid: commit.Message != ""},
			GitCommitAuthorHandle:         sql.NullString{String: commit.AuthorHandle, Valid: commit.AuthorHandle != ""},
			GitCommitAuthorAvatarUrl:      sql.NullString{String: commit.AuthorAvatarURL, Valid: commit.AuthorAvatarURL != ""},
			GitCommitTimestamp:            sql.NullInt64{Int64: commit.Timestamp, Valid: commit.Timestamp != 0},
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
		return "", err
	}
	return deploymentID, nil
}

// BuildDeployRequest constructs a DeployRequest for a git-sourced deployment.
func BuildDeployRequest(
	deploymentID string,
	row db.ListRepoConnectionDeployContextsRow,
	commitSHA string,
) *hydrav1.DeployRequest {
	return &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		Source: &hydrav1.DeployRequest_Git{
			Git: &hydrav1.GitSource{
				InstallationId: row.GithubRepoConnection.InstallationID,
				Repository:     row.GithubRepoConnection.RepositoryFullName,
				CommitSha:      commitSHA,
				ContextPath:    row.AppBuildSetting.DockerContext,
				DockerfilePath: row.AppBuildSetting.Dockerfile,
			},
		},
	}
}

// GroupEnvVarsByApp groups environment variables by app ID for efficient lookup.
func GroupEnvVarsByApp(envVars []db.ListEnvVarsForRepoConnectionsRow) map[string][]db.ListEnvVarsForRepoConnectionsRow {
	result := make(map[string][]db.ListEnvVarsForRepoConnectionsRow)
	for _, ev := range envVars {
		result[ev.AppID] = append(result[ev.AppID], ev)
	}
	return result
}
