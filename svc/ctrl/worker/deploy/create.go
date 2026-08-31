package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
	"unicode/utf8"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploytarget"
)

// Column widths for the commit metadata a row records. MySQL strict mode
// rejects an overlong value, so an unbounded commit message would surface as a
// 500 instead of a deployment.
const (
	commitMessageBytesMax         = 10240
	commitAuthorHandleBytesMax    = 256
	commitAuthorAvatarURLBytesMax = 512
	triggerReasonBytesMax         = 512
)

// createBlock is a caller precondition no retry can fix: no plan, no repository
// connection, an unresolvable commit.
//
// It travels back as a successful response rather than a terminal error because
// the Restate ingress collapses a handler failure into unstructured text. svc/api
// has to answer 412 with its own message, and it cannot parse a reason out of
// prose. The webhook benefits too: a policy block stops producing failed
// invocations on a repository that will never be eligible.
type createBlock struct {
	Reason hydrav1.CreateBlockedReason `json:"reason"`
	Detail string                      `json:"detail"`
}

// newBlockf builds a [createBlock] with a formatted detail. The f suffix follows
// fmt.Errorf: it takes a format string.
func newBlockf(reason hydrav1.CreateBlockedReason, format string, args ...any) *createBlock {
	return &createBlock{Reason: reason, Detail: fmt.Sprintf(format, args...)}
}

