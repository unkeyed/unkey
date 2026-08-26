package deployment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/deploy/idempotency"
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

	res, err := s.createAndDeploy(ctx, createParams{
		context:        ctxLoad,
		action:         "create",
		actor:          req.Msg.GetActor(),
		dockerImage:    req.Msg.GetDockerImage(),
		gitCommit:      req.Msg.GetGitCommit(),
		command:        req.Msg.GetCommand(),
		trigger:        triggerFromProto(req.Msg.GetTrigger()),
		triggeredBy:    req.Msg.GetTriggeredBy(),
		triggerReason:  req.Msg.GetTriggerReason(),
		idempotencyKey: req.Msg.GetIdempotencyKey(),
	})
	if err != nil {
		return nil, err
	}

	return connect.NewResponse(&ctrlv1.CreateDeploymentResponse{
		DeploymentId: res.deploymentID,
		Status:       convertDbStatusToProto(res.status),
		Replayed:     res.replayed,
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
		EnvironmentID: dctx.env.Environment.ID,
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
		fmt.Errorf("environment %q is not deployable: %s", dctx.env.Environment.Slug, strings.Join(messages, "; ")))
}

// recordCreateAudit writes a deployment.create audit log attributed to the
// actor supplied on the request, inside the transaction that inserts the
// deployment row so the two commit or fail together. A nil actor (callers
// not yet passing one) falls back to the system actor via actor.AuditType.
func (s *Service) recordCreateAudit(
	ctx context.Context,
	tx db.DBTX,
	dctx deploymentContext,
	deploymentID string,
	a *ctrlv1.ActorInfo,
) error {
	return s.auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
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
						"environment": dctx.env.Environment.Slug,
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
	action  string

	// actor attributed on the deployment.create audit log. Nil falls back to
	// the system actor. Unused outside the create action.
	actor *ctrlv1.ActorInfo

	// Source overrides. dockerImage wins if set; otherwise we auto-detect
	// from git repo connection (using gitCommit.commit_sha if provided) or
	// fall back to the live deployment's image.
	dockerImage string
	gitCommit   *ctrlv1.GitCommitInfo
	command     []string

	// Attribution persisted on the deployment row.
	trigger       db.DeploymentsTrigger
	triggeredBy   string
	triggerReason string

	// Caller-supplied deduplication key, scoped per workspace. When set, the
	// deployment id is derived from it, so a retry re-derives the same id and
	// the unique id rejects the duplicate insert. Empty disables dedup.
	idempotencyKey string
}

// createResult is what createAndDeploy answered with.
type createResult struct {
	deploymentID string

	// status at answer time: pending for a fresh insert or a heal, the row's
	// current status for a replay.
	status mysqltype.DeploymentsStatus

	// replayed: answered with a deployment that already existed instead of
	// inserting one. Only the inserting request writes the audit log.
	replayed bool
}

