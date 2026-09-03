package deploy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/deploy/deployfail"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

// Three kinds of failure, and picking the wrong one wedges a deployment:
//
//   - A [rejection] is a caller precondition. Answered successfully, with a reason.
//   - restate.TerminalError is a bug or corrupt data. Restate stops.
//   - Anything else is assumed transient and retried runMaxAttempts times.
//
// Prefer a rejection whenever the caller could act on the answer.

// Byte limits for the commit metadata columns. MySQL strict mode rejects a
// value wider than its column, so an untrimmed commit message fails the insert.
const (
	commitMessageBytesMax         = 10240
	commitAuthorHandleBytesMax    = 256
	commitAuthorAvatarURLBytesMax = 512
	triggerReasonBytesMax         = 512
)

// rejection is a caller precondition no retry can fix: no plan, no repository
// connection, an unresolvable commit. It answers successfully rather than
// failing, because the Restate ingress flattens a handler failure into
// unstructured text and svc/api needs a value to branch on for its 412.
type rejection struct {
	Reason hydrav1.CreateRejectionReason `json:"reason"`
	Detail string                        `json:"detail"`
}

// rejectf builds a [rejection] whose detail is formatted like fmt.Errorf.
func rejectf(reason hydrav1.CreateRejectionReason, format string, args ...any) *rejection {
	return &rejection{Reason: reason, Detail: fmt.Sprintf(format, args...)}
}

// Create writes a deployment row and starts its pipeline. See the proto for the
// contract.
//
// Assert, gate, build the payload, record it, act on the decision. Every stage
// that touches the database is journaled, so a crash resumes where it died.
//
// The legacy ctrl.v1.DeploymentService.CreateDeployment RPC still writes rows
// too, until its callers move over.
func (w *Workflow) Create(ctx restate.ObjectContext, req *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
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

	// Gates: target exists, workspace entitled and not suspended, environment
	// deployable.
	gates, err := restate.Run(ctx, func(runCtx restate.RunContext) (gateResult, error) {
		return w.checkGates(runCtx, req, status)
	}, restate.WithName("check gates"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, err
	}
	if gates.Rejection != nil {
		// The detail is logged, never returned: it names repositories and
		// deployments the caller may have no right to read.
		logger.Info(
			"deployment create rejected",
			"deployment_id", deploymentID,
			"app_id", req.GetAppId(),
			"reason", gates.Rejection.Reason.String(),
			"detail", gates.Rejection.Detail,
		)
		return &hydrav1.DeployCreateResponse{
			DeploymentId:    deploymentID,
			Outcome:         hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED,
			RejectionReason: gates.Rejection.Reason,
		}, nil
	}

	// Payload. Everything the row and the Deploy request are made of.
	payload, err := restate.Run(ctx, func(runCtx restate.RunContext) (deployPayload, error) {
		return w.buildPayload(runCtx, req, gates.Target, status)
	}, restate.WithName("build deploy payload"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, err
	}
	if payload.Rejection != nil {
		// The detail is logged, never returned: it names repositories and
		// deployments the caller may have no right to read.
		logger.Info(
			"deployment create rejected",
			"deployment_id", deploymentID,
			"app_id", req.GetAppId(),
			"reason", payload.Rejection.Reason.String(),
			"detail", payload.Rejection.Detail,
		)
		return &hydrav1.DeployCreateResponse{
			DeploymentId:    deploymentID,
			Outcome:         hydrav1.CreateOutcome_CREATE_OUTCOME_REJECTED,
			RejectionReason: payload.Rejection.Reason,
		}, nil
	}

	// Record: the row, its queued step, and its audit log, in one transaction.
	if err := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return w.recordDeployment(runCtx, deploymentID, gates.Target, payload, req.GetActor())
	}, restate.WithName("record deployment"), restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return nil, err
	}

	switch req.GetDecision() {
	case hydrav1.CreateDecision_CREATE_DECISION_DEPLOY:
		if err := w.startDeploy(ctx, deploymentID, gates.Target, payload); err != nil {
			return nil, err
		}

	case hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL:
		// This gains a body when the GitHub webhook moves over: the commit status
		// telling a contributor their push is waiting is still posted by
		// githubwebhook.blockDeploymentForApproval today.

	// Nothing to do once the row exists. Listed rather than defaulted so that a
	// new decision fails the exhaustive linter until someone handles it here.
	case hydrav1.CreateDecision_CREATE_DECISION_SKIP,
		hydrav1.CreateDecision_CREATE_DECISION_UNSPECIFIED:
	}

	return &hydrav1.DeployCreateResponse{
		DeploymentId:    deploymentID,
		Outcome:         hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		RejectionReason: hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_UNSPECIFIED,
	}, nil
}

