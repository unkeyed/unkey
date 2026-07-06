package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	githubclient "github.com/unkeyed/unkey/svc/ctrl/worker/github"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	// maxCommitMessageLength limits commit messages to prevent oversized database entries.
	maxCommitMessageLength = 10240
	// maxCommitAuthorHandleLength limits author handles (e.g., GitHub usernames).
	maxCommitAuthorHandleLength = 256
	// maxCommitAuthorAvatarLength limits avatar URL length.
	maxCommitAuthorAvatarLength = 512
	// maxTriggerReasonLength matches the trigger_reason column width.
	// Truncate at the boundary so a verbose operator note doesn't fail
	// the insert under MySQL strict mode.
	maxTriggerReasonLength = 512
	// noInstallationID is the zero value for a GitHub App installation ID.
	// When the caller has no installation we can only fall back to the public
	// GitHub API (and only if unauthenticated deployments are enabled).
	noInstallationID = int64(0)
)

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

// CreateDeployment creates a new deployment record and initiates an async Restate
// workflow. When source is omitted, the handler auto-detects: git-connected
// apps deploy HEAD of their default branch, non-git apps reuse the live
// deployment's Docker image.
//
// The workflow runs asynchronously keyed by {app, environment}, so different
// environments (e.g. prod vs preview) for the same app deploy in parallel while
// lifecycle operations within one environment remain serialized. Workspace-wide
// build concurrency is enforced separately via BuildSlotService. Returns the
// deployment ID and initial status.
func (s *Service) CreateDeployment(
	ctx context.Context,
	req *connect.Request[ctrlv1.CreateDeploymentRequest],
) (*connect.Response[ctrlv1.CreateDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	if req.Msg.GetProjectId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("project_id is required"))
	}

	appID := req.Msg.GetAppId()
	if appID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("app_id is required"))
	}

	ctxLoad, err := s.loadDeploymentContext(ctx, req.Msg.GetProjectId(), appID, req.Msg.GetEnvironmentSlug())
	if err != nil {
		return nil, err
	}

	keyspaceID := req.Msg.GetKeyspaceId()
	var keyAuthID *string
	if keyspaceID != "" {
		keyAuthID = &keyspaceID
	}

	deploymentID, err := s.createAndDeploy(ctx, createParams{
		context:       ctxLoad,
		dockerImage:   req.Msg.GetDockerImage(),
		gitCommit:     req.Msg.GetGitCommit(),
		keyAuthID:     keyAuthID,
		command:       req.Msg.GetCommand(),
		trigger:       triggerFromProto(req.Msg.GetTrigger()),
		triggeredBy:   req.Msg.GetTriggeredBy(),
		triggerReason: req.Msg.GetTriggerReason(),
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: deploymentID,
		Status:       ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}

// deploymentContext bundles the resolved project/app/env context needed to
// create a deployment. Loaded once at the RPC boundary and passed to the
// shared createAndDeploy helper.
type deploymentContext struct {
	project            db.Project
	workspaceID        string
	env                db.FindEnvironmentByAppIdAndSlugRow
	app                db.App
	appBuildSettings   db.AppBuildSetting
	appRuntimeSettings db.AppRuntimeSetting
	secretsBlob        []byte
}

// loadDeploymentContext resolves project, app, environment, settings, and
// app-scoped env vars into a single bundle. Used by both CreateDeployment
// (external) and RebuildDeployment (internal recovery) so neither RPC
// has to reimplement the lookup chain.
func (s *Service) loadDeploymentContext(
	ctx context.Context,
	projectID, appID, envSlug string,
) (deploymentContext, error) {
	project, err := s.db.FindProjectById(ctx, projectID)
	if err != nil {
		if db.IsNotFound(err) {
			return deploymentContext{}, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("project not found: %s", projectID))
		}
		return deploymentContext{}, connect.NewError(connect.CodeInternal, err)
	}

	env, err := s.db.FindEnvironmentByAppIdAndSlug(ctx, db.FindEnvironmentByAppIdAndSlugParams{
		AppID: appID,
		Slug:  envSlug,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return deploymentContext{}, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("environment '%s' not found for app '%s'", envSlug, appID))
		}
		return deploymentContext{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup environment: %w", err))
	}

	appWithSettings, err := s.db.FindAppWithSettings(ctx, db.FindAppWithSettingsParams{
		ID:            appID,
		EnvironmentID: env.Environment.ID,
	})
	if err != nil && db.IsNotFound(err) {
		return deploymentContext{}, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("app '%s' not found or missing settings", appID))
	}
	if err != nil {
		return deploymentContext{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup app: %w", err))
	}

	// All three records are resolved independently, so verify they belong to
	// the same project before letting a deployment row inherit a mismatched
	// (project_id, app_id, environment_id) triple. External entry points
	// (v2 API, webhook, dashboard) already guarantee this via workspace-scoped
	// joins; the guard catches future internal callers that pass IDs directly.
	if appWithSettings.App.ProjectID != project.ID {
		return deploymentContext{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("app %q does not belong to project %q", appID, project.ID))
	}
	if env.Environment.ProjectID != project.ID {
		return deploymentContext{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("environment %q does not belong to project %q", envSlug, project.ID))
	}

	appEnvVars, err := s.db.FindAppEnvVarsByAppAndEnv(ctx, db.FindAppEnvVarsByAppAndEnvParams{
		AppID:         appWithSettings.App.ID,
		EnvironmentID: env.Environment.ID,
	})
	if err != nil {
		return deploymentContext{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to fetch app environment variables: %w", err))
	}

	secretsBlob := []byte{}
	if len(appEnvVars) > 0 {
		secretsConfig := &ctrlv1.SecretsConfig{
			Secrets: make(map[string]string, len(appEnvVars)),
		}
		for _, ev := range appEnvVars {
			if !validation.IsValidEnvVarKey(ev.Key) {
				return deploymentContext{}, connect.NewError(connect.CodeInvalidArgument,
					fmt.Errorf("environment variable key %q is invalid: %s", ev.Key, validation.ErrMsgInvalidEnvVarKey))
			}
			secretsConfig.Secrets[ev.Key] = ev.Value
		}

		secretsBlob, err = protojson.Marshal(secretsConfig)
		if err != nil {
			return deploymentContext{}, connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to marshal secrets config: %w", err))
		}
	}

	return deploymentContext{
		project:            project,
		workspaceID:        project.WorkspaceID,
		env:                env,
		app:                appWithSettings.App,
		appBuildSettings:   appWithSettings.AppBuildSetting,
		appRuntimeSettings: appWithSettings.AppRuntimeSetting,
		secretsBlob:        secretsBlob,
	}, nil
}

// createParams carries everything createAndDeploy needs from a caller.
type createParams struct {
	context deploymentContext

	// Source overrides. dockerImage wins if set; otherwise we auto-detect
	// from git repo connection (using gitCommit.commit_sha if provided) or
	// fall back to the live deployment's image.
	dockerImage string
	gitCommit   *ctrlv1.GitCommitInfo
	keyAuthID   *string
	command     []string

	// Attribution persisted on the deployment row.
	trigger       db.DeploymentsTrigger
	triggeredBy   string
	triggerReason string
}

// createAndDeploy is the shared path used by both CreateDeployment and
// RebuildDeployment. It resolves the source (docker image / git / fallback),
// inserts the deployment row, kicks off the Restate workflow, persists the
// invocation id, and cancels superseded siblings.
func (s *Service) createAndDeploy(ctx context.Context, p createParams) (string, error) {
	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	c := p.context

	// Per-request command override (CLI/API) wins over the app's stored
	// default. Persisting only the default would mean the row disagrees with
	// what's actually running, which breaks rebuild and post-mortem flows.
	command := c.appRuntimeSettings.Command
	if len(p.command) > 0 {
		command = p.command
	}

	var deployReq *hydrav1.DeployRequest

	// Populate caller-provided commit metadata. Branch defaulting and GitHub
	// fill-in happen later, only when we're actually building from git — we
	// don't want to synthesize git metadata on docker-image redeploys.
	var commit commitFields
	gc := p.gitCommit
	explicitGit := gc != nil
	if gc != nil {
		commit.Branch = strings.TrimSpace(gc.GetBranch())
		commit.SHA = gc.GetCommitSha()
		commit.Message = trimLength(gc.GetCommitMessage(), maxCommitMessageLength)
		commit.AuthorHandle = trimLength(strings.TrimSpace(gc.GetAuthorHandle()), maxCommitAuthorHandleLength)
		commit.AuthorAvatarURL = trimLength(strings.TrimSpace(gc.GetAuthorAvatarUrl()), maxCommitAuthorAvatarLength)
		commit.Timestamp = gc.GetTimestamp()
		commit.ForkRepository = gc.GetForkRepository()
	}

	// Look up the GitHub repo connection once. Used both to decide source type
	// (git vs docker) and to resolve missing commit metadata synchronously.
	repoConn, repoErr := s.db.FindGithubRepoConnectionByAppId(ctx, c.app.ID)
	hasRepoConnection := repoErr == nil
	if repoErr != nil && !db.IsNotFound(repoErr) {
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup github repo connection: %w", repoErr))
	}

	switch {
	case p.dockerImage != "":
		// Explicit docker image (CLI, REST API): skip rebuild, redeploy as-is.
		// Don't touch git metadata — the caller owns whatever they passed.
		logger.Info("deployment will use prebuilt image",
			"deployment_id", deploymentID,
			"app_id", c.app.ID,
			"image", p.dockerImage)

		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			KeyAuthId:    p.keyAuthID,
			Command:      command,
			Source: &hydrav1.DeployRequest_DockerImage{
				DockerImage: &hydrav1.DockerImage{
					Image: p.dockerImage,
				},
			},
		}

	case explicitGit && !hasRepoConnection:
		// Caller asked for a specific commit, but the app has no git
		// connection. Refuse rather than silently redeploying the current
		// image (a different artifact than what was requested).
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has no GitHub repo connection; cannot deploy requested git commit", c.app.ID))

	case hasRepoConnection:
		// Git-connected app: fill missing commit metadata synchronously so
		// the deployment row is complete at insert time and buildImage can
		// run without any GitHub calls.
		// Only default to the app's default branch when neither SHA nor branch
		// were provided. If the caller pinned a SHA without a branch, that SHA
		// may live on a non-default branch: defaulting would record a wrong
		// branch alongside the right SHA.
		if commit.SHA == "" && commit.Branch == "" {
			commit.Branch = defaultBranch(c.app.DefaultBranch)
		}
		if err := commit.fillFromGitHub(
			s.github, repoConn.InstallationID, repoConn.RepositoryFullName,
			s.allowUnauthenticatedDeployments,
		); err != nil {
			return "", connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("failed to resolve git commit metadata: %w", err))
		}
		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			KeyAuthId:    p.keyAuthID,
			Command:      command,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repoConn.InstallationID,
					Repository:     repoConn.RepositoryFullName,
					CommitSha:      commit.SHA,
					ContextPath:    c.appBuildSettings.DockerContext,
					DockerfilePath: c.appBuildSettings.Dockerfile.String,
					BuildCommand:   c.appBuildSettings.BuildCommand.String,
					Branch:         commit.Branch,
					ForkRepository: commit.ForkRepository,
					PrNumber:       0,
				},
			},
		}

	default:
		// No docker image, no git commit, no repo connection: reuse current
		// deployment's image.
		dockerInfo, dockerErr := buildDockerSource(ctx, s.db, c.app, deploymentID)
		if dockerErr != nil {
			return "", dockerErr
		}
		commit = dockerInfo.commitFields

		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			KeyAuthId:    p.keyAuthID,
			Command:      command,
			Source: &hydrav1.DeployRequest_DockerImage{
				DockerImage: &hydrav1.DockerImage{
					Image: dockerInfo.dockerImage,
				},
			},
		}
	}

	trigger := p.trigger
	if trigger == "" {
		trigger = db.DeploymentsTriggerUnknown
	}

	// Truncate operator-supplied reason to the column width so a long
	// note doesn't bubble up as a 500 from MySQL.
	triggerReason := trimLength(p.triggerReason, maxTriggerReasonLength)

	err := s.db.InsertDeployment(ctx, db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   c.workspaceID,
		ProjectID:                     c.project.ID,
		AppID:                         c.app.ID,
		EnvironmentID:                 c.env.Environment.ID,
		SentinelConfig:                c.appRuntimeSettings.SentinelConfig,
		EncryptedEnvironmentVariables: c.secretsBlob,
		Command:                       command,
		Status:                        db.DeploymentsStatusPending,
		CreatedAt:                     now,
		UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
		GitCommitSha:                  sql.NullString{String: commit.SHA, Valid: commit.SHA != ""},
		GitBranch:                     sql.NullString{String: commit.Branch, Valid: commit.Branch != ""},
		GitCommitMessage:              sql.NullString{String: commit.Message, Valid: commit.Message != ""},
		GitCommitAuthorHandle:         sql.NullString{String: commit.AuthorHandle, Valid: commit.AuthorHandle != ""},
		GitCommitAuthorAvatarUrl:      sql.NullString{String: commit.AuthorAvatarURL, Valid: commit.AuthorAvatarURL != ""},
		GitCommitTimestamp:            sql.NullInt64{Int64: commit.Timestamp, Valid: commit.Timestamp != 0},
		CpuMillicores:                 c.appRuntimeSettings.CpuMillicores,
		MemoryMib:                     c.appRuntimeSettings.MemoryMib,
		StorageMib:                    c.appRuntimeSettings.StorageMib,
		Port:                          c.appRuntimeSettings.Port,
		ShutdownSignal:                db.DeploymentsShutdownSignal(c.appRuntimeSettings.ShutdownSignal),
		UpstreamProtocol:              db.DeploymentsUpstreamProtocol(c.appRuntimeSettings.UpstreamProtocol),
		Healthcheck:                   c.appRuntimeSettings.Healthcheck,
		PrNumber:                      sql.NullInt64{Int64: 0, Valid: false},
		ForkRepositoryFullName:        sql.NullString{String: commit.ForkRepository, Valid: commit.ForkRepository != ""},
		DeploymentTrigger:             trigger,
		TriggeredBy:                   sql.NullString{String: p.triggeredBy, Valid: p.triggeredBy != ""},
		TriggerReason:                 sql.NullString{String: triggerReason, Valid: triggerReason != ""},
	})
	if err != nil {
		logger.Error("failed to insert deployment", "error", err.Error())
		return "", connect.NewError(connect.CodeInternal, err)
	}

	logger.Info("starting deployment workflow",
		"deployment_id", deploymentID,
		"workspace_id", c.workspaceID,
		"project_id", c.project.ID,
		"app_id", c.app.ID,
		"environment", c.env.Environment.ID,
		"trigger", string(trigger),
	)

	// Send deployment request asynchronously, keyed by deployment_id —
	// each deployment runs as its own isolated workflow.
	invocation, err := s.deploymentClient(deploymentID).
		Deploy().
		Send(ctx, deployReq)
	if err != nil {
		logger.Error("failed to start deployment workflow", "error", err)

		updateErr := s.db.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
			ID:        deploymentID,
			Status:    db.DeploymentsStatusFailed,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			logger.Error("failed to mark deployment as failed", "deployment_id", deploymentID, "error", updateErr)
		}

		return "", connect.NewError(connect.CodeInternal, fmt.Errorf("unable to start workflow: %w", err))
	}

	invocationID := invocation.Id()
	if updateErr := s.db.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
		ID:           deploymentID,
		InvocationID: sql.NullString{Valid: true, String: invocationID},
		UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}); updateErr != nil {
		logger.Error("failed to persist invocation id",
			"deployment_id", deploymentID,
			"invocation_id", invocationID,
			"error", updateErr,
		)
	}

	logger.Info("deployment workflow started",
		"deployment_id", deploymentID,
		"invocation_id", invocationID,
	)

	if cancelErr := s.dedup.CancelOlderSiblings(ctx, dedup.Newer{
		ID:            deploymentID,
		AppID:         c.app.ID,
		EnvironmentID: c.env.Environment.ID,
		GitBranch:     commit.Branch,
		CreatedAt:     now,
	}); cancelErr != nil {
		logger.Error("failed to cancel superseded siblings",
			"deployment_id", deploymentID,
			"error", cancelErr,
		)
	}

	return deploymentID, nil
}