// createAndDeploy is the shared path used by both CreateDeployment and
// RebuildDeployment. It answers idempotency-key retries, checks workspace
// access and environment deployability, resolves the source, inserts the
// deployment row, kicks off the Restate workflow, and cancels superseded
// siblings.
func (s *Service) createAndDeploy(ctx context.Context, p createParams) (createResult, error) {
	c := p.context

	p.idempotencyKey = strings.TrimSpace(p.idempotencyKey)

	deploymentID := uid.New(uid.DeploymentPrefix)
	if p.idempotencyKey != "" {
		deploymentID = uid.Derived(uid.DeploymentPrefix, c.workspaceID, p.idempotencyKey)
		if res, handled, err := s.dedupKeyedCreate(ctx, p, deploymentID); handled {
			return res, err
		}
	}

	if err := s.ensureWorkspaceCanDeploy(ctx, c.workspaceID, p.action); err != nil {
		return createResult{}, err
	}
	if err := s.ensureEnvironmentDeployable(ctx, c); err != nil {
		return createResult{}, err
	}

	now := time.Now().UnixMilli()

	// Per-request command override (CLI/API) wins over the app's stored
	// default. Persisting only the default would mean the row disagrees with
	// what's actually running, which breaks rebuild and post-mortem flows.
	command := c.appRuntimeSettings.Command
	if len(p.command) > 0 {
		command = p.command
	}

	commit := commitFromRequest(p.gitCommit)
	deployReq, commit, err := s.resolveSource(ctx, c, deploymentID, command, commit, p.dockerImage, p.gitCommit != nil)
	if err != nil {
		return createResult{}, err
	}

	// A docker-source deployment's image is known and final at insert time, so
	// record it: a heal pins the build to it, the same as commit and command.
	// Git builds leave it empty until the workflow writes the built image.
	image := deployReq.GetDockerImage().GetImage()

	trigger := p.trigger
	if trigger == "" {
		trigger = db.DeploymentsTriggerUnknown
	}

	// Truncate operator-supplied reason to the column width so a long
	// note doesn't bubble up as a 500 from MySQL.
	triggerReason := trimLength(p.triggerReason, maxTriggerReasonLength)

	insertParams := db.InsertDeploymentParams{
		ID:                            deploymentID,
		K8sName:                       uid.DNS1035(12),
		WorkspaceID:                   c.workspaceID,
		ProjectID:                     c.project.ID,
		AppID:                         c.app.ID,
		EnvironmentID:                 c.env.Environment.ID,
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
		Image:                         sql.NullString{String: image, Valid: image != ""},
		DeploymentTrigger:             trigger,
		TriggeredBy:                   sql.NullString{String: p.triggeredBy, Valid: p.triggeredBy != ""},
		TriggerReason:                 sql.NullString{String: triggerReason, Valid: triggerReason != ""},
	}

	// The audit row commits with the insert, so a crash or a failed audit
	// write can never leave a deployment without its create audit. Replays
	// and heals answer with a row whose insert already audited; a rebuild
	// records its own deployment.rebuild event instead.
	insertErr := db.TxRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
		if err := db.NewQueries(tx).InsertDeployment(ctx, insertParams); err != nil {
			return err
		}
		if p.action == "create" {
			return s.recordCreateAudit(ctx, tx, c, deploymentID, p.actor)
		}
		return nil
	})
	if insertErr != nil && p.idempotencyKey != "" && db.IsDuplicateKeyError(insertErr) {
		// The id is taken by a row that appeared after the pre-check; resolve
		// it the same way. No row found means the duplicate has another cause,
		// so keep the original error.
		if res, handled, err := s.dedupKeyedCreate(ctx, p, deploymentID); handled {
			return res, err
		}
	}
	if insertErr != nil {
		logger.Error("failed to insert deployment", "error", insertErr.Error())
		return createResult{}, connect.NewError(connect.CodeInternal, insertErr)
	}

	logger.Info(
		"starting deployment workflow",
		"deployment_id", deploymentID,
		"workspace_id", c.workspaceID,
		"project_id", c.project.ID,
		"app_id", c.app.ID,
		"environment", c.env.Environment.ID,
		"trigger", string(trigger),
	)

	if err := s.sendWorkflow(ctx, deploymentID, deployReq); err != nil {
		// A keyed row is left pending: the caller retries the same key, the
		// heal re-sends, and the invocation-idempotent send attaches if
		// Restate accepted this send despite the error. Marking it failed
		// would spend the key and turn an ambiguous send into a duplicate
		// deployment under a new key. An unkeyed row has no retry path, so
		// it is marked failed rather than left pending forever.
		if p.idempotencyKey == "" {
			if updateErr := s.db.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
				ID:        deploymentID,
				Status:    mysqltype.DeploymentsStatusFailed,
				UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			}); updateErr != nil {
				logger.Error("failed to mark deployment as failed", "deployment_id", deploymentID, "error", updateErr)
			}
		}
		return createResult{}, err
	}

	s.cancelOlderSiblings(ctx, c, deploymentID, commit.Branch, now)

	return createResult{deploymentID: deploymentID, status: mysqltype.DeploymentsStatusPending, replayed: false}, nil
}

