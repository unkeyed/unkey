package deploy

import (
	"context"
	"strings"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploytarget"
)

// commitFields holds the git metadata a deployment row records. An empty field
// means unknown and is eligible to be filled from GitHub. It crosses the Restate
// journal as JSON, so it holds plain scalars.
type commitFields struct {
	SHA             string `json:"sha"`
	Branch          string `json:"branch"`
	Message         string `json:"message"`
	AuthorHandle    string `json:"author_handle"`
	AuthorAvatarURL string `json:"author_avatar_url"`
	Timestamp       int64  `json:"timestamp"`
	ForkRepository  string `json:"fork_repository"`
}

// buildSource is what the deployment builds from, flattened for the journal
// like [commitFields]. Exactly one of Image and Git is set.
type buildSource struct {
	Image string    `json:"image"`
	Git   *gitBuild `json:"git"`
}

// gitBuild is a git build with the repository connection already resolved from
// the app.
type gitBuild struct {
	InstallationID int64  `json:"installation_id"`
	Repository     string `json:"repository"`
	ContextPath    string `json:"context_path"`
	DockerfilePath string `json:"dockerfile_path"`
	BuildCommand   string `json:"build_command"`
	PRNumber       int64  `json:"pr_number"`
}

// sourceResult is what [Workflow.resolveSource] decided: a source with the
// commit metadata it completed, or a block.
type sourceResult struct {
	Source buildSource  `json:"source"`
	Commit commitFields `json:"commit"`
	Block  *createBlock `json:"block"`
}

// newSourceResult is a resolved source ready to build.
func newSourceResult(source buildSource, commit commitFields) sourceResult {
	return sourceResult{Source: source, Commit: commit, Block: nil}
}

// blockedSource is a refusal. Nothing is written, so it carries no source and
// no commit.
func blockedSource(block *createBlock) sourceResult {
	return sourceResult{
		Source: buildSource{Image: "", Git: nil},
		Commit: commitFields{}, //nolint:exhaustruct // an empty commit is the point
		Block:  block,
	}
}