// gateResult is what the gate stage decided. It carries the target because the
// first gate is proving the target exists, and it is the only place the target
// crosses the journal: later stages take it as an argument instead.
type gateResult struct {
	// Rejection set means nothing was written and nothing should be.
	Rejection *rejection `json:"rejection"`

	// Target is the project, app, environment and settings, read when the create
	// runs rather than when it was requested.
	Target db.FindDeployTargetRow `json:"target"`
}

// checkGates proves the target exists and may deploy.
//
// A skip passes the deployability gate untested: it never builds, and refusing
// it would leave the push with no record of having been seen at all.
func (w *Workflow) checkGates(
	ctx context.Context,
	req *hydrav1.DeployCreateRequest,
	status mysqltype.DeploymentsStatus,
) (gateResult, error) {
	var result gateResult

	target, err := w.db.FindDeployTarget(ctx, db.FindDeployTargetParams{
		ProjectID:   req.GetProjectId(),
		AppID:       req.GetAppId(),
		Environment: req.GetEnvironment(),
	})
	if err != nil {
		if !db.IsNotFound(err) {
			return result, fmt.Errorf("failed to lookup deploy target: %w", err)
		}
		// The caller checked this target first, so a miss here means it was
		// deleted mid-create. A rejection, not an error: no retry brings it back.
		result.Rejection = rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_TARGET_NOT_FOUND,
			"no deploy target for project '%s', app '%s', environment '%s'",
			req.GetProjectId(), req.GetAppId(), req.GetEnvironment(),
		)
		return result, nil
	}

	// Entitlement first, and only the first refusal is reported: a workspace that
	// cannot deploy at all has no use for its environment's problems. The
	// environment gate does report all of its own at once.
	rejected, err := w.checkDeployGate(ctx, target.WorkspaceID)
	if err != nil {
		return result, err
	}
	if rejected == nil && status != mysqltype.DeploymentsStatusSkipped {
		rejected = checkEnvironmentDeployable(target)
	}
	if rejected != nil {
		result.Rejection = rejected
		return result, nil
	}

	result.Target = target
	return result, nil
}

// checkDeployGate is the billing gate every create passes.
//
// Plan enforcement follows enforceDeployGate; the spend cap always blocks.
func (w *Workflow) checkDeployGate(ctx context.Context, workspaceID string) (*rejection, error) {
	entitlement, err := w.db.FindWorkspaceDeployEntitlement(ctx, workspaceID)
	if err != nil {
		if !db.IsNotFound(err) {
			return nil, err
		}
		// The query left-joins billing onto the workspace, so a workspace with no
		// billing row still answers with empty plan columns. A miss means the
		// workspace itself is gone, which no retry and no plan change fixes.
		return rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_TARGET_NOT_FOUND,
			"workspace %s not found", workspaceID,
		), nil
	}

	if !deploygate.Entitled(entitlement.Plan, entitlement.PlanOverride) {
		if w.enforceDeployGate {
			return rejectf(
				hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_NO_COMPUTE_PLAN,
				"workspace %s has no Compute plan", workspaceID,
			), nil
		}
		logger.Warn(
			"deploy gate would block deployment create",
			"event", "deploy_gate.would_block",
			"workspaceId", workspaceID,
		)
	}

	if entitlement.SpendSuspended.Bool {
		return rejectf(
			hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_SPEND_SUSPENDED,
			"workspace %s is suspended by its Compute spend cap", workspaceID,
		), nil
	}

	return nil, nil
}