// triggerFromProto maps the proto enum to the db enum, defaulting to
// "unknown" for the unspecified case.
func triggerFromProto(t ctrlv1.DeploymentTrigger) db.DeploymentsTrigger {
	switch t {
	case ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_GITHUB:
		return db.DeploymentsTriggerGithub
	case ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API:
		return db.DeploymentsTriggerApi
	case ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI:
		return db.DeploymentsTriggerCli
	case ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_DASHBOARD:
		return db.DeploymentsTriggerDashboard
	case ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_UNKEY:
		return db.DeploymentsTriggerUnkey
	case ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_UNSPECIFIED:
		return db.DeploymentsTriggerUnknown
	default:
		return db.DeploymentsTriggerUnknown
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
	app db.App,
	deploymentID string,
) (dockerSourceInfo, error) {
	if !app.CurrentDeploymentID.Valid || app.CurrentDeploymentID.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has no current deployment and no git connection; cannot redeploy", app.ID))
	}

	currentDeployment, err := database.FindDeploymentById(ctx, app.CurrentDeploymentID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return dockerSourceInfo{}, connect.NewError(connect.CodeNotFound,
				fmt.Errorf("current deployment %q not found", app.CurrentDeploymentID.String))
		}
		return dockerSourceInfo{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup current deployment: %w", err))
	}

	if !currentDeployment.Image.Valid || currentDeployment.Image.String == "" {
		return dockerSourceInfo{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("current deployment %q has no Docker image; cannot redeploy without git connection",
				app.CurrentDeploymentID.String))
	}

	logger.Info("deployment will reuse current deployment image",
		"deployment_id", deploymentID,
		"current_deployment_id", app.CurrentDeploymentID.String,
		"image", currentDeployment.Image.String)

	return dockerSourceInfo{
		dockerImage: currentDeployment.Image.String,
		commitFields: commitFields{
			SHA:             currentDeployment.GitCommitSha.String,
			Branch:          currentDeployment.GitBranch.String,
			Message:         currentDeployment.GitCommitMessage.String,
			AuthorHandle:    currentDeployment.GitCommitAuthorHandle.String,
			AuthorAvatarURL: currentDeployment.GitCommitAuthorAvatarUrl.String,
			Timestamp:       currentDeployment.GitCommitTimestamp.Int64,
			ForkRepository:  currentDeployment.ForkRepositoryFullName.String,
		},
	}, nil
}

