package deploy

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	restateadmin "github.com/unkeyed/unkey/pkg/restate/admin"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

// Build ends the queued step and runs the building step for a git source.
// Deploy calls it in [restateadmin.BuildConcurrencyScope] with the workspace
// id as the limit key, and Restate runs this handler only once the workspace
// is under its build cap. A pre-built image never reaches here; Deploy
// resolves it without taking a build slot.
//
// Build refuses a request outside the build scope or with a limit key other
// than its workspace: a wrong scope matches no rule and builds uncapped, a
// wrong limit key charges the build to another workspace. Restate picks the
// rule before it dispatches, so these checks report a mis-queued Build rather
// than prevent it. A missing rule is not detectable here; the cron owns that
func (w *Workflow) Build(ctx restate.WorkflowSharedContext, req *hydrav1.DeployRequest) (*hydrav1.BuildResponse, error) {
	if ctx.Request().Scope != restateadmin.BuildConcurrencyScope {
		return nil, fault.Wrap(
			restate.TerminalErrorf("build invoked in scope %q, want %q", ctx.Request().Scope, restateadmin.BuildConcurrencyScope),
			fault.Public("This build was not queued correctly."),
		)
	}

	deploymentID := restate.Key(ctx)
	if req.GetDeploymentId() != deploymentID {
		return nil, fault.Wrap(
			restate.TerminalErrorf("request deployment_id %q does not match workflow key %q", req.GetDeploymentId(), deploymentID),
			fault.Public("This build request is invalid."),
		)
	}

	deployment, err := restate.Run(ctx, func(runCtx restate.RunContext) (db.FindDeploymentForBuildRow, error) {
		return w.db.FindDeploymentForBuild(runCtx, deploymentID)
	}, restate.WithName("finding deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Failed to read from database. Please try again."))
	}

	if ctx.Request().LimitKey != deployment.WorkspaceID {
		return nil, fault.Wrap(
			restate.TerminalErrorf("build invoked with limit key %q, want workspace %q", ctx.Request().LimitKey, deployment.WorkspaceID),
			fault.Public("This build was not queued correctly."),
		)
	}

	// A cancel or supersede that landed while this Build was queued has already
	// ended the deployment, and deploycancel ended its open steps with the
	// reason. Deploy re-reads the row after Build and stops too
	if deployment.Status.IsTerminal() {
		logger.Info("deployment is already terminal, not building",
			"deployment_id", deployment.ID,
			"status", deployment.Status,
		)
		return &hydrav1.BuildResponse{}, nil
	}

	// Create opens the queued step when it writes the deployment row. It
	// ends here, once Restate has let this Build run. The end timestamp is
	// journalled so a retry of this handler reports the queue wait it
	// measured the first time round, not the time since
	queuedUntil, err := restate.Run(ctx, func(runCtx restate.RunContext) (int64, error) {
		now := time.Now().UnixMilli()
		return now, w.db.EndDeploymentStep(runCtx, db.EndDeploymentStepParams{
			DeploymentID: deployment.ID,
			Step:         db.DeploymentStepsStepQueued,
			EndedAt:      sql.NullInt64{Valid: true, Int64: now},
			Error:        sql.NullString{Valid: false, String: ""},
		})
	}, restate.WithName("end queued step"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fault.Wrap(err, fault.Public("Deployment could not be started."))
	}

	logger.Info("build starting",
		"workspace_id", deployment.WorkspaceID,
		"deployment_id", deployment.ID,
		"queued_for", time.Duration(queuedUntil-deployment.CreatedAt)*time.Millisecond,
	)

	stepErr := w.DeploymentStep(ctx, db.DeploymentStepsStepBuilding, deployment.ID, func() error {
		return w.buildImage(ctx, req, deployment)
	})
	if stepErr != nil {
		return nil, stepErr
	}

	return &hydrav1.BuildResponse{}, nil
}

