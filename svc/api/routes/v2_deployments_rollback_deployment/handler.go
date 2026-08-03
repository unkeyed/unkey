package handler

import (
	"context"
	"net/http"

	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsRollbackDeploymentRequestBody
	Response = openapi.V2DeploymentsRollbackDeploymentResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.DeployServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.rollbackDeployment"
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
			Action:       rbac.RollbackDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   dep.EnvironmentID,
			Action:       rbac.RollbackDeployment,
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

	// The rollback source is derived from the app's current live pointer, not
	// trusted from input. The ctrl workflow re-validates, so a concurrent promotion
	// that moves the pointer fails the rollback rather than swapping traffic onto a
	// stale source.
	app, err := db.Query.FindAppById(ctx, h.DB.RO(), dep.AppID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load app for rollback source derivation"),
			fault.Public("Failed to resolve the current live deployment."),
		)
	}

	if err := deploygate.CheckRollbackTarget(deploygate.RollbackInput{
		Status:              dep.Status,
		DesiredState:        dep.DesiredState,
		EnvironmentKind:     dep.EnvironmentKind,
		CurrentDeploymentID: app.CurrentDeploymentID.String,
		DeploymentID:        dep.ID,
	}); err != nil {
		return err
	}

	_, err = h.CtrlClient.Rollback(ctx, &ctrlv1.RollbackRequest{
		SourceDeploymentId: app.CurrentDeploymentID.String,
		TargetDeploymentId: dep.ID,
	})
	if err != nil {
		return deployment.MapCtrlError(err, "rollback deployment",
			"The rollback could not be performed. The target must be a ready production deployment in the same app and environment as the current live deployment.")
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
