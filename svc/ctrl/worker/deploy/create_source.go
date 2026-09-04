package deploy

import (
	"context"
	"strings"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// gitCommit is the git metadata on a deployment row. Empty fields are filled
// from GitHub.
type gitCommit struct {
	SHA             string `json:"sha"`
	Branch          string `json:"branch"`
	Message         string `json:"message"`
	AuthorHandle    string `json:"author_handle"`
	AuthorAvatarURL string `json:"author_avatar_url"`
	Timestamp       int64  `json:"timestamp"`
	ForkRepository  string `json:"fork_repository"`
}

// buildSource is what the deployment builds from. Exactly one of Image and Git
// is set.
type buildSource struct {
	Image string     `json:"image"`
	Git   *gitSource `json:"git"`
}

type gitSource struct {
	InstallationID int64  `json:"installation_id"`
	Repository     string `json:"repository"`
	ContextPath    string `json:"context_path"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildCommand   string `json:"build_command"`
	PRNumber       int64  `json:"pr_number"`
}

// resolvedSource is a source with its completed commit, or a rejection.
type resolvedSource struct {
	Source    buildSource                    `json:"source"`
	Commit    gitCommit                      `json:"commit"`
	Rejection *hydrav1.CreateRejectionReason `json:"rejection"`
}

func newRejectedSource(rejected *hydrav1.CreateRejectionReason) resolvedSource {
	var refused resolvedSource
	refused.Rejection = rejected
	return refused
}

// resolveSource picks what the deployment builds from and completes the commit
// metadata from GitHub. Without an explicit source the app's declared source
// decides.
func (w *Workflow) resolveSource(
	ctx context.Context,
	target db.FindDeployTargetRow,
	req *hydrav1.DeployCreateRequest,
	commit gitCommit,
) (resolvedSource, error) {
	switch source := req.GetSource().(type) {
	case *hydrav1.DeployCreateRequest_Image:
		// The caller's commit is recorded for display. None is looked up for an image.
		return imageSource(source.Image.GetImage(), commit, imageref.Normalize), nil

	case *hydrav1.DeployCreateRequest_Git:
		return w.resolveGitSource(target, commit, source.Git.GetPrNumber())

	case *hydrav1.DeployCreateRequest_ExistingDeployment:
		return w.resolveExistingDeployment(ctx, target,
			source.ExistingDeployment.GetDeploymentId(), source.ExistingDeployment.GetRequireNoNewer())

	default:
		if target.SourceType == db.AppsSourceTypeGit {
			return w.resolveGitSource(target, commit, 0)
		}
		if target.SourceType == db.AppsSourceTypeOci {
			if target.OciImageReference.String == "" {
				return newRejectedSource(rejectf(
					hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE,
					"OCI app %s has no image configured", target.AppID,
				)), nil
			}
			return imageSource(target.OciImageReference.String, commit, imageref.Normalize), nil
		}

		if target.GithubRepositoryFullName.Valid {
			return w.resolveGitSource(target, commit, 0)
		}
		if !target.CurrentDeploymentID.Valid || target.CurrentDeploymentID.String == "" {
			return newRejectedSource(rejectf(
				hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE,
				"app %s has no current deployment to redeploy and the request named no source",
				target.AppID,
			)), nil
		}
		// The current deployment is the newest by definition, so requireNoNewer
		// has nothing to check.
		return w.resolveExistingDeployment(ctx, target, target.CurrentDeploymentID.String, false)
	}
}

// imageSource normalizes a prebuilt image reference. Deploy refuses an implicit tag, so
// a reference the caller chose gets [imageref.Normalize] and one read off an old
// row gets [imageref.NormalizeHistorical].
func imageSource(image string, commit gitCommit, normalize func(string) (string, error)) resolvedSource {
	normalized, err := normalize(image)
	if err != nil {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_INVALID_IMAGE,
			"%s", err.Error(),
		))
	}
	return resolvedSource{
		Source:    buildSource{Image: normalized, Git: nil},
		Commit:    commit,
		Rejection: nil,
	}
}

// resolveGitSource resolves the app's repository connection and fills in the
// commit metadata the caller did not supply.
func (w *Workflow) resolveGitSource(
	target db.FindDeployTargetRow,
	commit gitCommit,
	prNumber int64,
) (resolvedSource, error) {
	// A connection can outlive a switch to OCI, so the declared source wins.
	if target.SourceType == db.AppsSourceTypeOci {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_REPO_CONNECTION,
			"app %s deploys an OCI image and has no repository to build from", target.AppID,
		)), nil
	}
	if !target.GithubRepositoryFullName.Valid {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_REPO_CONNECTION,
			"app %s has no GitHub repo connection", target.AppID,
		)), nil
	}
	// An app created as OCI gets no build settings row.
	if !target.HasBuildSettings {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_ENVIRONMENT_NOT_DEPLOYABLE,
			"environment %q of app %s has no build settings", target.EnvironmentSlug, target.AppID,
		)), nil
	}

	// A sha alone gets no default branch: it may not be on it, and sibling dedup
	// keys on branch, so a wrong branch is worse than none.
	if commit.SHA == "" && commit.Branch == "" {
		// GitHub reported the connection's branch. apps.default_branch is a
		// placeholder on newer apps.
		commit.Branch = target.GithubDefaultBranch.String
		if commit.Branch == "" {
			commit.Branch = target.DefaultBranch
		}
		if commit.Branch == "" {
			commit.Branch = "main"
		}
	}

	if fillErr := w.fillCommitFromGitHub(&commit, target); fillErr != nil {
		// The GitHub error can carry a raw response body, so it is logged here
		// and never returned.
		logger.Error(
			"failed to resolve git commit metadata",
			"app_id", target.AppID,
			"repository", target.GithubRepositoryFullName.String,
			"error", fillErr.Error(),
		)
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_COMMIT_NOT_RESOLVED,
			"could not resolve branch %q or commit %q in %s",
			commit.Branch, commit.SHA, target.GithubRepositoryFullName.String,
		)), nil
	}

	return resolvedSource{
		Source: buildSource{
			Image: "",
			Git: &gitSource{
				InstallationID: target.GithubInstallationID.Int64,
				Repository:     target.GithubRepositoryFullName.String,
				ContextPath:    target.DockerContext.String,
				DockerfilePath: target.Dockerfile.String,
				BuildCommand:   target.BuildCommand.String,
				PRNumber:       prNumber,
			},
		},
		Commit:    commit,
		Rejection: nil,
	}, nil
}

// resolveExistingDeployment reproduces what another deployment ran. The row's
// recorded source decides: a git build is rebuilt from its commit, an image
// deployment redeploys its image even if it recorded the commit it came from.
func (w *Workflow) resolveExistingDeployment(
	ctx context.Context,
	target db.FindDeployTargetRow,
	deploymentID string,
	requireNoNewer bool,
) (resolvedSource, error) {
	var failed resolvedSource

	src, err := w.db.FindDeploymentById(ctx, deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return newRejectedSource(rejectf(
				hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SOURCE_DEPLOYMENT_NOT_FOUND,
				"source deployment %s not found", deploymentID,
			)), nil
		}
		return failed, err
	}

	// The lookup is by primary key alone, so this is what stops a request from
	// rebuilding another workspace's deployment. A mismatch answers like a miss
	// so the reason never confirms that a foreign deployment exists.
	//
	// Environment is not compared. A request with no source rebuilds the app's
	// current deployment, which is always a production one, into whatever
	// environment the request targets.
	if src.WorkspaceID != target.WorkspaceID || src.ProjectID != target.ProjectID ||
		src.AppID != target.AppID {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SOURCE_DEPLOYMENT_NOT_FOUND,
			"source deployment %s does not belong to app %s",
			deploymentID, target.AppID,
		)), nil
	}

	if requireNoNewer {
		hasNewer, newerErr := w.db.HasNewerActiveDeployment(ctx, db.HasNewerActiveDeploymentParams{
			AppID:         src.AppID,
			EnvironmentID: src.EnvironmentID,
			GitBranch:     src.GitBranch,
			CreatedAt:     src.CreatedAt,
			DeploymentID:  src.ID,
		})
		if newerErr != nil {
			return failed, newerErr
		}
		if hasNewer {
			return newRejectedSource(rejectf(
				hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NEWER_DEPLOYMENT_EXISTS,
				"a newer active deployment exists for app %s, environment %s, branch %q",
				src.AppID, src.EnvironmentID, src.GitBranch.String,
			)), nil
		}
	}

	commit := gitCommit{
		SHA:             src.GitCommitSha.String,
		Branch:          src.GitBranch.String,
		Message:         src.GitCommitMessage.String,
		AuthorHandle:    src.GitCommitAuthorHandle.String,
		AuthorAvatarURL: src.GitCommitAuthorAvatarUrl.String,
		Timestamp:       src.GitCommitTimestamp.Int64,
		ForkRepository:  src.ForkRepositoryFullName.String,
	}

	// A row from before sources were recorded is a git build if it has a commit.
	gitBuild := src.Source == db.DeploymentsSourceGit ||
		(src.Source == db.DeploymentsSourceUnknown && commit.SHA != "")
	if gitBuild && commit.SHA != "" && target.GithubRepositoryFullName.Valid {
		return w.resolveGitSource(target, commit, src.PrNumber.Int64)
	}
	// A rebuild that reused the image would rebuild nothing. Only a row that
	// never recorded its source may fall back to one.
	if src.Source == db.DeploymentsSourceGit {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_REPO_CONNECTION,
			"git deployment %s has no commit and repository connection to rebuild from", src.ID,
		)), nil
	}

	// Deploy pins the digest it ran into image_resolved. Rows from before that
	// hold only image, possibly with an implicit tag.
	image := src.ImageResolved.String
	if image == "" {
		image = src.Image.String
	}
	if image == "" {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE,
			"deployment %s has neither a rebuildable commit nor an image",
			src.ID,
		)), nil
	}

	logger.Info(
		"deployment will reuse an existing deployment's image",
		"source_deployment_id", src.ID,
		"image", image,
	)
	return imageSource(image, commit, imageref.NormalizeHistorical), nil
}

// fillCommitFromGitHub fills the commit's empty fields from GitHub. The public
// API has no lookup by sha, so a sha is only completed when authenticated.
func (w *Workflow) fillCommitFromGitHub(commit *gitCommit, target db.FindDeployTargetRow) error {
	installationID := target.GithubInstallationID.Int64

	hasAuth := !w.allowUnauthenticatedDeployments || installationID != noInstallationID

	resolveRepo := target.GithubRepositoryFullName.String
	if commit.ForkRepository != "" {
		resolveRepo = commit.ForkRepository
	}

	var info githubclient.CommitInfo
	var err error

	switch {
	case commit.SHA == "":
		if commit.Branch == "" {
			return nil
		}
		if hasAuth {
			info, err = w.github.GetBranchHeadCommit(installationID, resolveRepo, commit.Branch)
		} else {
			info, err = w.github.GetBranchHeadCommitPublic(resolveRepo, commit.Branch)
		}
	case commit.Message == "" && hasAuth:
		info, err = w.github.GetCommitBySHA(installationID, resolveRepo, commit.SHA)
	default:
		return nil
	}
	if err != nil {
		return err
	}

	if commit.SHA == "" {
		commit.SHA = info.SHA
	}
	if commit.Message == "" {
		commit.Message = info.Message
	}
	if commit.AuthorHandle == "" {
		commit.AuthorHandle = strings.TrimSpace(info.AuthorHandle)
	}
	if commit.AuthorAvatarURL == "" {
		commit.AuthorAvatarURL = strings.TrimSpace(info.AuthorAvatarURL)
	}
	if commit.Timestamp == 0 && !info.Timestamp.IsZero() {
		commit.Timestamp = info.Timestamp.UnixMilli()
	}
	return nil
}