// cancelOlderSiblings supersedes still-queued older deployments on the same
// branch. Failure is logged, not returned: superseding is best-effort and must
// not fail a create that already inserted and sent.
func (s *Service) cancelOlderSiblings(ctx context.Context, c deploymentContext, deploymentID, branch string, createdAt int64) {
	if err := s.dedup.CancelOlderSiblings(ctx, dedup.Newer{
		ID:            deploymentID,
		AppID:         c.app.ID,
		EnvironmentID: c.env.Environment.ID,
		GitBranch:     branch,
		CreatedAt:     createdAt,
	}); err != nil {
		logger.Error(
			"failed to cancel superseded siblings",
			"deployment_id", deploymentID,
			"error", err,
		)
	}
}

// dedupKeyedCreate answers a keyed create with the row its derived id already
// names: a replay, a heal, or a spent-key/scope error. handled is false when
// no such row exists in this workspace, so the caller proceeds with its own
// path. The workspace check keeps a retry from ever answering with another
// tenant's deployment.
func (s *Service) dedupKeyedCreate(ctx context.Context, p createParams, deploymentID string) (createResult, bool, error) {
	existing, findErr := s.db.FindDeploymentById(ctx, deploymentID)
	if findErr != nil {
		if db.IsNotFound(findErr) {
			return createResult{}, false, nil //nolint:exhaustruct // zero value unused when handled is false
		}
		// A broken read must fail the request: proceeding would run gates,
		// GitHub resolution, and an insert against a database already erroring.
		return createResult{}, true, connect.NewError(connect.CodeInternal,
			fmt.Errorf("failed to look up deployment for idempotency key: %w", findErr))
	}
	if existing.WorkspaceID != p.context.workspaceID {
		return createResult{}, false, nil //nolint:exhaustruct // zero value unused when handled is false
	}

	action, resolveErr := resolveKeyedRetry(p.context, existing)
	if resolveErr != nil {
		return createResult{}, true, resolveErr
	}
	if action == actionHeal {
		res, healErr := s.healDeployment(ctx, p, existing)
		return res, true, healErr
	}

	// The only trace a dedup absorbed a duplicate: a retry burst that this
	// feature swallows is invisible without it.
	logger.Info("replayed keyed deployment create",
		"deployment_id", deploymentID,
		"workspace_id", p.context.workspaceID,
		"status", string(existing.Status),
	)
	return createResult{deploymentID: deploymentID, status: existing.Status, replayed: true}, true, nil
}

// keyedRetryAction is what to do with the row a keyed retry's key already
// created. Meaningful only when resolveKeyedRetry returns a nil error.
type keyedRetryAction int

const (
	actionReplay keyedRetryAction = iota
	actionHeal
)

// resolveKeyedRetry decides how a keyed retry answers for the row its key
// already created: replay it, heal it, or reject the key with an error.
func resolveKeyedRetry(c deploymentContext, d db.Deployment) (keyedRetryAction, error) {
	// A key is bound to one app and environment. Answering with a row from a
	// different target would silently ignore what the caller asked to deploy.
	if d.AppID != c.app.ID || d.EnvironmentID != c.env.Environment.ID {
		return actionReplay, idempotencyKeyScopeError(d.ID)
	}

	// Stuck row: ctrl died between the insert and the Restate send, so no
	// workflow will ever run it.
	if d.Status == mysqltype.DeploymentsStatusPending && !d.InvocationID.Valid {
		return actionHeal, nil
	}

	// Dead with no recorded workflow: nothing this key returns will ever
	// progress, so the key is spent. Replaying would leave the caller polling
	// a deployment that never moves.
	if !d.InvocationID.Valid && deadStatus(d.Status) {
		return actionReplay, spentIdempotencyKeyError(d.ID)
	}

	// Everything else replays as-is, even a failure after the workflow ran:
	// a retry gets the original outcome, not a second attempt.
	return actionReplay, nil
}

