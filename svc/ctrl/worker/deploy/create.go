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
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/validation"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	commitMessageBytesMax         = 10240
	commitAuthorHandleBytesMax    = 256
	commitAuthorAvatarURLBytesMax = 512
	triggerReasonBytesMax         = 512

	// Column widths for the values that cannot be trimmed, because a cut sha or
	// branch names something other than what the caller sent. These count
	// characters, not bytes: the columns are utf8mb4 varchars, so a branch of
	// 256 multi-byte characters fits, and the API validates the same limits in
	// runes.
	commitSHACharsMax      = 40
	branchCharsMax         = 256
	forkRepositoryCharsMax = 256
	triggeredByCharsMax    = 256
	imageCharsMax          = 512
)

// Create writes a deployment row and starts its pipeline. See the proto for the
// contract.
//
// The legacy ctrl.v1.DeploymentService.CreateDeployment RPC still writes rows
// too, until its callers move over.
func (w *Workflow) Create(ctx restate.ObjectContext, req *hydrav1.DeployCreateRequest) (*hydrav1.DeployCreateResponse, error) {
	deploymentID := restate.Key(ctx)

	if err := assert.All(
		assert.NotEmpty(req.GetProjectId(), "project_id is required"),
		assert.NotEmpty(req.GetAppId(), "app_id is required"),
		assert.NotEmpty(req.GetEnvironmentId(), "environment_id is required"),
	); err != nil {
		return nil, restate.TerminalError(err)
	}

	status, err := statusForDecision(req.GetDecision())
	if err != nil {
		return nil, err
	}

	target, err := w.loadDeploymentData(ctx, req)
	if err != nil {
		return nil, err
	}

	payload, rejection, err := w.validateAndBuildPayload(ctx, req, target, status)
	if err != nil {
		return nil, err
	}
	if rejection != nil {
		return &hydrav1.DeployCreateResponse{
			DeploymentId: deploymentID,
			Outcome:      rejection.Outcome,
			Detail:       rejection.Detail,
		}, nil
	}

	if err := w.insertDeployment(ctx, deploymentID, payload, req.GetActor()); err != nil {
		return nil, err
	}

	switch req.GetDecision() {
	case hydrav1.CreateDecision_CREATE_DECISION_DEPLOY:
		if err := w.startDeployment(ctx, deploymentID, payload); err != nil {
			return nil, err
		}
	case hydrav1.CreateDecision_CREATE_DECISION_AWAIT_APPROVAL:
		if err := w.requestAuthorization(ctx, deploymentID, req, payload); err != nil {
			return nil, err
		}
	case hydrav1.CreateDecision_CREATE_DECISION_SKIP,
		hydrav1.CreateDecision_CREATE_DECISION_UNSPECIFIED:
	}

	return &hydrav1.DeployCreateResponse{
		DeploymentId: deploymentID,
		Outcome:      hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED,
		Detail:       "",
	}, nil
}

// loadDeploymentData reads the target and its workspace's billing state in one
// query. A nil row is a target that does not exist.
func (w *Workflow) loadDeploymentData(ctx restate.Context, req *hydrav1.DeployCreateRequest) (*db.FindDeployTargetRow, error) {
	return restate.Run(ctx, func(runCtx restate.RunContext) (*db.FindDeployTargetRow, error) {
		target, err := w.db.FindDeployTarget(runCtx, db.FindDeployTargetParams{
			ProjectID:     req.GetProjectId(),
			AppID:         req.GetAppId(),
			EnvironmentID: req.GetEnvironmentId(),
		})
		if err != nil {
			if db.IsNotFound(err) {
				return nil, nil
			}
			return nil, fmt.Errorf("failed to lookup deploy target: %w", err)
		}
		return &target, nil
	}, restate.WithName("load deployment data"), restate.WithMaxRetryAttempts(runMaxAttempts))
}

