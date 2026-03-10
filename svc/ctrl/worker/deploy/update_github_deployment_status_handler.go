package deploy

import (
	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/logger"
)

// UpdateGitHubDeploymentStatus looks up a deployment's GitHub deployment ID and
// repo connection, then reports the requested state to the GitHub Deployments API.
// This runs on the worker which owns the GitHub App credentials.
func (w *Workflow) UpdateGitHubDeploymentStatus(ctx restate.ObjectContext, req *hydrav1.UpdateGitHubDeploymentStatusRequest) (*hydrav1.UpdateGitHubDeploymentStatusResponse, error) {
	deployment, err := restate.Run(ctx, func(ctx restate.RunContext) (*db.Deployment, error) {
		d, findErr := db.Query.FindDeploymentById(ctx, w.db.RO(), req.DeploymentId)
		if findErr != nil {
			return nil, findErr
		}
		return &d, nil
	})
	if err != nil {
		logger.Error("failed to find deployment for GitHub status update",
			"deployment_id", req.DeploymentId,
			"error", err,
		)
		return &hydrav1.UpdateGitHubDeploymentStatusResponse{}, nil
	}

	if !deployment.GithubDeploymentID.Valid {
		return &hydrav1.UpdateGitHubDeploymentStatusResponse{}, nil
	}

	repoConn, err := restate.Run(ctx, func(ctx restate.RunContext) (*db.GithubRepoConnection, error) {
		rc, findErr := db.Query.FindGithubRepoConnectionByAppId(ctx, w.db.RO(), deployment.AppID)
		if findErr != nil {
			return nil, findErr
		}
		return &rc, nil
	})
	if err != nil {
		logger.Error("failed to find repo connection for GitHub status update",
			"deployment_id", req.DeploymentId,
			"error", err,
		)
		return &hydrav1.UpdateGitHubDeploymentStatusResponse{}, nil
	}

	_ = restate.RunVoid(ctx, func(ctx restate.RunContext) error {
		return w.github.CreateDeploymentStatus(
			repoConn.InstallationID,
			repoConn.RepositoryFullName,
			deployment.GithubDeploymentID.Int64,
			req.State,
			"",
			"",
			req.Description,
		)
	})

	return &hydrav1.UpdateGitHubDeploymentStatusResponse{}, nil
}
