package deployment

import (
	"context"
	"fmt"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploytarget"
)

// noInstallationID is the zero value for a GitHub App installation ID.
// When the caller has no installation we can only fall back to the public
// GitHub API (and only if unauthenticated deployments are enabled).
const noInstallationID = int64(0)

// commitFields holds git commit metadata used on a deployment row. Empty
// fields mean "unknown" and are eligible to be filled from GitHub.
type commitFields struct {
	SHA             string
	Branch          string
	Message         string
	AuthorHandle    string
	AuthorAvatarURL string
	Timestamp       int64
	ForkRepository  string
}

// dockerSourceInfo holds the Docker image and inherited git metadata from a
// current deployment, used when redeploying a non-git project.
type dockerSourceInfo struct {
	commitFields
	dockerImage string
}

// resolveSource picks the build source: explicit docker image wins, explicit
// git without a repo connection is refused, a git-connected app builds from
// git (filling missing commit metadata from GitHub), everything else reuses
// the current deployment's image. The returned commitFields replace the
// caller's, because the fallback arm inherits them from the current
// deployment.
func (s *Service) resolveSource(
	ctx context.Context,
	c deploytarget.Target,
	deploymentID string,
	command []string,
	commit commitFields,
	dockerImage string,
	explicitGit bool,
) (*hydrav1.DeployRequest, commitFields, error) {
	// Look up the GitHub repo connection once. Used both to decide source type
	// (git vs docker) and to resolve missing commit metadata synchronously.
	repoConn, repoErr := s.db.FindGithubRepoConnectionByAppId(ctx, c.AppID)
	hasRepoConnection := repoErr == nil
	if repoErr != nil && !db.IsNotFound(repoErr) {
		return nil, commitFields{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup github repo connection: %w", repoErr))
	}

	switch {
	case dockerImage != "":
		// Explicit docker image (CLI, REST API): skip rebuild, redeploy as-is.
		// Don't touch git metadata — the caller owns whatever they passed.
		logger.Info("deployment will use prebuilt image",
			"deployment_id", deploymentID,
			"app_id", c.AppID,
			"image", dockerImage)

		return &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_DockerImage{
				DockerImage: &hydrav1.DockerImage{
					Image: dockerImage,
				},
			},
		}, commit, nil

	case explicitGit && !hasRepoConnection:
		// Caller asked for a specific commit, but the app has no git
		// connection. Refuse rather than silently redeploying the current
		// image (a different artifact than what was requested). Reaches the
		// v2 API as a 412.
		return nil, commitFields{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has no GitHub repo connection; cannot deploy requested git commit", c.AppID))

	case hasRepoConnection:
		// Git-connected app: fill missing commit metadata synchronously so
		// the deployment row is complete at insert time and buildImage can
		// run without any GitHub calls.
		// Only default to the app's default branch when neither SHA nor branch
		// were provided. If the caller pinned a SHA without a branch, that SHA
		// may live on a non-default branch: defaulting would record a wrong
		// branch alongside the right SHA.
		if commit.SHA == "" && commit.Branch == "" {
			commit.Branch = defaultBranch(c.DefaultBranch)
		}
		if err := commit.fillFromGitHub(
			s.github, repoConn.InstallationID, repoConn.RepositoryFullName,
			s.allowUnauthenticatedDeployments,
		); err != nil {
			// This error may carry the raw GitHub response body, which can reach
			// API callers. Log the detail, return a generic reason.
			logger.Error("failed to resolve git commit metadata",
				"app_id", c.AppID,
				"repository", repoConn.RepositoryFullName,
				"error", err.Error())
			return nil, commitFields{}, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("failed to resolve git commit metadata for the requested branch or commit"))
		}
		return &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repoConn.InstallationID,
					Repository:     repoConn.RepositoryFullName,
					CommitSha:      commit.SHA,
					ContextPath:    c.DockerContext,
					DockerfilePath: c.Dockerfile.String,
					BuildCommand:   c.BuildCommand.String,
					Branch:         commit.Branch,
					ForkRepository: commit.ForkRepository,
					PrNumber:       0,
				},
			},
		}, commit, nil

	default:
		// No docker image, no git commit, no repo connection: reuse current
		// deployment's image.
		dockerInfo, dockerErr := buildDockerSource(ctx, s.db, c, deploymentID)
		if dockerErr != nil {
			return nil, commitFields{}, dockerErr
		}

		return &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_DockerImage{
				DockerImage: &hydrav1.DockerImage{
					Image: dockerInfo.dockerImage,
				},
			},
		}, dockerInfo.commitFields, nil
	}
}

// defaultBranch returns the app's configured default branch, falling back
// to "main" when unset.
func defaultBranch(appDefault string) string {
	if appDefault != "" {
		return appDefault
	}
	return "main"
}