// healDeployment revives a stuck keyed row (pending, no invocation id): its
// create died between insert and send, so without this the deployment would
// never build. It skips the insert, sends the workflow, and writes the
// invocation id onto the row. Reports a replay.
//
// The build is pinned to what the row records (commit, command, and image),
// not the retry body: the row defines what this deployment id means, and the
// branch or body may have changed since. A row recording neither an image nor
// a commit cannot occur (every insert records one or the other), so its
// fallback to the retry body is defensive only.
//
// A send failure never marks the row failed: the workflow may already be
// running (the send attaches, not duplicates), and the next retry heals again.
func (s *Service) healDeployment(ctx context.Context, p createParams, row db.Deployment) (createResult, error) {
	c := p.context

	if err := s.ensureWorkspaceCanDeploy(ctx, c.workspaceID, p.action); err != nil {
		return createResult{}, err
	}
	if err := s.ensureEnvironmentDeployable(ctx, c); err != nil {
		return createResult{}, err
	}

	// A branch without a SHA (a docker create with git attribution) still
	// pins: sibling dedup must run with the row's branch, not the body's.
	commit := commitFromRequest(p.gitCommit)
	if row.GitCommitSha.Valid || row.GitBranch.Valid {
		commit = commitFieldsFromDeployment(row)
	}

	// The row defines what its id builds, so source intent comes from the row,
	// not the retry body. A recorded image pins a docker build. A recorded
	// commit without an image is a git build whose image the workflow never
	// wrote: drop the body's image so it cannot repoint the id at a different
	// artifact, and demand git so a deleted repo connection refuses instead of
	// falling back to the current deployment's image. The final else is
	// unreachable: every insert records an image or a commit, and a row that
	// predates image-at-insert carries a random id no derived id can name. It
	// keeps the body as a last resort rather than adding an error path for a
	// state only a future bug could produce.
	dockerImage := p.dockerImage
	explicitGit := p.gitCommit != nil
	gitPinned := false
	if row.Image.Valid {
		dockerImage = row.Image.String
	} else if row.GitCommitSha.Valid {
		dockerImage = ""
		explicitGit = true
		gitPinned = true
	}

	deployReq, commit, err := s.resolveSource(ctx, c, row.ID, row.Command, commit, dockerImage, explicitGit)
	if err != nil {
		// A git-pinned row whose repo connection is gone can never build: no
		// retry heals it, so leaving it pending loops the caller on the same
		// key forever. Fail the row so the next retry reads the key as spent.
		// Only this permanent refusal spends; transient resolution errors
		// leave the row pending for the next heal. A body-sourced git request
		// (gitPinned false) stays pending too: a retry with a docker body can
		// still heal it.
		if gitPinned && errors.Is(err, errNoRepoConnection) {
			// The row is failed with the queued step it never left, so the
			// dashboard shows the reason instead of an empty timeline. Both
			// writes share a transaction: a failed row without its step is
			// exactly the blank timeline this exists to prevent.
			spentAt := time.Now().UnixMilli()
			if spendErr := db.TxRetry(ctx, s.db.RW(), func(ctx context.Context, tx db.DBTX) error {
				q := db.NewQueries(tx)
				if stepErr := q.InsertDeploymentStep(ctx, db.InsertDeploymentStepParams{
					WorkspaceID:   row.WorkspaceID,
					ProjectID:     row.ProjectID,
					AppID:         row.AppID,
					EnvironmentID: row.EnvironmentID,
					DeploymentID:  row.ID,
					Step:          db.DeploymentStepsStepQueued,
					StartedAt:     uint64(row.CreatedAt),
				}); stepErr != nil {
					return stepErr
				}
				if endErr := q.EndDeploymentStep(ctx, db.EndDeploymentStepParams{
					DeploymentID: row.ID,
					Step:         db.DeploymentStepsStepQueued,
					EndedAt:      sql.NullInt64{Valid: true, Int64: spentAt},
					Error:        sql.NullString{Valid: true, String: unbuildableRowMessage},
				}); endErr != nil {
					return endErr
				}
				return q.UpdateDeploymentStatus(ctx, db.UpdateDeploymentStatusParams{
					ID:        row.ID,
					Status:    mysqltype.DeploymentsStatusFailed,
					UpdatedAt: sql.NullInt64{Valid: true, Int64: spentAt},
				})
			}); spendErr != nil {
				logger.Error("failed to spend unbuildable keyed deployment", "deployment_id", row.ID, "error", spendErr)
			}
		}
		return createResult{}, err
	}

	logger.Info(
		"healing stuck deployment",
		"deployment_id", row.ID,
		"workspace_id", c.workspaceID,
		"app_id", c.app.ID,
	)

	if err := s.sendWorkflow(ctx, row.ID, deployReq); err != nil {
		return createResult{}, err
	}

	// Dedup with the row's own age: the heal acts for the original create, so
	// it may only supersede siblings older than that create. The retry's time
	// would let an old stuck row cancel deployments newer than itself.
	s.cancelOlderSiblings(ctx, c, row.ID, commit.Branch, row.CreatedAt)

	return createResult{deploymentID: row.ID, status: mysqltype.DeploymentsStatusPending, replayed: true}, nil
}

