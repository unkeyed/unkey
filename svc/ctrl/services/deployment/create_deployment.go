package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/deploy/imageref"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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

// CreateDeployment creates a new deployment record and initiates an async Restate
// workflow. When source is omitted, Git apps deploy HEAD of their default branch,
// OCI apps deploy their configured default image, and historical untyped apps
// infer a source from their repository or current deployment.
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

	ociImage := req.Msg.GetOciImage()
	if ociImage != "" && req.Msg.GetDockerImage() != "" {
		return nil, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("oci_image and deprecated docker_image are mutually exclusive"))
	}
	if ociImage == "" {
		ociImage = req.Msg.GetDockerImage()
	}

	deploymentID, err := s.createAndDeploy(ctx, createParams{
		context:       ctxLoad,
		ociImage:      ociImage,
		gitCommit:     req.Msg.GetGitCommit(),
		command:       req.Msg.GetCommand(),
		trigger:       triggerFromProto(req.Msg.GetTrigger()),
		triggeredBy:   req.Msg.GetTriggeredBy(),
		triggerReason: req.Msg.GetTriggerReason(),
	})
	if err != nil {
		return nil, err
	}

	if auditErr := s.recordCreateAudit(ctx, ctxLoad, deploymentID, req.Msg.GetActor()); auditErr != nil {
		logger.Error(
			"failed to write deployment.create audit log",
			"deployment_id", deploymentID,
			"error", auditErr,
		)
	}

	return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: deploymentID,
		Status:       ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING,
	}), nil
}

