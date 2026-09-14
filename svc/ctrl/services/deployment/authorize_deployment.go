package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	githubclient "github.com/unkeyed/unkey/pkg/github"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/svc/ctrl/internal/auth"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
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
	if err := s.ensureWorkspaceCanDeploy(ctx, deployment.WorkspaceID); err != nil {
		return nil, err
	}

	commitSHA := deployment.GitCommitSha.String
	useOCI := deployment.Source == db.DeploymentsSourceOci ||
		((deployment.Source == db.DeploymentsSourceUnknown || deployment.Source == "") && commitSHA == "")

	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:     deployment.ProjectID,
		AppId:         deployment.AppID,
		EnvironmentId: deployment.EnvironmentID,
		Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Source:        nil,
		// This deployment already has a row, so Create's insert does nothing
		// and the trigger below is never written. Create still reads it: an
		// Unkey trigger means rebuild, anything else means redeploy
		Trigger: &hydrav1.Trigger{
			Source: ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_DASHBOARD,
			Actor:  nil,
			Reason: "",
		},
	}

	var repoConn *db.GithubRepoConnection
	if useOCI {
		image := deployment.ImageRequested
		if !image.Valid || image.String == "" {
			image = deployment.ImageResolved
		}
		if !image.Valid || image.String == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("OCI deployment %s has no image reference", deploymentID))
		}
		// The git arm below points Create at this deployment's own row, which
		// does not work for an image. Create reads images from
		// image_resolved, and that column is only filled in once a build
		// finishes, so a deployment awaiting approval has nothing there yet
		createReq.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: image.String, Commit: nil},
		}
	} else {
		if commitSHA == "" {
			return nil, connect.NewError(connect.CodeFailedPrecondition,
				fmt.Errorf("Git deployment %s has no commit SHA", deploymentID))
		}

		connection, connectionErr := s.db.FindGithubRepoConnectionByAppId(ctx, deployment.AppID)
		if connectionErr != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to find repo connection: %w", connectionErr))
		}
		repoConn = &connection

		// Point Create at this deployment's own row. Create then reads the
		// commit, the fork repository and the PR number out of the row
		// itself, so this code cannot get one of them wrong or leave it out
		createReq.Source = &hydrav1.DeployCreateRequest_ExistingDeployment{
			ExistingDeployment: &hydrav1.CreateExistingDeploymentSource{
				DeploymentId:  deploymentID,
				RequireLatest: false,
			},
		}
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

	// Create applies the same checks a push gets, Compute plan, spend cap,
	// schedulable region and the cpu and memory bounds, and then starts the
	// run. Going through it is what stops an approval from skipping them. It
	// also writes the invocation id that cancelling a deployment needs.
	resp, createErr := s.deploymentClient(deploymentID).Create().Request(ctx, createReq)
	if createErr != nil {
		s.revertAuthorization(ctx, deploymentID)
		logger.Error("failed to trigger deploy workflow after authorization",
			"deployment_id", deploymentID,
			"error", createErr,
		)
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("failed to trigger deploy workflow: %w", createErr))
	}

	// Create reports a refusal in the response rather than as an error, so
	// this is not an error path in Go. The row is already pending by now, and
	// without the revert it would stay that way: nothing would build it, and
	// the dashboard only offers an approve button for awaiting_approval
	if outcome := resp.GetOutcome(); outcome != hydrav1.CreateOutcome_CREATE_OUTCOME_CREATED {
		s.revertAuthorization(ctx, deploymentID)
		return nil, connect.NewError(connect.CodeFailedPrecondition,
			fmt.Errorf("deployment %s could not be started: %s", deploymentID, outcome.String()))
	}

	// Update commit status on GitHub
	if commitSHA != "" && repoConn != nil {
		if statusErr := s.github.CreateCommitStatus(
			repoConn.InstallationID,
			repoConn.RepositoryFullName,
			commitSHA,
			"success",
			"",
			"Deployment authorized and started",
			githubclient.DeployAuthorizationContext,
		); statusErr != nil {
			logger.Error("failed to update commit status to success", "error", statusErr)
		}
	}

	logger.Info("deployment authorized and workflow triggered",
		"deployment_id", deploymentID,
		"project_id", deployment.ProjectID,
	)

	return connect.NewResponse(&ctrlv1.AuthorizeDeploymentResponse{}), nil
}

// revertAuthorization puts the deployment back to awaiting_approval so the
// approve button reappears, for when an approval was accepted but no run
// started. The conditions that make it safe to do are in the statement rather
// than here; see RevertDeploymentAuthorization for why.
func (s *Service) revertAuthorization(ctx context.Context, deploymentID string) {
	if _, err := s.db.RevertDeploymentAuthorization(ctx, db.RevertDeploymentAuthorizationParams{
		ID:        deploymentID,
		UpdatedAt: sql.NullInt64{Int64: time.Now().UnixMilli(), Valid: true},
	}); err != nil {
		logger.Error("failed to revert deployment authorization",
			"deployment_id", deploymentID,
			"error", err,
		)
	}
}
