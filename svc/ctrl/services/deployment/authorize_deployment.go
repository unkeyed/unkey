package deployment

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/actor"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
	"github.com/unkeyed/unkey/svc/ctrl/internal/deploytarget"
)

// AuthorizeDeployment authorizes a deployment that is awaiting approval.
// It looks up the deployment by ID, verifies it is in awaiting_approval status,
// updates the status to pending, and triggers the deploy workflow.
func (s *Service) AuthorizeDeployment(ctx context.Context, req *connect.Request[ctrlv1.AuthorizeDeploymentRequest]) (*connect.Response[ctrlv1.AuthorizeDeploymentResponse], error) {
	if err := auth.Authenticate(req, s.bearer); err != nil {
		return nil, err
	}

	deploymentID := req.Msg.GetDeploymentId()

	if deploymentID == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("deployment_id is required"))
	}

	deployment, err := s.db.FindDeploymentById(ctx, deploymentID)
	if err != nil {
		if db.IsNotFound(err) {
			return nil, connect.NewError(connect.CodeNotFound, fmt.Errorf("deployment %s not found", deploymentID))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find deployment: %w", err))
	}

	if deployment.Status != mysqltype.DeploymentsStatusAwaitingApproval {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s is not awaiting approval (current status: %s)", deploymentID, deployment.Status))
	}
	if err := s.ensureWorkspaceCanDeploy(ctx, deployment.WorkspaceID, "authorize"); err != nil {
		return nil, err
	}

	// Loaded before the status changes, so a lookup failure does not leave the
	// deployment stuck as pending. The target is scoped to the app because a
	// repository connection is unique per app, not per project.
	target, err := deploytarget.Load(ctx, s.db,
		deployment.ProjectID, deployment.AppID, deployment.EnvironmentID, deploytarget.WithoutSecrets)
	if err != nil {
		var terminal *deploytarget.TerminalError
		if errors.As(err, &terminal) {
			return nil, connect.NewError(terminal.Code, errors.New(terminal.Message))
		}
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to load deploy target: %w", err))
	}
	if !target.GithubRepositoryFullName.Valid {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("app %s has no GitHub repo connection", deployment.AppID))
	}

	// Atomically transition from awaiting_approval → pending to prevent
	// concurrent authorization requests from triggering duplicate deploys.
	casResult, err := s.db.CompareAndSwapDeploymentStatus(ctx, db.CompareAndSwapDeploymentStatusParams{
		ID:             deploymentID,
		ExpectedStatus: mysqltype.DeploymentsStatusAwaitingApproval,
		NewStatus:      mysqltype.DeploymentsStatusPending,
		UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to update deployment status: %w", err))
	}
	rowsAffected, err := casResult.RowsAffected()
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to check rows affected: %w", err))
	}
	if rowsAffected == 0 {
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s is no longer awaiting approval (concurrent update)", deploymentID))
	}

	commitSHA := ""
	if deployment.GitCommitSha.Valid {
		commitSHA = deployment.GitCommitSha.String
	}

	branch := ""
	if deployment.GitBranch.Valid {
		branch = deployment.GitBranch.String
	}

	var prNumber int64
	if deployment.PrNumber.Valid {
		prNumber = deployment.PrNumber.Int64
	}

	// Forward the fork so the worker classifies this as a fork build and clones
	// the right repo. Today approval is only reached for live PRs (PrNumber > 0),
	// which already forces the fork path, but carrying ForkRepository keeps a
	// fork-ref-by-SHA deployment correct if it ever lands on the approval path.
	forkRepository := ""
	if deployment.ForkRepositoryFullName.Valid {
		forkRepository = deployment.ForkRepositoryFullName.String
	}

	deployReq := &hydrav1.DeployRequest{
		DeploymentId: deploymentID,
		Command:      deployment.Command,
		Source: &hydrav1.DeployRequest_Git{
			Git: &hydrav1.GitSource{
				InstallationId: target.GithubInstallationID.Int64,
				Repository:     target.GithubRepositoryFullName.String,
				CommitSha:      commitSHA,
				ContextPath:    target.DockerContext,
				DockerfilePath: target.Dockerfile.String,
				BuildCommand:   target.BuildCommand.String,
				Branch:         branch,
				PrNumber:       prNumber,
				ForkRepository: forkRepository,
			},
		},
	}

	// Keyed by deployment_id — each deployment runs as its own isolated
	// workflow so multiple deployments can build in parallel.
	invocation, sendErr := s.deploymentClient(deploymentID).Deploy().Send(ctx, deployReq)
	if sendErr != nil {
		// Revert status back to awaiting_approval since the deploy failed.
		if _, revertErr := s.db.CompareAndSwapDeploymentStatus(ctx, db.CompareAndSwapDeploymentStatusParams{
			ID:             deploymentID,
			ExpectedStatus: mysqltype.DeploymentsStatusPending,
			NewStatus:      mysqltype.DeploymentsStatusAwaitingApproval,
			UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		}); revertErr != nil {
			logger.Error("failed to revert deployment status after deploy failure",
				"deployment_id", deploymentID,
				"error", revertErr,
			)
		}
		logger.Error("failed to trigger deploy workflow after authorization",
			"deployment_id", deploymentID,
			"error", sendErr,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deploy workflow: %w", sendErr))
	}

	// Persist the invocation ID so the deployment can be cancelled later.
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

	// Replace the blocking "awaiting authorization" commit status that Create
	// posted. GitHubStatusService owns the status context string, so this write
	// updates that same check instead of adding a second one. Send, not Request:
	// the build is already dispatched, so an unreachable GitHub must not fail
	// the call.
	if commitSHA != "" {
		if _, statusErr := hydrav1.NewGitHubStatusServiceIngressClient(s.restate, deploymentID).
			SetCommitStatus().
			Send(ctx, &hydrav1.GitHubStatusCommitStatusRequest{
				InstallationId: target.GithubInstallationID.Int64,
				Repo:           target.GithubRepositoryFullName.String,
				CommitSha:      commitSHA,
				State:          hydrav1.GitHubCommitStatusState_GITHUB_COMMIT_STATUS_STATE_SUCCESS,
				TargetUrl:      "",
				Description:    "Deployment authorized and started",
			}); statusErr != nil {
			logger.Error("failed to update commit status to success", "error", statusErr)
		}
	}

	// ctrl is the single writer of this event, so an authorization from any
	// caller is audited exactly once. A request without an actor writes nothing:
	// a fabricated system actor would be worse than a missing entry.
	if a := req.Msg.GetActor(); a != nil {
		if auditErr := s.auditlogs.Insert(ctx, nil, []auditlog.AuditLog{
			{
				Event:         auditlog.DeploymentAuthorizeEvent,
				WorkspaceID:   deployment.WorkspaceID,
				Display:       fmt.Sprintf("Authorized deployment %s", deploymentID),
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
						Meta:        deploymentAuditMeta(deployment.ProjectID, deployment.AppID, deployment.EnvironmentID),
					},
				},
			},
		}); auditErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to audit authorize deployment: %w", auditErr))
		}
	}

	logger.Info("deployment authorized and workflow triggered",
		"deployment_id", deploymentID,
		"project_id", deployment.ProjectID,
	)

	return connect.NewResponse(&ctrlv1.AuthorizeDeploymentResponse{}), nil
}