// checkCreatable returns why a create is refused, or nil. Billing applies to
// every create; only the settings a build needs are gated on willBuild.
func checkCreatable(target db.FindDeployTargetRow, willBuild bool) *rejection {
	if !deploygate.Entitled(target.Plan, target.PlanOverride) {
		return rejectf(
			hydrav1.CreateOutcome_CREATE_OUTCOME_NO_COMPUTE_PLAN,
			"workspace %s has no Compute plan", target.WorkspaceID,
		)
	}
	if target.SpendSuspended.Bool {
		return rejectf(
			hydrav1.CreateOutcome_CREATE_OUTCOME_SPEND_SUSPENDED,
			"workspace %s is suspended by its Compute spend cap", target.WorkspaceID,
		)
	}
	if !willBuild {
		return nil
	}

	// Deploy validates port, cpu, memory and regions too, but rejecting here
	// means the caller gets a reason and no row is written.
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
		hydrav1.CreateOutcome_CREATE_OUTCOME_ENVIRONMENT_NOT_DEPLOYABLE,
		"%s", strings.Join(messages, "; "),
	)
}

// commitFromRequest reads the commit the caller supplied on whichever source
// arm carries one. Empty fields are filled from GitHub in resolveSource.
func commitFromRequest(req *hydrav1.DeployCreateRequest) gitCommit {
	var commit gitCommit

	gc := req.GetGit().GetCommit()
	if gc == nil {
		gc = req.GetImage().GetCommit()
	}
	if gc == nil {
		return commit
	}

	commit = gitCommit{
		SHA:             gc.GetCommitSha(),
		Branch:          strings.TrimSpace(gc.GetBranch()),
		Message:         gc.GetCommitMessage(),
		AuthorHandle:    strings.TrimSpace(gc.GetAuthorHandle()),
		AuthorAvatarURL: strings.TrimSpace(gc.GetAuthorAvatarUrl()),
		Timestamp:       gc.GetTimestamp(),
		ForkRepository:  gc.GetForkRepository(),
	}
	return commit
}

// deployPayload is everything the deployment row and the Deploy request are
// made of. It crosses the Restate journal as JSON, which is why the source and
// commit are plain structs rather than the proto oneof.
type deployPayload struct {
	Rejection *rejection `json:"rejection"`

	Target db.FindDeployTargetRow `json:"target"`

	Status    mysqltype.DeploymentsStatus `json:"status"`
	CreatedAt int64                       `json:"created_at"`

	// Vault ciphertext, so safe to journal.
	Secrets []byte `json:"secrets"`

	Command  []string `json:"command"`
	PRNumber int64    `json:"pr_number"`

	Source buildSource `json:"source"`
	Commit gitCommit   `json:"commit"`

	// The branch dedup may supersede on. Empty for an inherited image, unlike
	// Commit.Branch.
	RequestedBranch string `json:"requested_branch"`

	Trigger       db.DeploymentsTrigger `json:"trigger"`
	TriggeredBy   string                `json:"triggered_by"`
	TriggerReason string                `json:"trigger_reason"`

	// The deployment a rebuild reproduces. Empty otherwise.
	RebuildSourceID string `json:"rebuild_source_id"`
}