// trimLength truncates s to at most maxBytes bytes while preserving valid
// UTF-8: if the byte limit lands inside a multi-byte rune, the truncation
// happens at the previous rune boundary instead. This matters for columns
// where MySQL strict mode rejects malformed UTF-8.
func trimLength(s string, maxBytes int) string {
	if len(s) <= maxBytes {
		return s
	}
	cut := maxBytes
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}

// fillFromGitHub fills any empty fields by fetching commit metadata from
// GitHub. No-op when there's nothing worth fetching. The public (unauth)
// path has no lookup-by-SHA, so that branch is skipped when we can't
// authenticate (matches the previous behavior in deploy_handler.buildImage).
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
		cf.Message = trimLength(info.Message, maxCommitMessageLength)
	}
	if cf.AuthorHandle == "" {
		cf.AuthorHandle = trimLength(strings.TrimSpace(info.AuthorHandle), maxCommitAuthorHandleLength)
	}
	if cf.AuthorAvatarURL == "" {
		cf.AuthorAvatarURL = trimLength(strings.TrimSpace(info.AuthorAvatarURL), maxCommitAuthorAvatarLength)
	}
	if cf.Timestamp == 0 && !info.Timestamp.IsZero() {
		cf.Timestamp = info.Timestamp.UnixMilli()
	}
	return nil
}