// Create is the only writer of deployment rows. See the proto for the contract.
//
// Every step is journaled, so a crash resumes instead of leaving a half-written
// deployment:
//
//  1. row-exists backstop, which keeps a repeat idempotent past the idempotency
//     key's retention window
//  2. load the target, run the gates, resolve the build source
//  3. insert the row, its queued step, and its audit log in one transaction
//  4. pending only: record the invocation id, send Deploy, supersede siblings
func (w *Workflow) Create(ctx restate.ObjectContext, req *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	// Always the object key, never the payload. The row this writes and the lock
	// it holds have to be the same id for anything below to hold.
	deploymentID := restate.Key(ctx)

	if err := assert.All(
		assert.NotEmpty(req.GetProjectId(), "project_id is required"),
		assert.NotEmpty(req.GetAppId(), "app_id is required"),
		assert.NotEmpty(req.GetEnvironment(), "environment is required"),
	); err != nil {
		return nil, restate.TerminalError(err)
	}

	status, err := statusForDecision(req.GetDecision())
	if err != nil {
		return nil, err
	}

	existing, err := restate.Run(ctx, func(runCtx restate.RunContext) (backstop, error) {
		return w.findExistingDeployment(runCtx, deploymentID)
	}, restate.WithName("check for existing deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, err
	}
	if existing.Found {
		// The idempotency key stops replaying once its retention window passes.
		// The row is what keeps a repeat idempotent after that.
		if existing.ProjectID != req.GetProjectId() || existing.AppID != req.GetAppId() {
			// A derived id hashes the workspace, app, and environment, so two
			// targets cannot produce one id. If they somehow did, answering
			// would hand this caller someone else's deployment.
			return nil, restate.TerminalError(fmt.Errorf(
				"deployment %s belongs to project %s app %s, not project %s app %s",
				deploymentID, existing.ProjectID, existing.AppID, req.GetProjectId(), req.GetAppId(),
			))
		}

		logger.Info("deployment create replayed an existing row",
			"deployment_id", deploymentID,
			"status", string(existing.Status),
		)
		return createResponse(hydrav1.CreateOutcome_CREATE_OUTCOME_REPLAYED, nil), nil
	}

	// A derived id can only come back if its row is gone, and the only thing
	// that deletes deployment rows is an environment cascade, which takes the
	// environment id feeding the derived hash with it. So this object should
	// have no history. Clearing the awakeable costs nothing and keeps a
	// half-torn-down object from parking a fresh deployment on a stale wait.
	restate.Clear(ctx, instancesReadyAwakeableKey)

	resolution, err := restate.Run(ctx, func(runCtx restate.RunContext) (createResolution, error) {
		return w.resolveCreate(runCtx, req)
	}, restate.WithName("resolve create"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, err
	}
	if resolution.Block != nil {
		logger.Info("deployment create blocked",
			"deployment_id", deploymentID,
			"app_id", req.GetAppId(),
			"reason", resolution.Block.Reason.String(),
			"detail", resolution.Block.Detail,
		)
		return createResponse(hydrav1.CreateOutcome_CREATE_OUTCOME_BLOCKED, resolution.Block), nil
	}

	createdAt, err := restate.Run(ctx, func(runCtx restate.RunContext) (int64, error) {
		return w.insertDeployment(runCtx, req, deploymentID, resolution, status)
	}, restate.WithName("insert deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, err
	}

	logger.Info("deployment created",
		"deployment_id", deploymentID,
		"app_id", req.GetAppId(),
		"environment", req.GetEnvironment(),
		"status", string(status),
	)

	// The row exists either way; the decision says what happens to it now.
	switch req.GetDecision() {
	case hydrav1.CreateDecision_CREATE_DECISION_DEPLOY:
		if err := w.startDeploy(ctx, deploymentID, req, resolution, createdAt); err != nil {
			return nil, err
		}
	case hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL:
		w.requestAuthorization(ctx, deploymentID, req, resolution)
	case hydrav1.CreateDecision_CREATE_DECISION_SKIP:
		// Nothing to do: the row is the whole point.
	case hydrav1.CreateDecision_CREATE_DECISION_UNSPECIFIED:
		// Rejected before anything was written. See statusForDecision.
	}

	return createResponse(hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED, nil), nil
}

// createResponse builds the handler's answer: what happened, and why not if it
// did not. A nil block leaves the reason unset, so a BLOCKED outcome cannot ship
// without one.
func createResponse(outcome hydrav1.CreateOutcome, block *createBlock) *hydrav1.DeployCreateResponse {
	resp := &hydrav1.DeployCreateResponse{
		Outcome:       outcome,
		BlockedReason: hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_UNSPECIFIED,
	}
	if block != nil {
		resp.BlockedReason = block.Reason
	}
	return resp
}

// backstop is the journaled answer to whether the row already exists.
type backstop struct {
	Found     bool                        `json:"found"`
	Status    mysqltype.DeploymentsStatus `json:"status"`
	ProjectID string                      `json:"project_id"`
	AppID     string                      `json:"app_id"`
}

func (w *Workflow) findExistingDeployment(ctx context.Context, deploymentID string) (backstop, error) {
	row, err := w.db.FindDeploymentById(ctx, deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return backstop{Found: false, Status: "", ProjectID: "", AppID: ""}, nil
		}
		return backstop{}, err //nolint:exhaustruct // zero value unused on error
	}
	return backstop{Found: true, Status: row.Status, ProjectID: row.ProjectID, AppID: row.AppID}, nil
}

// createResolution is what [Workflow.resolveCreate] decided, flattened to
// scalars: the journal round-trips through JSON, which a proto oneof does not
// survive.
type createResolution struct {
	// Block set means nothing was written and nothing should be.
	Block *createBlock `json:"block"`

	WorkspaceSlug string `json:"workspace_slug"`
	EnvironmentID string `json:"environment_id"`

	// Command is resolved once here so the row and the Deploy request cannot
	// disagree about what the container runs.
	Command []string `json:"command"`

	// PRNumber goes on the row for every decision, including a skip, which has
	// no source to carry it.
	PRNumber int64 `json:"pr_number"`

	Source buildSource  `json:"source"`
	Commit commitFields `json:"commit"`
}

// resolveCreate loads the target, runs the gates, and resolves the build source.
// One journaled step because all three share the target, and the target itself
// does not survive the journal's JSON round trip.
func (w *Workflow) resolveCreate(ctx context.Context, req *hydrav1.DeployCreateRequest) (createResolution, error) {
	target, loadErr := deploytarget.Load(ctx, w.db, req.GetProjectId(), req.GetAppId(), req.GetEnvironment(), deploytarget.WithoutSecrets)
	var terminal *deploytarget.TerminalError
	switch {
	case errors.As(loadErr, &terminal):
		// The caller checked this target first, so a miss here means it was
		// deleted mid-create. A block, not an error: no retry brings it back.
		return createResolution{Block: newBlockf( //nolint:exhaustruct // only the block is read
			hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_TARGET_NOT_FOUND,
			"%s", terminal.Message,
		)}, nil
	case loadErr != nil:
		return createResolution{}, loadErr //nolint:exhaustruct // zero value unused on error
	}

	block, err := w.checkDeployGate(ctx, target.WorkspaceID)
	if err != nil {
		return createResolution{}, err //nolint:exhaustruct // zero value unused on error
	}
	if block != nil {
		return createResolution{Block: block}, nil //nolint:exhaustruct // only the block is read
	}

	commit := commitFromProto(req.GetGit().GetCommit())
	prNumber := req.GetGit().GetPrNumber()
	source := buildSource{Image: "", Git: nil}

	// A skipped row never builds, so it has no source to resolve. Resolving one
	// anyway would make recording the skip depend on the app's repository
	// connection and on GitHub answering, and the whole point of the row is that
	// it survives to say this commit was seen and deliberately not built.
	if req.GetDecision() != hydrav1.CreateDecision_CREATE_DECISION_SKIP {
		resolved, resolveErr := w.resolveSource(ctx, target, req, commit)
		if resolveErr != nil {
			return createResolution{}, resolveErr //nolint:exhaustruct // zero value unused on error
		}
		if resolved.Block != nil {
			return createResolution{Block: resolved.Block}, nil //nolint:exhaustruct // only the block is read
		}
		if resolved.Source.Git == nil && resolved.Source.Image == "" {
			return createResolution{}, restate.TerminalError(errors.New("no build source: set git, image, or existing_deployment")) //nolint:exhaustruct // zero value unused on error
		}
		source, commit = resolved.Source, resolved.Commit
		if source.Git != nil {
			// A rebuild takes the PR number off the deployment it reproduces,
			// not off the request.
			prNumber = source.Git.PRNumber
		}
	}

	// A per-request override (e.g. `unkey deploy --command`) wins over the app's
	// stored default, so the row records what actually runs.
	command := target.Command
	if len(req.GetCommand()) > 0 {
		command = req.GetCommand()
	}

	return createResolution{
		Block:         nil,
		WorkspaceSlug: target.WorkspaceSlug,
		EnvironmentID: target.EnvironmentID,
		Command:       command,
		PRNumber:      prNumber,
		Source:        source,
		Commit:        commit,
	}, nil
}

// checkDeployGate is the billing gate every create passes. A workspace with no
// billing row reads as having no plan rather than as an error: that is the
// normal state before anyone subscribes.
//
// Plan enforcement honours the rollout flag; the spend cap always blocks.
func (w *Workflow) checkDeployGate(ctx context.Context, workspaceID string) (*createBlock, error) {
	entitlement, err := w.db.FindWorkspaceDeployEntitlement(ctx, workspaceID)
	if err != nil && !db.IsNotFound(err) {
		return nil, err
	}

	if !deploygate.Entitled(entitlement.Plan, entitlement.PlanOverride) {
		if w.enforceDeployGate {
			return newBlockf(
				hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_NO_COMPUTE_PLAN,
				"workspace %s has no Compute plan", workspaceID,
			), nil
		}
		logger.Warn("deploy gate would block deployment create",
			"event", "deploy_gate.would_block",
			"workspaceId", workspaceID,
		)
	}

	if entitlement.SpendSuspended.Bool {
		return newBlockf(
			hydrav1.CreateBlockedReason_CREATE_BLOCKED_REASON_SPEND_SUSPENDED,
			"workspace %s is suspended by its Compute spend cap", workspaceID,
		), nil
	}

	return nil, nil
}

// insertDeployment writes the row, its queued step, and its audit log in one
// transaction, and returns the created_at it settled on.
//
// The target is loaded again, with secrets this time, so the row records the
// settings current when the create runs rather than when it was asked for.
func (w *Workflow) insertDeployment(
	ctx context.Context,
	req *hydrav1.DeployCreateRequest,
	deploymentID string,
	resolution createResolution,
	status mysqltype.DeploymentsStatus,
) (int64, error) {
	// A skipped row never builds, so it holds no secrets at rest.
	secrets := deploytarget.WithSecrets
	if status == mysqltype.DeploymentsStatusSkipped {
		secrets = deploytarget.WithoutSecrets
	}

	target, loadErr := deploytarget.Load(ctx, w.db, req.GetProjectId(), req.GetAppId(), req.GetEnvironment(), secrets)
	var terminal *deploytarget.TerminalError
	if errors.As(loadErr, &terminal) {
		return 0, restate.TerminalError(loadErr)
	}
	if loadErr != nil {
		return 0, loadErr
	}

	// The caller's ordering timestamp wins so sibling dedup keeps push order even
	// when async sends land out of order. Reading the clock here rather than in
	// the handler keeps it inside the journaled step.
	createdAt := req.GetOrderingTimestamp()
	if createdAt == 0 {
		createdAt = time.Now().UnixMilli()
	}

	commit := resolution.Commit
	commitMessage := trimBytes(commit.Message, commitMessageBytesMax)
	authorHandle := trimBytes(commit.AuthorHandle, commitAuthorHandleBytesMax)
	authorAvatarURL := trimBytes(commit.AuthorAvatarURL, commitAuthorAvatarURLBytesMax)
	triggerReason := trimBytes(req.GetTriggerReason(), triggerReasonBytesMax)
	triggeredBy := req.GetTriggeredBy()

	insertErr := db.TxRetry(ctx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if err := db.NewQueries(tx).InsertDeployment(txCtx, db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   target.WorkspaceID,
			ProjectID:                     target.ProjectID,
			AppID:                         target.AppID,
			EnvironmentID:                 target.EnvironmentID,
			SentinelConfig:                target.SentinelConfig,
			EncryptedEnvironmentVariables: target.SecretsBlob,
			Command:                       resolution.Command,
			Status:                        status,
			CreatedAt:                     createdAt,
			UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
			GitCommitSha:                  sql.NullString{String: commit.SHA, Valid: commit.SHA != ""},
			GitBranch:                     sql.NullString{String: commit.Branch, Valid: commit.Branch != ""},
			GitCommitMessage:              sql.NullString{String: commitMessage, Valid: commitMessage != ""},
			GitCommitAuthorHandle:         sql.NullString{String: authorHandle, Valid: authorHandle != ""},
			GitCommitAuthorAvatarUrl:      sql.NullString{String: authorAvatarURL, Valid: authorAvatarURL != ""},
			GitCommitTimestamp:            sql.NullInt64{Int64: commit.Timestamp, Valid: commit.Timestamp != 0},
			CpuMillicores:                 target.CpuMillicores,
			MemoryMib:                     target.MemoryMib,
			StorageMib:                    target.StorageMib,
			Port:                          target.Port,
			ShutdownSignal:                db.DeploymentsShutdownSignal(target.ShutdownSignal),
			UpstreamProtocol:              db.DeploymentsUpstreamProtocol(target.UpstreamProtocol),
			Healthcheck:                   target.Healthcheck,
			PrNumber:                      sql.NullInt64{Int64: resolution.PRNumber, Valid: resolution.PRNumber != 0},
			ForkRepositoryFullName:        sql.NullString{String: commit.ForkRepository, Valid: commit.ForkRepository != ""},
			DeploymentTrigger:             triggerFromProto(req.GetTrigger()),
			TriggeredBy:                   sql.NullString{String: triggeredBy, Valid: triggeredBy != ""},
			TriggerReason:                 sql.NullString{String: triggerReason, Valid: triggerReason != ""},
		}); err != nil {
			return err
		}

		// Deploy ends this step and never inserts it, so a row without it shows
		// no queue time and starts visibly at Starting.
		if err := db.NewQueries(tx).InsertDeploymentStep(txCtx, db.InsertDeploymentStepParams{
			WorkspaceID:   target.WorkspaceID,
			ProjectID:     target.ProjectID,
			AppID:         target.AppID,
			EnvironmentID: target.EnvironmentID,
			DeploymentID:  deploymentID,
			Step:          db.DeploymentStepsStepQueued,
			StartedAt:     uint64(createdAt),
		}); err != nil {
			return err
		}

		return w.auditlogs.Insert(txCtx, tx, w.createAuditLogs(req, target, deploymentID, status))
	})
	if insertErr != nil {
		if !db.IsDuplicateKeyError(insertErr) {
			return 0, insertErr
		}
		// An earlier attempt committed before the journal recorded it. Keep its
		// created_at: sibling dedup compares against it, and this attempt's
		// later clock would make the row look newer than it is.
		row, findErr := w.db.FindDeploymentById(ctx, deploymentID)
		if findErr != nil {
			return 0, fmt.Errorf("duplicate key on insert but deployment %s not found: %w", deploymentID, findErr)
		}
		return row.CreatedAt, nil
	}

	return createdAt, nil
}

