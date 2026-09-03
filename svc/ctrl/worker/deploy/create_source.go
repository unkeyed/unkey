package deploy

import (
	"context"
	"strings"

	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	"github.com/unkeyed/unkey/pkg/fault"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// gitCommit holds the git metadata a deployment row records. An empty field
// means unknown and is eligible to be filled from GitHub.
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

// gitSource is where a git build reads its code from, with the repository
// connection already resolved from the app. The commit travels separately,
// because a rebuild carries one the caller never sent.
type gitSource struct {
	InstallationID int64  `json:"installation_id"`
	Repository     string `json:"repository"`
	ContextPath    string `json:"context_path"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildCommand   string `json:"build_command"`
	PRNumber       int64  `json:"pr_number"`
}

// resolvedSource is what [Workflow.resolveSource] decided: a source with the
// commit metadata it completed, or a rejection. Exactly one of those is set.
type resolvedSource struct {
	Source    buildSource `json:"source"`
	Commit    gitCommit   `json:"commit"`
	Rejection *rejection  `json:"rejection"`
}

// newRejectedSource is a refusal. Nothing is written, so it carries no source and
// no commit.
func newRejectedSource(rejected *rejection) resolvedSource {
	var refused resolvedSource
	refused.Rejection = rejected
	return refused
}

// deployRequest turns a resolved source back into the request Deploy consumes.
func (s buildSource) deployRequest(deploymentID string, command []string, commit gitCommit) *hydrav1.DeployRequest {
	if s.Git == nil {
		return &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_OciImage{
				OciImage: &hydrav1.OciImage{Image: s.Image},
			},
		}
	}

	return &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		Command:      command,
		Source: &hydrav1.DeployRequest_Git{
			Git: &hydrav1.GitSource{
				InstallationId: s.Git.InstallationID,
				Repository:     s.Git.Repository,
				CommitSha:      commit.SHA,
				ContextPath:    s.Git.ContextPath,
				DockerfilePath: s.Git.DockerfilePath,
				BuildCommand:   s.Git.BuildCommand,
				Branch:         commit.Branch,
				ForkRepository: commit.ForkRepository,
				PrNumber:       s.Git.PRNumber,
			},
		},
	}
}

// resolveSource picks what the deployment builds from and completes the commit
// metadata, so the row is whole at insert time and the build needs no further
// GitHub lookups.
//
// A refusal is a rejection, not an error: no connected repository, an
// unresolvable branch, a source with no artifact. Those stay distinguishable
// from a broken GitHub, which errors and is retried.
func (w *Workflow) resolveSource(
	ctx context.Context,
	target db.FindDeployTargetRow,
	req *hydrav1.DeployCreateRequest,
	commit gitCommit,
) (resolvedSource, error) {
	switch source := req.GetSource().(type) {
	case *hydrav1.DeployCreateRequest_Image:
		// A prebuilt image redeploys as it is. Synthesizing a commit would label
		// the row with code the image may not contain.
		image := source.Image.GetImage()
		if err := imageref.Validate(image); err != nil {
			// Otherwise the build fails on it later, leaving a row whose whole
			// story is a pull error.
			return newRejectedSource(rejectf(
				hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_INVALID_IMAGE,
				"%s", fault.UserFacingMessage(err),
			)), nil
		}
		return resolvedSource{
			Source:    buildSource{Image: image, Git: nil},
			Commit:    commit,
			Rejection: nil,
		}, nil

	case *hydrav1.DeployCreateRequest_Git:
		return w.resolveGitSource(target, commit, source.Git.GetPrNumber())

	case *hydrav1.DeployCreateRequest_ExistingDeployment:
		return w.resolveExistingDeployment(ctx, target,
			source.ExistingDeployment.GetDeploymentId(), source.ExistingDeployment.GetRequireNoNewer())

	default:
		// No source named means "ship this app again": a connected repository
		// answers with the head of its default branch, and only an app without
		// one falls back to what it runs now. This mirrors the legacy RPC.
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
		// Redeploying the live deployment is never an operator resurrecting an
		// older one, so the guardrail does not apply.
		return w.resolveExistingDeployment(ctx, target, target.CurrentDeploymentID.String, false)
	}
}

// resolveGitSource resolves the app's repository connection and fills in the
// commit metadata the caller did not supply.
func (w *Workflow) resolveGitSource(
	target db.FindDeployTargetRow,
	commit gitCommit,
	prNumber int64,
) (resolvedSource, error) {
	if !target.GithubRepositoryFullName.Valid {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_REPO_CONNECTION,
			"app %s has no GitHub repo connection", target.AppID,
		)), nil
	}

	// Only default the branch when the caller named neither. A commit pinned
	// without a branch may live off the default branch, and a wrong branch beside
	// a right commit is worse than none: sibling dedup keys on branch.
	if commit.SHA == "" && commit.Branch == "" {
		commit.Branch = target.DefaultBranch
		if commit.Branch == "" {
			commit.Branch = "main"
		}
	}

	if fillErr := commit.fillFromGitHub(
		w.github, target.GithubInstallationID.Int64, target.GithubRepositoryFullName.String,
		w.allowUnauthenticatedDeployments,
	); fillErr != nil {
		// The GitHub error can carry a raw response body. Log it and reject with
		// a reason, so nothing upstream echoes it back.
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
				ContextPath:    target.DockerContext,
				DockerfilePath: target.Dockerfile.String,
				BuildCommand:   target.BuildCommand.String,
				PRNumber:       prNumber,
			},
		},
		Commit:    commit,
		Rejection: nil,
	}, nil
}

// resolveExistingDeployment rebuilds what another deployment ran: its commit
// when the app still has a repository connection, otherwise its image.
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

	// The id comes from the request and is looked up by primary key alone, so
	// nothing else ties it to the app being deployed to: without this a request
	// could rebuild another workspace's deployment and run an image it has no
	// right to pull. A mismatch answers exactly like a miss, so the reason never
	// confirms that an unreachable deployment exists.
	//
	// Environment is deliberately excluded. Rebuilding across environments of one
	// app is legitimate, and the unset-source path relies on it: the app's
	// current deployment is only ever set for production.
	if src.WorkspaceID != target.WorkspaceID || src.ProjectID != target.ProjectID ||
		src.AppID != target.AppID {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SOURCE_DEPLOYMENT_NOT_FOUND,
			"source deployment %s does not belong to app %s",
			deploymentID, target.AppID,
		)), nil
	}

	// Guardrail for operator rebuilds: refuse to resurrect a deployment someone
	// has already shipped past.
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

	// A commit is only rebuildable while the app still has the connection to
	// fetch it from. Without one, use the image the source produced.
	if commit.SHA != "" {
		if target.GithubRepositoryFullName.Valid {
			return w.resolveGitSource(target, commit, src.PrNumber.Int64)
		}
	}

	if !src.Image.Valid || src.Image.String == "" {
		return newRejectedSource(rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_SOURCE_IMAGE,
			"deployment %s has neither a rebuildable commit nor an image",
			src.ID,
		)), nil
	}

	logger.Info(
		"deployment will reuse an existing deployment's image",
		"source_deployment_id", src.ID,
		"image", src.Image.String,
	)
	return resolvedSource{
		Source:    buildSource{Image: src.Image.String, Git: nil},
		Commit:    commit,
		Rejection: nil,
	}, nil
}

// fillFromGitHub fills empty fields from GitHub, and is a no-op when there is
// nothing worth fetching. The public path has no lookup-by-SHA, so that branch
// is skipped without authentication.
func (cf *gitCommit) fillFromGitHub(
	gh githubclient.GitHubClient,
	installationID int64,
	repo string,
	allowUnauth bool,
) error {
	// The public API is only for a repository with no installation, and only when
	// unauthenticated deployments are enabled.
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