// checkEnvironmentDeployable refuses an environment whose runtime or regional
// settings the pipeline cannot satisfy. Deploy checks both again as a backstop,
// but a row written here could only ever reach FAILED.
func checkEnvironmentDeployable(target db.FindDeployTargetRow) *rejection {
	// The offending value is what makes it actionable, and svc/api's own
	// pre-flight already reports it that way.
	messages := make([]string, 0, 2)
	for _, violation := range deployfail.RuntimeViolations(target.Port, target.CpuMillicores, target.MemoryMib) {
		messages = append(messages, fmt.Sprintf("%s (is %d)", violation.Message, violation.Actual))
	}
	if !target.HasSchedulableRegion {
		messages = append(messages, deployfail.MsgNoSchedulableRegions)
	}

	if len(messages) == 0 {
		return nil
	}
	return rejectf(
		hydrav1.CreateRejectionReason_CREATE_REJECTION_REASON_ENVIRONMENT_NOT_DEPLOYABLE,
		"environment %q is not deployable: %s", target.EnvironmentSlug, strings.Join(messages, "; "),
	)
}

// deployPayload is everything the create derived that is not already on the
// target. Nothing downstream reads the request for a column, so the row cannot
// disagree with what gets built.
//
// It crosses the journal as JSON, so the source and commit are flattened to
// scalars: a proto oneof does not survive that trip. Strings arrive trimmed to
// their column widths.
type deployPayload struct {
	// Rejection set means nothing was written and nothing should be.
	Rejection *rejection `json:"rejection"`

	Status mysqltype.DeploymentsStatus `json:"status"`

	// Settled in this journaled stage, so a retry reuses the first attempt's
	// value rather than a later clock.
	CreatedAt int64 `json:"created_at"`

	// The environment's variables as a SecretsConfig. Vault ciphertext, never
	// plaintext, so the journal is a safe place to carry it.
	Secrets []byte `json:"secrets"`

	// Resolved once so the row and the Deploy request agree on what runs.
	Command []string `json:"command"`

	// Recorded even for a skip, which resolves no source to carry it.
	PRNumber int64 `json:"pr_number"`

	Source buildSource `json:"source"`
	Commit gitCommit   `json:"commit"`

	Trigger       db.DeploymentsTrigger `json:"trigger"`
	TriggeredBy   string                `json:"triggered_by"`
	TriggerReason string                `json:"trigger_reason"`

	// The deployment this one reproduces. Turns the audit entry into a rebuild.
	RebuildSourceID string `json:"rebuild_source_id"`
}