// createAuditLogs builds the audit entry for a new row. A skipped row gets none:
// nothing happened that a customer needs to see.
//
// An operator rebuild is recorded as deployment.rebuild with both deployments as
// resources. The trigger marks it: UNKEY means Unkey itself asked for the
// deployment, which is only ever a rebuild.
func (w *Workflow) createAuditLogs(
	req *hydrav1.DeployCreateRequest,
	target deploytarget.Target,
	deploymentID string,
	status mysqltype.DeploymentsStatus,
) []auditlog.AuditLog {
	if status == mysqltype.DeploymentsStatusSkipped {
		return nil
	}

	a := req.GetActor()
	entry := auditlog.AuditLog{
		Event:         auditlog.DeploymentCreateEvent,
		WorkspaceID:   target.WorkspaceID,
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
					"projectId":   target.ProjectID,
					"appId":       target.AppID,
					"environment": target.EnvironmentSlug,
				},
			},
		},
	}

	sourceID := req.GetExistingDeployment().GetDeploymentId()
	if req.GetTrigger() != ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_UNKEY || sourceID == "" {
		return []auditlog.AuditLog{entry}
	}

	entry.Event = auditlog.DeploymentRebuildEvent
	entry.Display = fmt.Sprintf("Unkey rebuilt deployment %s as %s", sourceID, deploymentID)
	if reason := req.GetTriggerReason(); reason != "" {
		entry.Display = fmt.Sprintf("%s (reason: %s)", entry.Display, reason)
	}
	entry.Resources[0].Meta["role"] = "new"
	entry.Resources = append([]auditlog.AuditLogResource{
		{
			Type:        auditlog.DeploymentResourceType,
			ID:          sourceID,
			Name:        "",
			DisplayName: sourceID,
			Meta:        map[string]any{"role": "source"},
		},
	}, entry.Resources...)

	return []auditlog.AuditLog{entry}
}