// errNoRepoConnection marks the refusal to build a git request for an app
// without a repo connection. healDeployment matches it to tell this permanent
// failure apart from transient resolution errors, which stay healable.
var errNoRepoConnection = errors.New("no GitHub repo connection; cannot deploy requested git commit")

// unbuildableRowMessage is recorded on the queued step of a keyed row spent
// because the commit it pins can never build. It reaches the dashboard
// timeline, so it says what a user can act on.
const unbuildableRowMessage = "This app has no GitHub repo connection, so the commit this deployment was created for can never build."

// resolveSource picks the deployment's build source and completes its commit
// metadata. An explicit docker image wins; an explicit git request without a
// repo connection is refused; a git-connected app builds from git, filling
// missing commit metadata from GitHub so the row is complete at insert time;
// everything else reuses the current deployment's image. The returned
// commitFields replace the caller's, because the fallback arm inherits them
// from the current deployment.
func (s *Service) resolveSource(
	ctx context.Context,
	c deploymentContext,
	deploymentID string,
	command []string,
	commit commitFields,
	dockerImage string,
	explicitGit bool,
) (*hydrav1.DeployRequest, commitFields, error) {
	// Look up the GitHub repo connection once. Used both to decide source type
	// (git vs docker) and to resolve missing commit metadata synchronously.
	repoConn, repoErr := s.db.FindGithubRepoConnectionByAppId(ctx, c.app.ID)
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
			"app_id", c.app.ID,
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
		// image (a different artifact than what was requested).
		return nil, commitFields{}, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %q has %w", c.app.ID, errNoRepoConnection))

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
			// This error may carry the raw GitHub response body, which can reach
			// API callers. Log the detail, return a generic reason.
			logger.Error("failed to resolve git commit metadata",
				"app_id", c.app.ID,
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
					ContextPath:    c.appBuildSettings.DockerContext,
					DockerfilePath: c.appBuildSettings.Dockerfile.String,
					BuildCommand:   c.appBuildSettings.BuildCommand.String,
					Branch:         commit.Branch,
					ForkRepository: commit.ForkRepository,
					PrNumber:       0,
				},
			},
		}, commit, nil

	default:
		// No docker image, no git commit, no repo connection: reuse current
		// deployment's image.
		dockerInfo, dockerErr := buildDockerSource(ctx, s.db, c.app, deploymentID)
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

