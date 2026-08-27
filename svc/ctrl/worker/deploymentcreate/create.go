package deploymentcreate

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
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/dedup"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploytarget"
)

const (
	// maxCommitMessageLength limits commit messages to prevent oversized database entries.
	maxCommitMessageLength = 10240
	// maxCommitAuthorHandleLength limits author handles (e.g., GitHub usernames).
	maxCommitAuthorHandleLength = 256
	// maxCommitAuthorAvatarLength limits avatar URL length.
	maxCommitAuthorAvatarLength = 512
	// maxTriggerReasonLength matches the trigger_reason column width.
	maxTriggerReasonLength = 512
)

// insertOutcome is the journaled result of the insert step.
type insertOutcome struct {
	// EnvironmentID and CreatedAt feed the sibling dedup after the send.
	EnvironmentID string
	CreatedAt     int64
}

// Create is the durable half of a deployment create: insert row plus create
// audit in one transaction, start the Deploy workflow, record the invocation
// id, cancel superseded siblings. Every step is journaled, so a crashed
// create resumes instead of leaving a stuck row, and a keyed retry receives
// the journaled response.
func (s *Service) Create(ctx restate.Context, req *hydrav1.DeploymentCreateRequest) (*hydrav1.DeploymentCreateResponse, error) {
	deploymentID := req.GetDeployRequest().GetDeploymentId()
	if deploymentID == "" {
		// Ctrl generates the id before the call, so a missing one is a caller
		// bug that no retry can fix.
		return nil, restate.TerminalError(fmt.Errorf("deploy_request.deployment_id is required"))
	}

	resp := &hydrav1.DeploymentCreateResponse{
		Nonce:        req.GetNonce(),
		DeploymentId: deploymentID,
	}

	out, err := restate.Run(ctx, func(runCtx restate.RunContext) (insertOutcome, error) {
		return s.insertDeployment(runCtx, req, deploymentID)
	}, restate.WithName("insert deployment"))
	if err != nil {
		return nil, err
	}
	invocation := hydrav1.NewDeployServiceClient(ctx, deploymentID).Deploy().Send(req.GetDeployRequest())

	invocationID := invocation.GetInvocationId()
	if persistErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		return s.db.UpdateDeploymentInvocationID(runCtx, db.UpdateDeploymentInvocationIDParams{
			ID:           deploymentID,
			InvocationID: sql.NullString{Valid: true, String: invocationID},
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
	}, restate.WithName("persist invocation id")); persistErr != nil {
		return nil, persistErr
	}

	// Dedup is best-effort: the closure swallows its own error so a failed
	// cancel never fails the create. The RunVoid error still propagates; it
	// can carry Restate protocol signals (suspension, cancellation).
	if runErr := restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		if cancelErr := s.dedup.CancelOlderSiblings(runCtx, dedup.Newer{
			ID:            deploymentID,
			AppID:         req.GetAppId(),
			EnvironmentID: out.EnvironmentID,
			GitBranch:     req.GetGitCommit().GetBranch(),
			CreatedAt:     out.CreatedAt,
		}); cancelErr != nil {
			logger.Error(
				"failed to cancel superseded siblings",
				"deployment_id", deploymentID,
				"error", cancelErr,
			)
		}
		return nil
	}, restate.WithName("cancel superseded siblings")); runErr != nil {
		return nil, runErr
	}

	logger.Info(
		"deployment created",
		"deployment_id", deploymentID,
		"app_id", req.GetAppId(),
		"environment_slug", req.GetEnvironmentSlug(),
		"invocation_id", invocationID,
	)

	return resp, nil
}