// startDeploy records the invocation id, sends Deploy to this same object, and
// supersedes older queued siblings on the branch.
//
// Send, not Request: Deploy is another exclusive handler on this key and could
// not start until this one returned. The invocation id is written straight after
// so a cancel arriving next instant has something to cancel; Deploy re-persists
// it because that write still races the send.
func (w *Workflow) startDeploy(
	ctx restate.ObjectContext,
	deploymentID string,
	req *hydrav1.DeployCreateRequest,
	resolution createResolution,
	createdAt int64,
) error {
	invocation := hydrav1.NewDeployServiceClient(ctx, deploymentID).
		Deploy().
		Send(resolution.Source.deployRequest(deploymentID, resolution.Command, resolution.Commit))

	invocationID := invocation.GetInvocationId()
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.db.UpdateDeploymentInvocationID(runCtx, db.UpdateDeploymentInvocationIDParams{
			ID:           deploymentID,
			InvocationID: sql.NullString{Valid: true, String: invocationID},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("persist invocation id"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return err
	}

	// Best effort: the closure swallows its own error so a failed cancel never
	// fails a deployment that is already queued. The RunVoid error still
	// propagates, since it can carry Restate protocol signals.
	if runErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		if cancelErr := w.dedup.CancelOlderSiblings(runCtx, dedup.Newer{
			ID:            deploymentID,
			AppID:         req.GetAppId(),
			EnvironmentID: resolution.EnvironmentID,
			GitBranch:     resolution.Commit.Branch,
			CreatedAt:     createdAt,
		}); cancelErr != nil {
			logger.Error("failed to cancel superseded siblings",
				"deployment_id", deploymentID,
				"error", cancelErr,
			)
		}
		return nil
	}, restate.WithName("cancel superseded siblings"), restate.WithMaxRetryAttempts(runMaxAttempts)); runErr != nil {
		return runErr
	}

	logger.Info("deployment workflow started",
		"deployment_id", deploymentID,
		"app_id", req.GetAppId(),
		"invocation_id", invocationID,
	)
	return nil
}