// validateAndBuildPayload decides every rejection, then resolves the source and
// secrets into the complete [deployPayload].
func (w *Workflow) validateAndBuildPayload(
	ctx restate.Context,
	req *hydrav1.DeployCreateRequest,
	target *db.FindDeployTargetRow,
	status mysqltype.DeploymentsStatus,
) (deployPayload, *rejection, error) {
	built, err := restate.Run(ctx, func(runCtx restate.RunContext) (deployPayload, error) {
		var payload deployPayload

		// The caller already verified the target, so a miss means it was deleted
		// mid-create.
		if target == nil {
			payload.Rejection = rejectf(
				hydrav1.CreateOutcome_CREATE_OUTCOME_TARGET_NOT_FOUND,
				"no deploy target for project '%s', app '%s', environment '%s'",
				req.GetProjectId(), req.GetAppId(), req.GetEnvironmentId(),
			)
			return payload, nil
		}

		// A skip never builds: it needs no deployable environment, repository
		// connection, or GitHub answer, and refusing it would leave the push with
		// no record.
		willBuild := status != mysqltype.DeploymentsStatusSkipped

		if payload.Rejection = checkCreatable(*target, willBuild); payload.Rejection != nil {
			return payload, nil
		}

		commit := commitFromRequest(req)
		if tooLong := assert.All(
			assert.LessOrEqual(utf8.RuneCountInString(commit.SHA), commitSHACharsMax, "commit sha is too long"),
			assert.LessOrEqual(utf8.RuneCountInString(commit.Branch), branchCharsMax, "branch is too long"),
			assert.LessOrEqual(utf8.RuneCountInString(commit.ForkRepository), forkRepositoryCharsMax, "fork repository is too long"),
			assert.LessOrEqual(utf8.RuneCountInString(req.GetTriggeredBy()), triggeredByCharsMax, "triggered_by is too long"),
		); tooLong != nil {
			return payload, restate.TerminalError(tooLong)
		}

		prNumber := req.GetGit().GetPrNumber()
		source := buildSource{Image: "", Git: nil}
		secrets := []byte{}

		if willBuild {
			var err error
			if secrets, err = w.loadSecrets(runCtx, target.AppID, target.EnvironmentID); err != nil {
				return payload, err
			}

			resolved, err := w.resolveSource(runCtx, *target, req, commit)
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
				// A rebuild carries the PR number of the deployment it reproduces.
				prNumber = source.Git.PRNumber
			}
		}

		commit.Message = trimBytes(commit.Message, commitMessageBytesMax)
		commit.AuthorHandle = trimBytes(commit.AuthorHandle, commitAuthorHandleBytesMax)
		commit.AuthorAvatarURL = trimBytes(commit.AuthorAvatarURL, commitAuthorAvatarURLBytesMax)

		payload.Target = *target
		payload.Status = status
		payload.CreatedAt = time.Now().UnixMilli()
		payload.Secrets = secrets
		payload.Command = target.Command
		payload.PRNumber = prNumber
		payload.Source = source
		payload.Commit = commit
		if source.Git != nil || req.GetImage() != nil {
			payload.RequestedBranch = commit.Branch
		}
		payload.Trigger = triggerFromProto(req.GetTrigger())
		payload.TriggeredBy = req.GetTriggeredBy()
		payload.TriggerReason = trimBytes(req.GetTriggerReason(), triggerReasonBytesMax)
		payload.RebuildSourceID = req.GetExistingDeployment().GetDeploymentId()
		return payload, nil
	}, restate.WithName("validate and build deploy payload"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return deployPayload{}, nil, err
	}
	return built, built.Rejection, nil
}

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

// insertDeployment writes the row, its queued step, and its audit log in one
// transaction.
func (w *Workflow) insertDeployment(
	ctx restate.Context,
	deploymentID string,
	payload deployPayload,
	a *ctrlv1.ActorInfo,
) error {
	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		target, commit := payload.Target, payload.Commit

		// A skip resolves no source, so the commit it recorded is what says the
		// push it skipped was a git one.
		source := db.DeploymentsSourceUnknown
		switch {
		case payload.Source.Git != nil:
			source = db.DeploymentsSourceGit
		case payload.Source.Image != "":
			source = db.DeploymentsSourceOci
		case commit.SHA != "":
			source = db.DeploymentsSourceGit
		}

		insertErr := db.TxRetry(runCtx, w.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
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

			return w.auditlogs.Insert(txCtx, tx, createAuditLogs(payload, deploymentID, a))
		})

		// A duplicate key is only this create's own committed row, which no retry
		// can clear. Any other row on this id never passed the checks above, and
		// deploying it would skip the gates it is owed, approval included.
		if insertErr != nil && db.IsDuplicateKeyError(insertErr) {
			existing, findErr := w.db.FindDeploymentById(runCtx, deploymentID)
			if findErr != nil || existing.AppID != target.AppID || existing.Status != payload.Status {
				return restate.TerminalError(fmt.Errorf("deployment id %s is not available", deploymentID))
			}
			return nil
		}
		return insertErr
	}, restate.WithName("insert deployment"), restate.WithMaxRetryAttempts(runMaxAttempts))
}