func (s *Service) insertDeployment(
	ctx context.Context,
	req *hydrav1.DeploymentCreateRequest,
	deploymentID string,
) (insertOutcome, error) {
	target, loadErr := deploytarget.Load(ctx, s.db, req.GetProjectId(), req.GetAppId(), req.GetEnvironmentSlug(), deploytarget.WithSecrets)
	var terminal *deploytarget.TerminalError
	if errors.As(loadErr, &terminal) {
		// Ctrl validated this target before the call, so a terminal failure
		// here means it was deleted mid-create.
		return insertOutcome{}, restate.TerminalError(loadErr) //nolint:exhaustruct // zero value unused on error
	}
	// Anything else is transient; a plain error makes Restate retry.
	if loadErr != nil {
		return insertOutcome{}, loadErr //nolint:exhaustruct // zero value unused on error
	}

	now := time.Now().UnixMilli()
	commit := req.GetGitCommit()

	// Truncate to column widths so a long commit message doesn't bubble up
	// as a 500 from MySQL strict mode.
	commitMessage := trimLength(commit.GetCommitMessage(), maxCommitMessageLength)
	authorHandle := trimLength(commit.GetAuthorHandle(), maxCommitAuthorHandleLength)
	authorAvatarURL := trimLength(commit.GetAuthorAvatarUrl(), maxCommitAuthorAvatarLength)
	triggerReason := trimLength(req.GetTriggerReason(), maxTriggerReasonLength)
	triggeredBy := req.GetTriggeredBy()

	insertErr := db.TxRetry(ctx, s.db.RW(), func(txCtx context.Context, tx db.DBTX) error {
		if err := db.NewQueries(tx).InsertDeployment(txCtx, db.InsertDeploymentParams{
			ID:                            deploymentID,
			K8sName:                       uid.DNS1035(12),
			WorkspaceID:                   target.WorkspaceID,
			ProjectID:                     target.Project.ID,
			AppID:                         target.App.ID,
			EnvironmentID:                 target.Env.Environment.ID,
			SentinelConfig:                target.AppRuntimeSettings.SentinelConfig,
			EncryptedEnvironmentVariables: target.SecretsBlob,
			Command:                       req.GetDeployRequest().GetCommand(),
			Status:                        mysqltype.DeploymentsStatusPending,
			CreatedAt:                     now,
			UpdatedAt:                     sql.NullInt64{Valid: false, Int64: 0},
			GitCommitSha:                  sql.NullString{String: commit.GetCommitSha(), Valid: commit.GetCommitSha() != ""},
			GitBranch:                     sql.NullString{String: commit.GetBranch(), Valid: commit.GetBranch() != ""},
			GitCommitMessage:              sql.NullString{String: commitMessage, Valid: commitMessage != ""},
			GitCommitAuthorHandle:         sql.NullString{String: authorHandle, Valid: authorHandle != ""},
			GitCommitAuthorAvatarUrl:      sql.NullString{String: authorAvatarURL, Valid: authorAvatarURL != ""},
			GitCommitTimestamp:            sql.NullInt64{Int64: commit.GetTimestamp(), Valid: commit.GetTimestamp() != 0},
			CpuMillicores:                 target.AppRuntimeSettings.CpuMillicores,
			MemoryMib:                     target.AppRuntimeSettings.MemoryMib,
			StorageMib:                    target.AppRuntimeSettings.StorageMib,
			Port:                          target.AppRuntimeSettings.Port,
			ShutdownSignal:                db.DeploymentsShutdownSignal(target.AppRuntimeSettings.ShutdownSignal),
			UpstreamProtocol:              db.DeploymentsUpstreamProtocol(target.AppRuntimeSettings.UpstreamProtocol),
			Healthcheck:                   target.AppRuntimeSettings.Healthcheck,
			PrNumber:                      sql.NullInt64{Int64: 0, Valid: false},
			ForkRepositoryFullName:        sql.NullString{String: commit.GetForkRepository(), Valid: commit.GetForkRepository() != ""},
			DeploymentTrigger:             triggerFromProto(req.GetTrigger()),
			TriggeredBy:                   sql.NullString{String: triggeredBy, Valid: triggeredBy != ""},
			TriggerReason:                 sql.NullString{String: triggerReason, Valid: triggerReason != ""},
		}); err != nil {
			return err
		}

		// A rebuild skips this: it records its own deployment.rebuild event in
		// ctrl. A nil actor falls back to the system actor via actor.AuditType.
		if req.GetAction() != hydrav1.DeploymentCreateAction_DEPLOYMENT_CREATE_ACTION_REBUILD {
			a := req.GetActor()
			return s.auditlogs.Insert(txCtx, tx, []auditlog.AuditLog{
				{
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
								"projectId":   target.Project.ID,
								"appId":       target.App.ID,
								"environment": target.Env.Environment.Slug,
							},
						},
					},
				},
			})
		}
		return nil
	})
	if insertErr != nil {
		if !db.IsDuplicateKeyError(insertErr) {
			return insertOutcome{}, insertErr //nolint:exhaustruct // zero value unused on error
		}
		// duplicate key (not the idempotency key): an earlier attempt
		// committed this row before the journal recorded it, so reuse its
		// created_at instead of this attempt's later clock.
		existing, findErr := s.db.FindDeploymentById(ctx, deploymentID)
		if findErr != nil {
			return insertOutcome{}, fmt.Errorf("duplicate key on insert but deployment %s not found: %w", deploymentID, findErr) //nolint:exhaustruct // zero value unused on error
		}
		return insertOutcome{
			EnvironmentID: target.Env.Environment.ID,
			CreatedAt:     existing.CreatedAt,
		}, nil
	}

	return insertOutcome{
		EnvironmentID: target.Env.Environment.ID,
		CreatedAt:     now,
	}, nil
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