// buildPayload turns a gated request into the complete [deployPayload].
//
// A skip resolves no source and holds no secrets: recording it must not depend
// on the repository connection or on GitHub answering.
func (w *Workflow) buildPayload(
	ctx context.Context,
	req *hydrav1.DeployCreateRequest,
	target db.FindDeployTargetRow,
	status mysqltype.DeploymentsStatus,
) (deployPayload, error) {
	var payload deployPayload

	willBuild := status != mysqltype.DeploymentsStatusSkipped

	secrets := []byte{}
	if willBuild {
		var err error
		secrets, err = w.loadSecrets(ctx, target.AppID, target.EnvironmentID)
		if err != nil {
			return payload, err
		}
	}

	// Caller-supplied commit metadata, whitespace normalized. Empty fields mean
	// unknown and are filled from GitHub in resolveSource, so an image redeploy
	// never synthesizes git metadata.
	var commit gitCommit
	if gc := req.GetGit().GetCommit(); gc != nil {
		commit = gitCommit{
			SHA:             gc.GetCommitSha(),
			Branch:          strings.TrimSpace(gc.GetBranch()),
			Message:         gc.GetCommitMessage(),
			AuthorHandle:    strings.TrimSpace(gc.GetAuthorHandle()),
			AuthorAvatarURL: strings.TrimSpace(gc.GetAuthorAvatarUrl()),
			Timestamp:       gc.GetTimestamp(),
			ForkRepository:  gc.GetForkRepository(),
		}
	}
	prNumber := req.GetGit().GetPrNumber()
	source := buildSource{Image: "", Git: nil}

	if willBuild {
		resolved, err := w.resolveSource(ctx, target, req, commit)
		if err != nil {
			return payload, err
		}
		if resolved.Rejection != nil {
			payload.Rejection = resolved.Rejection
			return payload, nil
		}
		if resolved.Source.Git == nil && resolved.Source.Image == "" {
			return payload, restate.TerminalError(errors.New("no build source: set git, image, or existing_deployment"))
		}
		source, commit = resolved.Source, resolved.Commit
		if source.Git != nil {
			// A rebuild takes the PR number off the deployment it reproduces, not
			// off the request.
			prNumber = source.Git.PRNumber
		}
	}

	// A per-request command wins over the app's default, so the row records what
	// actually runs.
	command := target.Command
	if len(req.GetCommand()) > 0 {
		command = req.GetCommand()
	}

	commit.Message = trimBytes(commit.Message, commitMessageBytesMax)
	commit.AuthorHandle = trimBytes(commit.AuthorHandle, commitAuthorHandleBytesMax)
	commit.AuthorAvatarURL = trimBytes(commit.AuthorAvatarURL, commitAuthorAvatarURLBytesMax)

	payload.Status = status
	payload.CreatedAt = time.Now().UnixMilli()
	payload.Secrets = secrets
	payload.Command = command
	payload.PRNumber = prNumber
	payload.Source = source
	payload.Commit = commit
	payload.Trigger = triggerFromProto(req.GetTrigger())
	payload.TriggeredBy = req.GetTriggeredBy()
	payload.TriggerReason = trimBytes(req.GetTriggerReason(), triggerReasonBytesMax)
	payload.RebuildSourceID = req.GetExistingDeployment().GetDeploymentId()
	return payload, nil
}

// toDeployRequest assembles the request Deploy consumes. The payload already
// holds the source, the command, and the commit, so only the id it runs under
// comes from outside.
func (p deployPayload) toDeployRequest(deploymentID string) *hydrav1.DeployRequest {
	if p.Source.Git == nil {
		return &hydrav1.DeployRequest{
			DeploymentId: deploymentID,
			Command:      p.Command,
			Source: &hydrav1.DeployRequest_OciImage{
				OciImage: &hydrav1.OciImage{Image: p.Source.Image},
			},
		}
	}

	git := p.Source.Git
	return &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		Command:      p.Command,
		Source: &hydrav1.DeployRequest_Git{
			Git: &hydrav1.GitSource{
				InstallationId: git.InstallationID,
				Repository:     git.Repository,
				CommitSha:      p.Commit.SHA,
				ContextPath:    git.ContextPath,
				DockerfilePath: git.DockerfilePath,
				BuildCommand:   git.BuildCommand,
				Branch:         p.Commit.Branch,
				ForkRepository: p.Commit.ForkRepository,
				PrNumber:       git.PRNumber,
			},
		},
	}
}