// buildDockerSource looks up the app's current deployment's Docker image and carries
// over its git metadata for the new deployment record.
func buildDockerSource(
	ctx context.Context,
	database db.Database,
	c deploytarget.Target,
	deploymentID string,
) (dockerSourceInfo, error) {
	if !c.CurrentDeploymentID.Valid || c.CurrentDeploymentID.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has no current deployment and no git connection; cannot redeploy", c.AppID))
	}

	currentDeployment, err := database.FindDeploymentById(ctx, c.CurrentDeploymentID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return dockerSourceInfo{}, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("current deployment %q not found", c.CurrentDeploymentID.String))
		}
		return dockerSourceInfo{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup current deployment: %w", err))
	}

	if !currentDeployment.Image.Valid || currentDeployment.Image.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("current deployment %q has no Docker image; cannot redeploy without git connection",
				c.CurrentDeploymentID.String))
	}

	logger.Info("deployment will reuse current deployment image",
		"deployment_id", deploymentID,
		"current_deployment_id", c.CurrentDeploymentID.String,
		"image", currentDeployment.Image.String)

	return dockerSourceInfo{
		dockerImage:  currentDeployment.Image.String,
		commitFields: commitFieldsFromDeployment(currentDeployment),
	}, nil
}

// commitFromRequest maps caller-provided commit metadata onto commitFields,
// normalizing whitespace only: GitHub fill-in happens in resolveSource
// (docker redeploys must not synthesize git metadata) and column-width
// truncation in the create worker, at the database boundary.
func commitFromRequest(gc *ctrlv1.GitCommitInfo) commitFields {
	if gc == nil {
		return commitFields{} //nolint:exhaustruct // empty fields mean "unknown" by contract
	}
	return commitFields{
		SHA:             gc.GetCommitSha(),
		Branch:          strings.TrimSpace(gc.GetBranch()),
		Message:         gc.GetCommitMessage(),
		AuthorHandle:    strings.TrimSpace(gc.GetAuthorHandle()),
		AuthorAvatarURL: strings.TrimSpace(gc.GetAuthorAvatarUrl()),
		Timestamp:       gc.GetTimestamp(),
		ForkRepository:  gc.GetForkRepository(),
	}
}

// commitFieldsFromDeployment reads the git metadata a deployment row records.
func commitFieldsFromDeployment(d db.Deployment) commitFields {
	return commitFields{
		SHA:             d.GitCommitSha.String,
		Branch:          d.GitBranch.String,
		Message:         d.GitCommitMessage.String,
		AuthorHandle:    d.GitCommitAuthorHandle.String,
		AuthorAvatarURL: d.GitCommitAuthorAvatarUrl.String,
		Timestamp:       d.GitCommitTimestamp.Int64,
		ForkRepository:  d.ForkRepositoryFullName.String,
	}
}

// fillFromGitHub fills empty fields from GitHub; no-op when nothing is worth
// fetching. The public (unauth) path has no lookup-by-SHA, so that branch is
// skipped without authentication.
func (cf *commitFields) fillFromGitHub(
	gh githubclient.GitHubClient,
	installationID int64,
	repo string,
	allowUnauth bool,
) error {
	// Use the authenticated GitHub path whenever a real installation is
	// available; only fall back to the public API when unauth is explicitly
	// enabled and we have no installation to auth with.
	hasAuth := !allowUnauth || installationID != noInstallationID

	resolveRepo := repo
	if cf.ForkRepository != "" {
		resolveRepo = cf.ForkRepository
	}

	var info githubclient.CommitInfo
	var err error

	switch {
	case cf.SHA == "":
		if cf.Branch == "" {
			return nil
		}
		if hasAuth {
			info, err = gh.GetBranchHeadCommit(installationID, resolveRepo, cf.Branch)
		} else {
			info, err = gh.GetBranchHeadCommitPublic(resolveRepo, cf.Branch)
		}
	case cf.Message == "" && hasAuth:
		info, err = gh.GetCommitBySHA(installationID, resolveRepo, cf.SHA)
	default:
		return nil
	}
	if err != nil {
		return err
	}

	if cf.SHA == "" {
		cf.SHA = info.SHA
	}
	if cf.Message == "" {
		cf.Message = info.Message
	}
	if cf.AuthorHandle == "" {
		cf.AuthorHandle = strings.TrimSpace(info.AuthorHandle)
	}
	if cf.AuthorAvatarURL == "" {
		cf.AuthorAvatarURL = strings.TrimSpace(info.AuthorAvatarURL)
	}
	if cf.Timestamp == 0 && !info.Timestamp.IsZero() {
		cf.Timestamp = info.Timestamp.UnixMilli()
	}
	return nil
}