// requestAuthorization posts the failing commit status that tells a contributor
// their push is waiting for a project member. Fire and forget: the row already
// records that, and GitHub being down must not fail the create.
func (w *Workflow) requestAuthorization(
	ctx restate.ObjectContext,
	deploymentID string,
	req *hydrav1.DeployCreateRequest,
	resolution createResolution,
) {
	if resolution.Source.Git == nil {
		return
	}

	logURL := fmt.Sprintf("%s/%s/projects/%s/deployments/%s",
		w.dashboardURL, resolution.WorkspaceSlug, req.GetProjectId(), deploymentID,
	)

	hydrav1.NewGitHubStatusServiceClient(ctx, deploymentID).
		SetCommitStatus().
		Send(&hydrav1.GitHubStatusCommitStatusRequest{
			InstallationId: resolution.Source.Git.InstallationID,
			Repo:           resolution.Source.Git.Repository,
			CommitSha:      resolution.Commit.SHA,
			State:          hydrav1.GitHubCommitStatusState_GITHUB_COMMIT_STATUS_STATE_FAILURE,
			TargetUrl:      logURL,
			Description:    "Awaiting authorization from a project member",
		})

	logger.Info("deployment blocked for authorization",
		"deployment_id", deploymentID,
		"project_id", req.GetProjectId(),
	)
}