// sendWorkflow starts the deployment's Restate workflow and writes the
// resulting invocation id onto the row. The send is keyed by the deployment id at the invocation
// layer, so a heal re-send attaches to an already-running workflow instead of
// building twice; the virtual object key alone only serializes. Attachment
// only holds inside Restate's idempotency retention window (one day by
// default): a re-send after that starts a second invocation for the same
// deployment id. What to do
// with the row on failure is the caller's call: an unkeyed create marks it
// failed, keyed creates and heals leave it pending for the next retry.
func (s *Service) sendWorkflow(ctx context.Context, deploymentID string, deployReq *hydrav1.DeployRequest) error {
	invocation, err := s.deploymentClient(deploymentID).
		Deploy().
		Send(ctx, deployReq, restate.WithIdempotencyKey(deploymentID))
	if err != nil {
		logger.Error("failed to start deployment workflow", "error", err)
		return connect.NewError(connect.CodeInternal, fmt.Errorf("unable to start workflow: %w", err))
	}

	invocationID := invocation.Id()
	if updateErr := s.db.UpdateDeploymentInvocationID(ctx, db.UpdateDeploymentInvocationIDParams{
		ID:           deploymentID,
		InvocationID: sql.NullString{Valid: true, String: invocationID},
		UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
	}); updateErr != nil {
		// A lost write here looks like a create that died before the Send;
		// a keyed retry heals it and records the id.
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
	return nil
}

// deadStatus reports a deployment that ended without succeeding. A row in such
// a state with no invocation id never ran a workflow and never will. Ready is
// the one terminal status that cannot be dead: reaching it proves the workflow
// ran, so a ready row with a lost invocation id replays instead.
func deadStatus(s mysqltype.DeploymentsStatus) bool {
	return s.IsTerminal() && s != mysqltype.DeploymentsStatusReady
}

// spentIdempotencyKeyError reports a key bound to a deployment that already
// ended and cannot rerun. Nothing can restart it; the caller has to send a
// new key.
func spentIdempotencyKeyError(deploymentID string) error {
	err := connect.NewError(connect.CodeAlreadyExists,
		fmt.Errorf("idempotency key is bound to deployment %s, which already ended and cannot rerun", deploymentID))
	err.Meta().Set(idempotency.MetaKey, idempotency.ReasonKeySpent)
	return err
}

// idempotencyKeyScopeError reports a key already bound to a deployment in a
// different app or environment of the same workspace.
func idempotencyKeyScopeError(deploymentID string) error {
	err := connect.NewError(connect.CodeAlreadyExists,
		fmt.Errorf("idempotency key is already bound to deployment %s in a different app or environment", deploymentID))
	err.Meta().Set(idempotency.MetaKey, idempotency.ReasonScopeMismatch)
	return err
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
		dockerImage:  currentDeployment.Image.String,
		commitFields: commitFieldsFromDeployment(currentDeployment),
	}, nil
}

// commitFromRequest maps caller-provided commit metadata onto commitFields,
// trimming to column bounds. Branch defaulting and GitHub fill-in happen in
// resolveSource, only when actually building from git — docker-image redeploys
// must not synthesize git metadata.
func commitFromRequest(gc *ctrlv1.GitCommitInfo) commitFields {
	if gc == nil {
		return commitFields{} //nolint:exhaustruct // empty fields mean "unknown" by contract
	}
	return commitFields{
		SHA:             gc.GetCommitSha(),
		Branch:          strings.TrimSpace(gc.GetBranch()),
		Message:         trimLength(gc.GetCommitMessage(), maxCommitMessageLength),
		AuthorHandle:    trimLength(strings.TrimSpace(gc.GetAuthorHandle()), maxCommitAuthorHandleLength),
		AuthorAvatarURL: trimLength(strings.TrimSpace(gc.GetAuthorAvatarUrl()), maxCommitAuthorAvatarLength),
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