// buildImage builds the container image on the configured build backend and
// persists the image reference and build ID to the database.
//
// The commit must already be resolved: Create does that before dispatching, so a
// request arriving here without a SHA is a bug and fails terminally.
//
// Returns a terminal error for a source that is not git, since only a git
// source needs building, and for build failures that cannot be retried
// (e.g. bad Dockerfile)
func (w *Workflow) buildImage(ctx restate.Context, req *hydrav1.DeployRequest, deployment db.FindDeploymentForBuildRow) error {
	source, isGit := req.GetSource().(*hydrav1.DeployRequest_Git)
	if !isGit {
		return fault.Wrap(
			restate.ToTerminalError(fmt.Errorf("build invoked for source type %T, only git needs building", req.GetSource())),
			fault.Public("This deployment source does not need a build."),
		)
	}

	commitSHA := source.Git.GetCommitSha()
	forkRepo := source.Git.GetForkRepository()

	if commitSHA == "" {
		return fault.Wrap(
			restate.ToTerminalError(fmt.Errorf("git source missing commit SHA for deployment %q", deployment.ID)),
			fault.Public("Deployment has no resolved commit; cannot build."),
		)
	}

	params := gitBuildParams{
		InstallationID: source.Git.GetInstallationId(),
		Repository:     source.Git.GetRepository(),
		ForkRepository: forkRepo,
		CommitSHA:      commitSHA,
		ContextPath:    source.Git.GetContextPath(),
		// Normalized here because the value routes the build method below:
		// a whitespace-only setting must mean "no Dockerfile configured"
		DockerfilePath: strings.TrimSpace(source.Git.GetDockerfilePath()),
		// Trimmed so a whitespace-only setting means "let Railpack auto-detect"
		BuildCommand:                  strings.TrimSpace(source.Git.GetBuildCommand()),
		ProjectID:                     deployment.ProjectID,
		AppID:                         deployment.AppID,
		DeploymentID:                  deployment.ID,
		WorkspaceID:                   deployment.WorkspaceID,
		PrNumber:                      source.Git.GetPrNumber(),
		EncryptedEnvironmentVariables: deployment.EncryptedEnvironmentVariables,
		EnvironmentID:                 deployment.EnvironmentID,
	}

	// The configured Dockerfile path decides the build method: when the
	// app's build settings name a Dockerfile it is used, otherwise the
	// app is built with Railpack (no Dockerfile required).
	var build *buildResult
	var err error
	if params.DockerfilePath == "" {
		logger.Info(
			"no dockerfile configured, building with railpack",
			"deployment_id", deployment.ID,
			"repository", params.Repository,
			"commit_sha", params.CommitSHA,
		)
		build, err = w.buildRailpackImageFromGit(ctx, params)
	} else {
		build, err = w.buildDockerImageFromGit(ctx, params)
	}
	if err != nil {
		// fault.Public set inside buildDockerImageFromGit is lost because
		// restate.Run serialises terminal errors, stripping the fault wrapper.
		// Re-extract the user message on this side of the Restate boundary.
		publicMsg := fault.UserFacingMessage(err)
		if publicMsg == "" {
			publicMsg = extractUserBuildError(err)
		}
		return fault.Wrap(
			fmt.Errorf("failed to build docker image from git: %w", err),
			fault.Public(publicMsg),
		)
	}
	resolvedImage := build.ImageName

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.UpdateDeploymentBuildID(runCtx, db.UpdateDeploymentBuildIDParams{
			ID:        deployment.ID,
			BuildID:   sql.NullString{Valid: true, String: build.BuildID},
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("update deployment build id"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return fault.Wrap(
			fmt.Errorf("failed to update deployment build ID: %w", err),
			fault.Public("Updating build metadata failed."),
		)
	}

	err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.UpdateDeploymentImage(runCtx, db.UpdateDeploymentImageParams{
			ID:            deployment.ID,
			ImageResolved: sql.NullString{Valid: true, String: resolvedImage},
			UpdatedAt:     sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("update deployment image"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return fault.Wrap(err, fault.Public("Unable to save deployment image."))
	}

	return nil
}