// createAuditLogs builds the audit entry for a new row. A skip gets none, and a
// rebuild is recorded as deployment.rebuild naming both deployments.
func createAuditLogs(
	payload deployPayload,
	deploymentID string,
	a *ctrlv1.ActorInfo,
) []auditlog.AuditLog {
	if payload.Status == mysqltype.DeploymentsStatusSkipped {
		return nil
	}

	target := payload.Target
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

// startDeployment sends Deploy, records its invocation id, then supersedes older
// queued siblings on the branch.
func (w *Workflow) startDeployment(
	ctx restate.ObjectContext,
	deploymentID string,
	payload deployPayload,
) error {
	target := payload.Target
	invocation := hydrav1.NewDeployServiceClient(ctx, deploymentID).
		Deploy().
		Send(payload.toDeployRequest(deploymentID))

	// An empty id would leave a deployment nothing can cancel. Only a Restate
	// bug produces one, since any other malformed value panics, and the id is
	// journaled so a retry would replay it: terminal rather than forever.
	invocationID := invocation.GetInvocationId()
	if invocationID == "" {
		return restate.TerminalError(
			fmt.Errorf("restate returned an empty invocation id for deployment %s", deploymentID),
		)
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

	// Best effort: a failed cancel must not fail a deployment that is already
	// queued. The RunVoid error still propagates because it can carry Restate
	// protocol signals.
	if runErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		if cancelErr := w.dedup.CancelOlderSiblings(runCtx, dedup.Newer{
			ID:            deploymentID,
			AppID:         target.AppID,
			EnvironmentID: target.EnvironmentID,
			GitBranch:     payload.RequestedBranch,
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

// requestAuthorization posts the commit status that tells a contributor their
// push is waiting for a project member. A GitHub failure is logged, not
// returned: the row already records the wait and the create has succeeded.
func (w *Workflow) requestAuthorization(
	ctx restate.ObjectContext,
	deploymentID string,
	req *hydrav1.DeployCreateRequest,
	payload deployPayload,
) error {
	// An image has no commit to post against, and without a GitHub App there is
	// nothing to post through.
	if payload.Source.Git == nil || w.allowUnauthenticatedDeployments {
		return nil
	}

	logURL := fmt.Sprintf("%s/%s/projects/%s/deployments/%s",
		w.dashboardURL, payload.Target.WorkspaceSlug, req.GetProjectId(), deploymentID,
	)

	if err := restate.RunVoid(ctx, func(_ restate.RunContext) error {
		if statusErr := w.github.CreateCommitStatus(
			payload.Source.Git.InstallationID,
			payload.Source.Git.Repository,
			payload.Commit.SHA,
			"failure",
			logURL,
			"Awaiting authorization from a project member",
			githubclient.DeployAuthorizationContext,
		); statusErr != nil {
			logger.Error(
				"failed to post authorization commit status",
				"deployment_id", deploymentID,
				"error", statusErr,
			)
		}
		return nil
	}, restate.WithName("create commit status for authorization"),
		restate.WithMaxRetryAttempts(runMaxAttempts)); err != nil {
		return err
	}

	logger.Info(
		"deployment awaiting authorization",
		"deployment_id", deploymentID,
		"project_id", req.GetProjectId(),
	)
	return nil
}

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
		// An invalid key is corrupt stored data. No retry fixes it.
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

// rejection is why a create was refused. It is a successful response rather
// than an error because the Restate ingress turns handler errors into plain
// text, and svc/api needs a structured outcome for its status code. The detail
// can name repositories and deployments the caller may not be allowed to see,
// so svc/api decides per outcome whether to show it.
type rejection struct {
	Outcome hydrav1.CreateOutcome `json:"outcome"`
	Detail  string                `json:"detail"`
}

func rejectf(outcome hydrav1.CreateOutcome, format string, args ...any) *rejection {
	detail := fmt.Sprintf(format, args...)
	logger.Info(
		"deployment create rejected",
		"outcome", outcome.String(),
		"detail", detail,
	)
	return &rejection{Outcome: outcome, Detail: detail}
}