// ensureEnvironmentDeployable rejects an environment whose runtime or regional
// settings would fail the deploy pipeline, before the workflow is enqueued. This
// is the RPC-level enforcement point every caller (v2 API, deprecated deploy API,
// CLI, future internal callers) passes through, so an undeployable deployment
// never gets enqueued; the worker keeps the same checks as a backstop. Runtime
// bounds share deployfail.RuntimeViolations with the worker and the API pre-flight.
func (s *Service) ensureEnvironmentDeployable(ctx context.Context, dctx deploymentContext) error {
	messages := make([]string, 0)
	for _, v := range deployfail.RuntimeViolations(
		dctx.appRuntimeSettings.Port,
		dctx.appRuntimeSettings.CpuMillicores,
		dctx.appRuntimeSettings.MemoryMib,
	) {
		messages = append(messages, v.Message)
	}

	regional, err := s.db.FindAppRegionalSettingsByAppAndEnv(ctx, db.FindAppRegionalSettingsByAppAndEnvParams{
		AppID:         dctx.app.ID,
		EnvironmentID: dctx.env.ID,
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load regional settings: %w", err))
	}
	if !slices.ContainsFunc(regional, func(r db.FindAppRegionalSettingsByAppAndEnvRow) bool { return r.RegionCanSchedule }) {
		messages = append(messages, deployfail.MsgNoSchedulableRegions)
	}

	if len(messages) == 0 {
		return nil
	}
	return connect.NewError(connect.CodeFailedPrecondition,
		fmt.Errorf("environment %q is not deployable: %s", dctx.env.Slug, strings.Join(messages, "; ")))
}

// recordCreateAudit writes a deployment.create audit log attributed to the
// actor supplied on the request. Every surface routes through this RPC, so this
// is the single place deployments get audited. A nil actor (callers not yet
// passing one) falls back to the system actor via actor.AuditType.
func (s *Service) recordCreateAudit(
	ctx context.Context,
	dctx deploymentContext,
	deploymentID string,
	a *ctrlv1.ActorInfo,
) error {
	return s.auditlogs.Insert(ctx, nil, []auditlog.AuditLog{
		{
			Event:         auditlog.DeploymentCreateEvent,
			WorkspaceID:   dctx.workspaceID,
			Display:       fmt.Sprintf("Created deployment %s", deploymentID),
			ActorID:       a.GetId(),
			ActorType:     actor.AuditType(a.GetType()),
			ActorName:     a.GetName(),
			ActorMeta:     actor.Meta(a.GetMeta()),
			RemoteIP:      a.GetRemoteIp(),
			UserAgent:     a.GetUserAgent(),
			CorrelationID: "",
			Resources: []auditlog.AuditLogResource{
				{
					Type:        auditlog.DeploymentResourceType,
					ID:          deploymentID,
					Name:        "",
					DisplayName: deploymentID,
					Meta: map[string]any{
						"projectId":   dctx.project.ID,
						"appId":       dctx.app.ID,
						"environment": dctx.env.Slug,
					},
				},
			},
		},
	})
}

// deploymentContext bundles the resolved project/app/env context needed to
// create a deployment. Loaded once at the RPC boundary and passed to the
// shared createAndDeploy helper.
type deploymentContext struct {
	project            db.Project
	workspaceID        string
	env                deploymentEnvironment
	app                deploymentApp
	appRuntimeSettings appRuntimeSettings
	secretsBlob        []byte
}

type deploymentEnvironment struct {
	ID   string
	Slug string
}

type deploymentApp struct {
	ID                  string
	SourceType          db.AppsSourceType
	CurrentDeploymentID sql.NullString
}

type appRuntimeSettings struct {
	Port             int32
	CpuMillicores    int32
	MemoryMib        int32
	StorageMib       uint32
	Command          mysqltype.StringSlice
	Healthcheck      mysqltype.NullHealthcheck
	ShutdownSignal   db.AppRuntimeSettingsShutdownSignal
	UpstreamProtocol db.AppRuntimeSettingsUpstreamProtocol
	SentinelConfig   []byte
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

	appWithSettings, err := s.db.FindAppWithRuntimeSettings(ctx, db.FindAppWithRuntimeSettingsParams{
		ID:            appID,
		EnvironmentID: env.ID,
	})
	if err != nil && db.IsNotFound(err) {
		return deploymentContext{}, connect.NewError(connect.CodeNotFound,
			fmt.Errorf("app '%s' not found or missing settings", appID))
	}
	if err != nil {
		return deploymentContext{}, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup app: %w", err))
	}

	// These records are resolved independently, so verify their ownership
	// before persisting a mismatched project/app/environment tuple.
	if appWithSettings.AppProjectID != project.ID {
		return deploymentContext{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("app %q does not belong to project %q", appID, project.ID))
	}
	if env.ProjectID != project.ID {
		return deploymentContext{}, connect.NewError(connect.CodeInvalidArgument,
			fmt.Errorf("environment %q does not belong to project %q", envSlug, project.ID))
	}

	appEnvVars, err := s.db.FindAppEnvVarsByAppAndEnv(ctx, db.FindAppEnvVarsByAppAndEnvParams{
		AppID:         appWithSettings.AppID,
		EnvironmentID: env.ID,
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
		project:     project,
		workspaceID: project.WorkspaceID,
		env:         deploymentEnvironment{ID: env.ID, Slug: envSlug},
		app: deploymentApp{
			ID:                  appWithSettings.AppID,
			SourceType:          appWithSettings.AppSourceType,
			CurrentDeploymentID: appWithSettings.AppCurrentDeploymentID,
		},
		appRuntimeSettings: appRuntimeSettings{
			Port: appWithSettings.RuntimeSettingsPort, CpuMillicores: appWithSettings.RuntimeSettingsCpuMillicores,
			MemoryMib: appWithSettings.RuntimeSettingsMemoryMib, StorageMib: appWithSettings.RuntimeSettingsStorageMib,
			Command: appWithSettings.RuntimeSettingsCommand, Healthcheck: appWithSettings.RuntimeSettingsHealthcheck,
			ShutdownSignal: appWithSettings.RuntimeSettingsShutdownSignal, UpstreamProtocol: appWithSettings.RuntimeSettingsUpstreamProtocol,
			SentinelConfig: appWithSettings.RuntimeSettingsSentinelConfig,
		},
		secretsBlob: secretsBlob,
	}, nil
}

// createParams carries everything createAndDeploy needs from a caller.
type createParams struct {
	context deploymentContext

	ociImage  string
	gitCommit *ctrlv1.GitCommitInfo
	command   []string

	// Attribution persisted on the deployment row.
	trigger       db.DeploymentsTrigger
	triggeredBy   string
	triggerReason string
}

func (s *Service) createAndDeploy(ctx context.Context, p createParams) (string, error) {
	c := p.context
	if err := s.ensureWorkspaceCanDeploy(ctx, c.workspaceID); err != nil {
		return "", err
	}
	if err := s.ensureEnvironmentDeployable(ctx, c); err != nil {
		return "", err
	}

	deploymentID := uid.New(uid.DeploymentPrefix)
	now := time.Now().UnixMilli()

	// Per-request command override (CLI/API) wins over the app's stored
	// default. Persisting only the default would mean the row disagrees with
	// what's actually running, which breaks rebuild and post-mortem flows.
	command := c.appRuntimeSettings.Command
	if len(p.command) > 0 {
		command = p.command
	}

	var deployReq *hydrav1.DeployRequest
	deploymentSource := db.DeploymentsSourceUnknown
	requestedImage := ""

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

	repoConn, repoErr := s.db.FindGithubRepoConnectionByAppId(ctx, c.app.ID)
	hasRepoConnection := repoErr == nil
	if repoErr != nil && !db.IsNotFound(repoErr) {
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup github repo connection: %w", repoErr))
	}

	useGit := false
	switch {
	case p.ociImage != "":
		imageReference, imageErr := imageref.Normalize(p.ociImage)
		if imageErr != nil {
			return "", connect.NewError(connect.CodeInvalidArgument, imageErr)
		}
		deploymentSource = db.DeploymentsSourceOci
		requestedImage = imageReference
		logger.Info("deployment will use prebuilt image",
			"deployment_id", deploymentID,
			"app_id", c.app.ID,
			"image", requestedImage)

		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_OciImage{
				OciImage: &hydrav1.OciImage{
					Image: requestedImage,
				},
			},
		}

	case explicitGit && c.app.SourceType == db.AppsSourceTypeOci:
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("OCI-sourced app %q cannot deploy a Git commit", c.app.ID))

	case explicitGit && !hasRepoConnection:
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has no GitHub repo connection; cannot deploy requested git commit", c.app.ID))

	case explicitGit:
		useGit = true

	case c.app.SourceType == db.AppsSourceTypeGit:
		if !hasRepoConnection {
			return "", connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("GitHub-sourced app %q has no repository connection", c.app.ID))
		}
		useGit = true

	case c.app.SourceType == db.AppsSourceTypeOci:
		ociSource, ociErr := s.db.FindAppSourceOciByAppId(ctx, c.app.ID)
		if ociErr != nil {
			if db.IsNotFound(ociErr) {
				return "", connect.NewError(connect.CodeFailedPrecondition,
					fmt.Errorf("OCI-sourced app %q has no source configuration", c.app.ID))
			}
			return "", connect.NewError(connect.CodeInternal,
				fmt.Errorf("failed to load OCI source: %w", ociErr))
		}
		imageReference, imageErr := imageref.Normalize(ociSource.ImageReference)
		if imageErr != nil {
			return "", connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("OCI source for app %q is invalid: %w", c.app.ID, imageErr))
		}
		commit = commitFields{ //nolint:exhaustruct
		}
		deploymentSource = db.DeploymentsSourceOci
		requestedImage = imageReference
		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_OciImage{
				OciImage: &hydrav1.OciImage{Image: requestedImage},
			},
		}

	case hasRepoConnection:
		useGit = true

	default:
		imageReference, ociErr := buildOciSource(ctx, s.db, c.app.ID, c.app.CurrentDeploymentID, deploymentID)
		if ociErr != nil {
			return "", ociErr
		}
		commit = commitFields{ //nolint:exhaustruct
		}
		deploymentSource = db.DeploymentsSourceOci
		requestedImage = imageReference
		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_OciImage{
				OciImage: &hydrav1.OciImage{Image: requestedImage},
			},
		}
	}

	if useGit {
		buildSettings, buildErr := s.db.FindAppBuildSettingByAppEnv(ctx, db.FindAppBuildSettingByAppEnvParams{
			AppID:         c.app.ID,
			EnvironmentID: c.env.ID,
		})
		if buildErr != nil {
			if db.IsNotFound(buildErr) {
				return "", connect.NewError(connect.CodeFailedPrecondition,
					fmt.Errorf("GitHub-sourced app %q has no build settings for environment %q", c.app.ID, c.env.Slug))
			}
			return "", connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load build settings: %w", buildErr))
		}

		if commit.SHA == "" && commit.Branch == "" {
			commit.Branch = defaultBranch(repoConn.DefaultBranch)
		}
		if err := commit.fillFromGitHub(
			s.github, repoConn.InstallationID, repoConn.RepositoryFullName,
			s.allowUnauthenticatedDeployments,
		); err != nil {
			// This error may carry the raw GitHub response body, which can reach
			// API callers. Log the detail, return a generic reason.
			logger.Error("failed to resolve git commit metadata",
				"app_id", c.app.ID,
				"repository", repoConn.RepositoryFullName,
				"error", err.Error())
			return "", connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("failed to resolve git commit metadata for the requested branch or commit"))
		}
		deployReq = &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      command,
			Source: &hydrav1.DeployRequest_Git{
				Git: &hydrav1.GitSource{
					InstallationId: repoConn.InstallationID,
					Repository:     repoConn.RepositoryFullName,
					CommitSha:      commit.SHA,
					ContextPath:    buildSettings.DockerContext,
					DockerfilePath: buildSettings.Dockerfile.String,
					BuildCommand:   buildSettings.BuildCommand.String,
					Branch:         commit.Branch,
					ForkRepository: commit.ForkRepository,
					PrNumber:       0,
				},
			},
		}
		deploymentSource = db.DeploymentsSourceGit
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
		EnvironmentID:                 c.env.ID,
		Source:                        deploymentSource,
		ImageRequested:                sql.NullString{String: requestedImage, Valid: requestedImage != ""},
		SentinelConfig:                c.appRuntimeSettings.SentinelConfig,
		EncryptedEnvironmentVariables: c.secretsBlob,
		Command:                       command,
		Status:                        mysqltype.DeploymentsStatusPending,
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

	logger.Info(
		"starting deployment workflow",
		"deployment_id", deploymentID,
		"workspace_id", c.workspaceID,
		"project_id", c.project.ID,
		"app_id", c.app.ID,
		"environment", c.env.ID,
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
			Status:    mysqltype.DeploymentsStatusFailed,
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
		logger.Error(
			"failed to persist invocation id",
			"deployment_id", deploymentID,
			"invocation_id", invocationID,
			"error", updateErr,
		)
	}

	logger.Info(
		"deployment workflow started",
		"deployment_id", deploymentID,
		"invocation_id", invocationID,
	)

	if cancelErr := s.dedup.CancelOlderSiblings(ctx, dedup.Newer{
		ID:            deploymentID,
		AppID:         c.app.ID,
		EnvironmentID: c.env.ID,
		GitBranch:     commit.Branch,
		CreatedAt:     now,
	}); cancelErr != nil {
		logger.Error(
			"failed to cancel superseded siblings",
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

func defaultBranch(connectionDefault sql.NullString) string {
	if connectionDefault.Valid && connectionDefault.String != "" {
		return connectionDefault.String
	}
	return "main"
}

func buildOciSource(
	ctx context.Context,
	database db.Database,
	appID string,
	currentDeploymentID sql.NullString,
	deploymentID string,
) (string, error) {
	if !currentDeploymentID.Valid || currentDeploymentID.String == "" {
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has no current deployment and no git connection; cannot redeploy", appID))
	}

	currentDeployment, err := database.FindDeploymentById(ctx, currentDeploymentID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return "", connect.NewError(connect.CodeNotFound,
				fmt.Errorf("current deployment %q not found", currentDeploymentID.String))
		}
		return "", connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to lookup current deployment: %w", err))
	}

	resolvedImage := resolvedDeploymentImage(currentDeployment)
	if !resolvedImage.Valid || resolvedImage.String == "" {
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("current deployment %q has no OCI image; cannot redeploy without git connection",
				currentDeploymentID.String))
	}

	logger.Info("deployment will reuse current deployment image",
		"deployment_id", deploymentID,
		"current_deployment_id", currentDeploymentID.String,
		"image", resolvedImage.String)

	imageReference, err := imageref.NormalizeHistorical(resolvedImage.String)
	if err != nil {
		return "", connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("current deployment %q has an invalid OCI image: %w", currentDeploymentID.String, err))
	}
	return imageReference, nil
}

func resolvedDeploymentImage(deployment db.Deployment) sql.NullString {
	if deployment.ImageResolved.Valid && deployment.ImageResolved.String != "" {
		return deployment.ImageResolved
	}
	return deployment.Image
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
