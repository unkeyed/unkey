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
	Request  = openapi.V2DeploymentsStartDeploymentRequestBody
	Response = openapi.V2DeploymentsStartDeploymentResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.DeployServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.startDeployment"
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
			Action:       rbac.StartDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   dep.EnvironmentID,
			Action:       rbac.StartDeployment,
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

	// A missing billing row means the workspace was never linked to billing and
	// is not suspended.
	billing, err := db.Query.FindWorkspaceBillingByWorkspaceID(ctx, h.DB.RO(), principal.WorkspaceID)
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error loading workspace billing"),
			fault.Public("Failed to retrieve workspace billing state."),
		)
	}

	// "Stopped" is keyed on desired_state, not status: stopping sets
	// desired_state=stopped immediately while status only flips once krane drains
	// the last instance.
	if err := deploygate.CheckStartTarget(deploygate.StartInput{
		DesiredState:   dep.DesiredState,
		IsProduction:   dep.EnvironmentIsProduction,
		SpendSuspended: billing.SpendSuspended,
	}); err != nil {
		return err
	}

	_, err = h.CtrlClient.WakeDeployment(ctx, &ctrlv1.WakeDeploymentRequest{
		DeploymentId: dep.ID,
	})
	if err != nil {
		return deployment.MapCtrlError(err, "start deployment",
			"The deployment could not be started. It must be stopped and belong to a non-production environment.")
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