// deployRequest turns a resolved source back into the request Deploy consumes.
func (s buildSource) deployRequest(deploymentID string, command []string, commit commitFields) *hydrav1.DeployRequest {
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
// metadata that goes on the row, so the row is whole at insert time and the
// build needs no further GitHub lookups.
//
// A refusal comes back as a block, not an error: no connected repository, an
// unresolvable branch, a source deployment with no artifact. The caller has to
// tell those apart from a broken GitHub, which returns an error and is retried.
func (w *Workflow) resolveSource(
	ctx context.Context,
	target deploytarget.Target,
	req *hydrav1.DeployCreateRequest,
	commit commitFields,
) (sourceResult, error) {
	switch source := req.GetSource().(type) {
	case *hydrav1.DeployCreateRequest_Image:
		// A prebuilt image redeploys as it is. Synthesizing a commit would label
		// the row with code the image may not contain.
		return newSourceResult(buildSource{Image: source.Image.GetImage(), Git: nil}, commit), nil

	case *hydrav1.DeployCreateRequest_Git:
		return w.resolveGitSource(ctx, target, commit, source.Git.GetPrNumber())

	case *hydrav1.DeployCreateRequest_ExistingDeployment:
		return w.resolveExistingDeployment(ctx, target, req)

	default:
		// No source at all. resolveCreate rejects the empty result as a caller bug.
		return newSourceResult(buildSource{Image: "", Git: nil}, commit), nil
	}
}

// resolveGitSource resolves the app's repository connection and fills in the
// commit metadata the caller did not supply.
func (w *Workflow) resolveGitSource(
	ctx context.Context,
	target deploytarget.Target,
	commit commitFields,
	prNumber int64,
) (sourceResult, error) {
	if !target.GithubRepositoryFullName.Valid {
		return blockedSource(newBlockf(
			hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NO_REPO_CONNECTION,
			"app %s has no GitHub repo connection", target.AppID,
		)), nil
	}

	// Only default the branch when the caller named neither a commit nor a
	// branch. A commit pinned without a branch may live off the default branch,
	// and recording the wrong branch beside the right commit is worse than
	// recording no branch: sibling dedup keys on branch.
	if commit.SHA == "" && commit.Branch == "" {
		commit.Branch = defaultBranch(target.DefaultBranch)
	}

	if fillErr := commit.fillFromGitHub(
		w.github, target.GithubInstallationID.Int64, target.GithubRepositoryFullName.String,
		w.allowUnauthenticatedDeployments,
	); fillErr != nil {
		// The GitHub error can carry a raw response body. Log the detail and
		// block with a reason, so nothing upstream echoes it back.
		logger.Error("failed to resolve git commit metadata",
			"app_id", target.AppID,
			"repository", target.GithubRepositoryFullName.String,
			"error", fillErr.Error(),
		)
		return blockedSource(newBlockf(
			hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_COMMIT_NOT_RESOLVED,
			"could not resolve branch %q or commit %q in %s",
			commit.Branch, commit.SHA, target.GithubRepositoryFullName.String,
		)), nil
	}

	return newSourceResult(buildSource{
		Image: "",
		Git: &gitBuild{
			InstallationID: target.GithubInstallationID.Int64,
			Repository:     target.GithubRepositoryFullName.String,
			ContextPath:    target.DockerContext,
			DockerfilePath: target.Dockerfile.String,
			BuildCommand:   target.BuildCommand.String,
			PRNumber:       prNumber,
		},
	}, commit), nil
}

// resolveExistingDeployment rebuilds what another deployment ran: its commit
// when the app still has a repository connection, otherwise its image.
func (w *Workflow) resolveExistingDeployment(
	ctx context.Context,
	target deploytarget.Target,
	req *hydrav1.DeployCreateRequest,
) (sourceResult, error) {
	existing := req.GetExistingDeployment()

	src, err := w.db.FindDeploymentById(ctx, existing.GetDeploymentId())
	if err != nil {
		if db.IsNotFound(err) {
			return blockedSource(newBlockf(
				hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_TARGET_NOT_FOUND,
				"source deployment %s not found", existing.GetDeploymentId(),
			)), nil
		}
		return sourceResult{}, err //nolint:exhaustruct // zero value unused on error
	}

	// Guardrail for operator rebuilds: refuse to resurrect an older deployment
	// when someone has already shipped past it.
	if existing.GetRequireNoNewer() {
		hasNewer, newerErr := w.db.HasNewerActiveDeployment(ctx, db.HasNewerActiveDeploymentParams{
			AppID:         src.AppID,
			EnvironmentID: src.EnvironmentID,
			GitBranch:     src.GitBranch,
			CreatedAt:     src.CreatedAt,
			DeploymentID:  src.ID,
		})
		if newerErr != nil {
			return sourceResult{}, newerErr //nolint:exhaustruct // zero value unused on error
		}
		if hasNewer {
			return blockedSource(newBlockf(
				hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NEWER_DEPLOYMENT_EXISTS,
				"a newer active deployment exists for app %s, environment %s, branch %q",
				src.AppID, src.EnvironmentID, src.GitBranch.String,
			)), nil
		}
	}

	commit := commitFieldsFromDeployment(src)

	// A commit is only rebuildable while the app still has the connection to
	// fetch it from. Without one, fall back to the image the source produced.
	if commit.SHA != "" {
		if target.GithubRepositoryFullName.Valid {
			return w.resolveGitSource(ctx, target, commit, src.PrNumber.Int64)
		}
	}

	if !src.Image.Valid || src.Image.String == "" {
		return blockedSource(newBlockf(
			hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NO_SOURCE_IMAGE,
			"deployment %s has neither a rebuildable commit nor an image",
			src.ID,
		)), nil
	}

	logger.Info("deployment will reuse an existing deployment's image",
		"source_deployment_id", src.ID,
		"image", src.Image.String,
	)
	return newSourceResult(buildSource{Image: src.Image.String, Git: nil}, commit), nil
}

// defaultBranch returns the app's configured default branch, falling back to
// "main" when unset.
func defaultBranch(appDefault string) string {
	if appDefault != "" {
		return appDefault
	}
	return "main"
}

// commitFromProto maps caller-supplied commit metadata onto [commitFields],
// normalizing whitespace only. GitHub fill-in happens in resolveSource, so an
// image redeploy never synthesizes git metadata; truncation to the column widths
// happens at the database boundary.
func commitFromProto(gc *ctrlv1.GitCommitInfo) commitFields {
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

// fillFromGitHub fills empty fields from GitHub; a no-op when nothing is worth
// fetching. The public path has no lookup-by-SHA, so that branch is skipped
// without authentication.
func (cf *commitFields) fillFromGitHub(
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