// statusForDecision maps the caller's decision onto the status the row starts
// in. The two vocabularies are separate on purpose: a decision is what the
// caller wants, a status is what the deployment is.
//
// An unset decision is a caller bug, not a default. Reading it as DEPLOY would
// build a commit whose watch paths said to skip it.
func statusForDecision(decision hydrav1.CreateDecision) (mysqltype.DeploymentsStatus, error) {
	switch decision {
	case hydrav1.CreateDecision_CREATE_DECISION_DEPLOY:
		return mysqltype.DeploymentsStatusPending, nil
	case hydrav1.CreateDecision_CREATE_DECISION_SKIP:
		return mysqltype.DeploymentsStatusSkipped, nil
	case hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL:
		return mysqltype.DeploymentsStatusAwaitingApproval, nil
	case hydrav1.CreateDecision_CREATE_DECISION_UNSPECIFIED:
		return "", restate.TerminalError(errors.New("decision is required"))
	default:
		return "", restate.TerminalError(fmt.Errorf("unknown decision %q", decision.String()))
	}
}

// triggerFromProto maps the wire trigger onto the stored enum. An unknown
// trigger is recorded as "unknown" so the row still says where it came from.
func triggerFromProto(trigger ctrlv1.DeploymentTrigger) db.DeploymentsTrigger {
	switch trigger {
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

// trimBytes truncates s to at most bytesMax bytes on a rune boundary. Cutting
// mid-rune yields malformed UTF-8, which MySQL strict mode rejects.
func trimBytes(s string, bytesMax int) string {
	if len(s) <= bytesMax {
		return s
	}
	cut := bytesMax
	for cut > 0 && !utf8.RuneStart(s[cut]) {
		cut--
	}
	return s[:cut]
}