// recordDeployment writes the row, its queued step, and its audit log in one
// transaction. Half a record is worse than none: no queued step means no queue
// time, and no audit entry means nobody can see who asked for the deployment.
func (w *Workflow) recordDeployment(
	ctx context.Context,
	deploymentID string,
	target db.FindDeployTargetRow,
	payload deployPayload,
	a *ctrlv1.ActorInfo,
) error {
	commit := payload.Commit

	// A skip resolves no source and records unknown, the same as a row that
	// predates source tracking.
	source := db.DeploymentsSourceUnknown
	switch {
	case payload.Source.Git != nil:
		source = db.DeploymentsSourceGit
	case payload.Source.Image != "":
		source = db.DeploymentsSourceOci
	}

	insertErr := db.TxRetry(ctx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if err := db.NewQueries(tx).InsertDeployment(txCtx, db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   target.WorkspaceID,
			ProjectID:                     target.ProjectID,
			AppID:                         target.AppID,
			EnvironmentID:                 target.EnvironmentID,
			Source:                        source,
			ImageRequested:                sql.NullString{String: payload.Source.Image, Valid: payload.Source.Image != ""},
			SentinelConfig:                target.SentinelConfig,
			EncryptedEnvironmentVariables: payload.Secrets,
			Command:                       payload.Command,
			Status:                        payload.Status,
			CreatedAt:                     payload.CreatedAt,
			UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
			GitCommitSha:                  sql.NullString{String: commit.SHA, Valid: commit.SHA != ""},
			GitBranch:                     sql.NullString{String: commit.Branch, Valid: commit.Branch != ""},
			GitCommitMessage:              sql.NullString{String: commit.Message, Valid: commit.Message != ""},
			GitCommitAuthorHandle:         sql.NullString{String: commit.AuthorHandle, Valid: commit.AuthorHandle != ""},
			GitCommitAuthorAvatarUrl:      sql.NullString{String: commit.AuthorAvatarURL, Valid: commit.AuthorAvatarURL != ""},
			GitCommitTimestamp:            sql.NullInt64{Int64: commit.Timestamp, Valid: commit.Timestamp != 0},
			CpuMillicores:                 target.CpuMillicores,
			MemoryMib:                     target.MemoryMib,
			StorageMib:                    target.StorageMib,
			Port:                          target.Port,
			ShutdownSignal:                db.DeploymentsShutdownSignal(target.ShutdownSignal),
			UpstreamProtocol:              db.DeploymentsUpstreamProtocol(target.UpstreamProtocol),
			Healthcheck:                   target.Healthcheck,
			PrNumber:                      sql.NullInt64{Int64: payload.PRNumber, Valid: payload.PRNumber != 0},
			ForkRepositoryFullName:        sql.NullString{String: commit.ForkRepository, Valid: commit.ForkRepository != ""},
			DeploymentTrigger:             payload.Trigger,
			TriggeredBy:                   sql.NullString{String: payload.TriggeredBy, Valid: payload.TriggeredBy != ""},
			TriggerReason:                 sql.NullString{String: payload.TriggerReason, Valid: payload.TriggerReason != ""},
		}); err != nil {
			return err
		}

		// Deploy ends this step but never inserts it.
		if err := db.NewQueries(tx).InsertDeploymentStep(txCtx, db.InsertDeploymentStepParams{
			WorkspaceID:   target.WorkspaceID,
			ProjectID:     target.ProjectID,
			AppID:         target.AppID,
			EnvironmentID: target.EnvironmentID,
			DeploymentID:  deploymentID,
			Step:          db.DeploymentStepsStepQueued,
			StartedAt:     uint64(payload.CreatedAt),
		}); err != nil {
			return err
		}

		return w.auditlogs.Insert(txCtx, tx, createAuditLogs(target, payload, deploymentID, a))
	})

	// A duplicate key means this already committed and we are running again:
	// TxRetry re-runs the transaction on anything that looks transient, and a
	// commit whose acknowledgement was lost looks exactly like that. Failing
	// here would burn every retry on an error no attempt can clear, leaving a
	// row that never builds and, with no invocation id, can never be cancelled.
	if insertErr != nil && !db.IsDuplicateKeyError(insertErr) {
		return insertErr
	}
	return nil
}

