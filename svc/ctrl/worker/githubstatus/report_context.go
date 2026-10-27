package githubstatus

import (
	"errors"

	restate "github.com/restatedev/sdk-go"
	"github.com/unkeyed/unkey/pkg/notification"
	"github.com/unkeyed/unkey/pkg/notification/deployment"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

type reportContext struct {
	InstallationID     int64  `json:"installationId"`
	RepositoryID       int64  `json:"repositoryId"`
	Repository         string `json:"repository"`
	DeploymentOrder    uint64 `json:"deploymentOrder"`
	GitHubDeploymentID int64  `json:"githubDeploymentId"`
	PullRequest        int    `json:"pullRequest"`
	CommitSHA          string `json:"commitSha"`
	Branch             string `json:"branch"`
	Production         bool   `json:"production"`
}

func (s *Service) resolveReport(ctx restate.RunContext, event notification.Event, payload deployment.Payload) (reportContext, error) {
	var target reportContext
	row, err := s.db.FindDeploymentById(ctx, payload.DeploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return target, restate.ToTerminalError(errors.New("deployment not found"))
		}
		return target, err
	}
	if row.WorkspaceID != event.WorkspaceID || row.ProjectID != event.Resource.ProjectID ||
		row.AppID != event.Resource.AppID || row.EnvironmentID != event.Resource.EnvironmentID {
		return target, restate.ToTerminalError(errors.New("deployment notification scope mismatch"))
	}
	connection, err := s.db.FindGithubRepoConnectionByAppId(ctx, row.AppID)
	if db.IsNotFound(err) {
		return target, nil
	}
	if err != nil {
		return target, err
	}
	if connection.WorkspaceID != row.WorkspaceID || connection.ProjectID != row.ProjectID {
		return target, restate.ToTerminalError(errors.New("GitHub connection scope mismatch"))
	}
	environment, err := s.db.FindDeploymentForDeploy(ctx, row.ID)
	if err != nil {
		return target, err
	}
	return reportContext{
		InstallationID: connection.InstallationID, RepositoryID: connection.RepositoryID, Repository: connection.RepositoryFullName,
		DeploymentOrder: row.Pk, GitHubDeploymentID: row.GithubDeploymentID.Int64,
		PullRequest: int(row.PrNumber.Int64), CommitSHA: row.GitCommitSha.String, Branch: row.GitBranch.String,
		Production: environment.EnvironmentKind.IsProduction(),
	}, nil
}
