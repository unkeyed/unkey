package deployment

import (
	"database/sql"
	"fmt"
	"time"

	restate "github.com/restatedev/sdk-go"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/svc/ctrl/internal/audit"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func (v *VirtualObject) AuthorizeDeployment(ctx restate.ObjectContext, req *hydrav1.AuthorizeDeploymentRequest) (*hydrav1.AuthorizeDeploymentResponse, error) {
	deploymentID := restate.Key(ctx)
	if id := req.GetDeploymentId(); id != "" && id != deploymentID {
		return nil, restate.ToTerminalError(fmt.Errorf("deployment_id %q does not match object key %q", id, deploymentID), restate.WithErrorCode(400))
	}

	deployment, err := v.loadDeployment(ctx, deploymentID, "authorize")
	if err != nil {
		return nil, err
	}

	if deployment.Status != mysqltype.DeploymentsStatusAwaitingApproval {
		return nil, restate.ToTerminalError(fmt.Errorf("deployment is not awaiting approval, status is %s", deployment.Status), restate.WithErrorCode(412))
	}

	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:     deployment.ProjectID,
		AppId:         deployment.AppID,
		EnvironmentId: deployment.EnvironmentID,
		Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Source:        nil,
		Trigger: &hydrav1.Trigger{
			Source: ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API,
			Actor:  req.GetActor(),
			Reason: "",
		},
	}

	commitSHA := deployment.GitCommitSha.String
	useOCI := deployment.Source == db.DeploymentsSourceOci ||
		((deployment.Source == db.DeploymentsSourceUnknown || deployment.Source == "") && commitSHA == "")
	if useOCI {
		image := deployment.ImageRequested
		if !image.Valid || image.String == "" {
			image = deployment.ImageResolved
		}
		if !image.Valid || image.String == "" {
			return nil, restate.ToTerminalError(fmt.Errorf("OCI deployment has no image reference"), restate.WithErrorCode(412))
		}
		createReq.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: image.String, Commit: nil},
		}
	} else {
		if commitSHA == "" {
			return nil, restate.ToTerminalError(fmt.Errorf("git deployment has no commit SHA"), restate.WithErrorCode(412))
		}
		// Create reads the commit, fork repository and PR number from this row
		createReq.Source = &hydrav1.DeployCreateRequest_ExistingDeployment{
			ExistingDeployment: &hydrav1.CreateExistingDeploymentSource{
				DeploymentId:  deploymentID,
				RequireLatest: false,
			},
		}
	}

	// Create only accepts an existing row whose status matches the decision, so
	// the row must be pending before the call. The compare-and-swap loses to a
	// cancel that lands after the read
	swapped, err := restate.Run(ctx, func(runCtx restate.RunContext) (bool, error) {
		result, swapErr := v.db.CompareAndSwapDeploymentStatus(runCtx, db.CompareAndSwapDeploymentStatusParams{
			ID:             deploymentID,
			ExpectedStatus: mysqltype.DeploymentsStatusAwaitingApproval,
			NewStatus:      mysqltype.DeploymentsStatusPending,
			UpdatedAt:      sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
		if swapErr != nil {
			return false, swapErr
		}
		rows, rowsErr := result.RowsAffected()
		return rows == 1, rowsErr
	}, restate.WithName("mark deployment pending"), restate.WithMaxRetryAttempts(runMaxAttempts))
	if err != nil {
		return nil, fmt.Errorf("mark deployment pending: %w", err)
	}
	if !swapped {
		return nil, restate.ToTerminalError(fmt.Errorf("deployment is no longer awaiting approval"), restate.WithErrorCode(412))
	}

	resp, err := hydrav1.NewDeployWorkflowClient(ctx, deploymentID).Create().Request(createReq)
	if err != nil {
		if revertErr := v.revertAuthorization(ctx, deploymentID); revertErr != nil {
			return nil, revertErr
		}
		return nil, restate.ToTerminalError(fmt.Errorf("create deployment: %w", err))
	}
	if outcome := resp.GetOutcome(); outcome != hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED {
		if revertErr := v.revertAuthorization(ctx, deploymentID); revertErr != nil {
			return nil, revertErr
		}
		return nil, restate.ToTerminalError(fmt.Errorf("deployment could not be started: %s %s", outcome, resp.GetDetail()), restate.WithErrorCode(412))
	}

	// Replaces the failing approval status Create posted on the commit. The run
	// has already started, so GitHub errors are logged and dropped
	if !useOCI && !v.allowUnauthenticatedDeployments {
		err = restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
			repo, findErr := v.db.FindGithubRepoConnectionByAppId(runCtx, deployment.AppID)
			if db.IsNotFound(findErr) {
				return nil
			}
			if findErr != nil {
				return findErr
			}
			return v.github.CreateCommitStatus(
				repo.InstallationID,
				repo.RepositoryFullName,
				commitSHA,
				"success",
				"",
				"Deployment authorized and started",
				githubclient.DeployAuthorizationContext,
			)
		}, restate.WithName("create commit status for authorized deployment"), restate.WithMaxRetryDuration(30*time.Second))
		if err != nil {
			logger.Error("failed to post authorized commit status",
				"deployment_id", deploymentID,
				"error", err,
			)
		}
	}

	if err := audit.InsertDeploymentLifecycle(
		ctx,
		v.auditlogs,
		req.GetActor(),
		req.GetCorrelationId(),
		audit.DeploymentRef{
			ID:            deployment.ID,
			WorkspaceID:   deployment.WorkspaceID,
			ProjectID:     deployment.ProjectID,
			AppID:         deployment.AppID,
			EnvironmentID: deployment.EnvironmentID,
		},
		auditlog.DeploymentAuthorizeEvent,
		fmt.Sprintf("Authorized deployment %s", deploymentID),
	); err != nil {
		return nil, fmt.Errorf("insert authorize deployment audit log: %w", err)
	}

	return &hydrav1.AuthorizeDeploymentResponse{}, nil
}

// revertAuthorization puts the row back to awaiting_approval when no run
// started, so the approval can be retried. RevertDeploymentAuthorization holds
// the conditions that make this safe
func (v *VirtualObject) revertAuthorization(ctx restate.ObjectContext, deploymentID string) error {
	return restate.RunVoid(ctx, func(runCtx restate.RunContext) error {
		_, err := v.db.RevertDeploymentAuthorization(runCtx, db.RevertDeploymentAuthorizationParams{
			ID:        deploymentID,
			UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
		})
		return err
	}, restate.WithName("revert deployment authorization"), restate.WithMaxRetryAttempts(runMaxAttempts))
}