// createAuditLogs builds the audit entry for a new row. A skip gets none.
//
// A rebuild is recorded as deployment.rebuild naming both deployments. The UNKEY
// trigger identifies it, because Unkey itself only ever asks for rebuilds.
func createAuditLogs(
	target db.FindDeployTargetRow,
	payload deployPayload,
	deploymentID string,
	a *ctrlv1.ActorInfo,
) []auditlog.AuditLog {
	if payload.Status == mysqltype.DeploymentsStatusSkipped {
		return nil
	}

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

	if payload.Trigger != db.DeploymentsTriggerUnkey || payload.RebuildSourceID == "" {
		return []auditlog.AuditLog{entry}
	}

	entry.Event = auditlog.DeploymentRebuildEvent
	entry.Display = fmt.Sprintf("Unkey rebuilt deployment %s as %s", payload.RebuildSourceID, deploymentID)
	if payload.TriggerReason != "" {
		entry.Display = fmt.Sprintf("%s (reason: %s)", entry.Display, payload.TriggerReason)
	}
	entry.Resources[0].Meta["role"] = "new"
	entry.Resources = append([]auditlog.AuditLogResource{
		{
			Type:        auditlog.DeploymentResourceType,
			ID:          payload.RebuildSourceID,
			Name:        "",
			DisplayName: payload.RebuildSourceID,
			Meta:        map[string]any{"role": "source"},
		},
	}, entry.Resources...)

	return []auditlog.AuditLog{entry}
}

// startDeploy sends Deploy, records the invocation id it returns, then
// supersedes older queued siblings on the branch.
//
// Send, not Request: Deploy is another exclusive handler on this key and cannot
// start until this one returns. The id is written straight after the send so a
// cancel arriving next instant has something to cancel; Deploy writes it again
// because this write still races that cancel.
func (w *Workflow) startDeploy(
	ctx restate.ObjectContext,
	deploymentID string,
	target db.FindDeployTargetRow,
	payload deployPayload,
) error {
	invocation := hydrav1.NewDeployServiceClient(ctx, deploymentID).
		Deploy().
		Send(payload.toDeployRequest(deploymentID))

	// Restate always answers a successful Send with an invocation id. An empty
	// one is a bug, and it would leave a deployment no cancel can reach.
	invocationID := invocation.GetInvocationId()
	if invocationID == "" {
		return fmt.Errorf("restate returned an empty invocation id for deployment %s", deploymentID)
	}

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
			AppID:         target.AppID,
			EnvironmentID: target.EnvironmentID,
			GitBranch:     payload.Commit.Branch,
			CreatedAt:     payload.CreatedAt,
		}); cancelErr != nil {
			logger.Error(
				"failed to cancel superseded siblings",
				"deployment_id", deploymentID,
				"error", cancelErr,
			)
		}
		return nil
	}, restate.WithName("cancel superseded siblings"), restate.WithMaxRetryAttempts(runMaxAttempts)); runErr != nil {
		return runErr
	}

	logger.Info(
		"deployment workflow started",
		"deployment_id", deploymentID,
		"app_id", target.AppID,
		"invocation_id", invocationID,
	)
	return nil
}

// statusForDecision maps the caller's decision onto the status the row starts
// in. A decision is what the caller wants; a status is what the deployment is.
//
// An unset decision is a caller bug, not a default: reading it as DEPLOY would
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

// loadSecrets marshals the environment's variables into the SecretsConfig the
// deployment row stores. The values are vault ciphertext.
func (w *Workflow) loadSecrets(ctx context.Context, appID, environmentID string) ([]byte, error) {
	envVars, err := w.db.FindAppEnvVarsByAppAndEnv(ctx, db.FindAppEnvVarsByAppAndEnvParams{
		AppID:         appID,
		EnvironmentID: environmentID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to fetch app environment variables: %w", err)
	}
	if len(envVars) == 0 {
		return []byte{}, nil
	}

	config := &ctrlv1.SecretsConfig{Secrets: make(map[string]string, len(envVars))}
	for _, ev := range envVars {
		// A key that cannot be exported into a container is a defect in stored
		// data. No retry fixes it.
		if !validation.IsValidEnvVarKey(ev.Key) {
			return nil, restate.TerminalError(fmt.Errorf(
				"environment variable key %q is invalid: %s", ev.Key, validation.ErrMsgInvalidEnvVarKey,
			))
		}
		config.Secrets[ev.Key] = ev.Value
	}

	marshaled, err := protojson.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal secrets config: %w", err)
	}
	return marshaled, nil
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
