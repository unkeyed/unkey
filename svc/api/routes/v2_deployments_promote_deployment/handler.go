package handler

import (
	"context"
	"net/http"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsPromoteDeploymentRequestBody
	Response = openapi.V2DeploymentsPromoteDeploymentResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.DeployServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.promoteDeployment"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	dep, err := deployment.FindDeployment(ctx, h.DB, principal.WorkspaceID, req.DeploymentId)
	if err != nil {
		return err
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.PromoteDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   dep.EnvironmentID,
			Action:       rbac.PromoteDeployment,
		}),
	))
	if err != nil {
		return fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking deployment existence"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	// The deployment starts serving live traffic the moment routes swap, so it
	// must have completed successfully.
	if dep.Status != db.DeploymentsStatusReady {
		return fault.New(
			"deployment not ready",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("promotion target is not in ready status"),
			fault.Public("The deployment is not ready."),
		)
	}

	// A demoted deployment keeps status ready while it drains toward standby
	// (only krane's final instance report flips it to stopped), so status alone
	// would let traffic swap onto a deployment that is shutting down.
	if dep.DesiredState != db.DeploymentsDesiredStateRunning {
		return fault.New(
			"deployment shutting down",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("promotion target desired_state is not running"),
			fault.Public("The deployment is shutting down and cannot serve traffic."),
		)
	}

	// Promote swaps apps.current_deployment_id, which tracks the production live
	// deployment, so it only applies to production.
	if dep.EnvironmentSlug != "production" {
		return fault.New(
			"not a production deployment",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("promote is only allowed on production environments"),
			fault.Public("Only production deployments can be promoted."),
		)
	}

	app, err := db.Query.FindAppById(ctx, h.DB.RO(), dep.AppID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load app for promotion eligibility"),
			fault.Public("Failed to resolve the current live deployment."),
		)
	}
	if !app.CurrentDeploymentID.Valid || app.CurrentDeploymentID.String == "" {
		return fault.New(
			"no live deployment",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("app has no current deployment to promote over"),
			fault.Public("The app has no live deployment to promote over."),
		)
	}
	// Promoting the live deployment is only meaningful as a rollback
	// confirmation; otherwise it is a no-op the caller likely did not intend.
	if app.CurrentDeploymentID.String == dep.ID && !app.IsRolledBack {
		return fault.New(
			"deployment already live",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("promotion target is already the live deployment"),
			fault.Public("The deployment is already live."),
		)
	}

	_, err = h.CtrlClient.Promote(ctx, &ctrlv1.PromoteRequest{
		TargetDeploymentId: dep.ID,
	})
	if err != nil {
		return deployment.MapCtrlError(err, "promote deployment",
			"The deployment could not be promoted. It must be ready, belong to the production environment, and not already be live.")
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
