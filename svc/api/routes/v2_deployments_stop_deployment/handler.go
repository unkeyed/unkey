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
	Request  = openapi.V2DeploymentsStopDeploymentRequestBody
	Response = openapi.V2DeploymentsStopDeploymentResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.DeployServiceClient
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.stopDeployment"
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
			Action:       rbac.StopDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   dep.EnvironmentID,
			Action:       rbac.StopDeployment,
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

	if dep.Status != db.DeploymentsStatusReady {
		return fault.New(
			"deployment not running",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("stop target is not in ready status"),
			fault.Public("The deployment is not running."),
		)
	}

	// A draining deployment keeps status ready until krane removes its last
	// instance, so desired_state is the only signal that a stop is already in
	// flight. Ctrl enforces the same rule; checking it here avoids a doomed
	// round-trip and returns a precise message.
	if dep.DesiredState != db.DeploymentsDesiredStateRunning {
		return fault.New(
			"deployment already stopping",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("stop target desired_state is not running"),
			fault.Public("The deployment is already stopping."),
		)
	}

	// Production deployments are never stopped, so this action does not apply.
	// The joined slug lets us gate here without a second environment lookup.
	if dep.EnvironmentSlug == "production" {
		return fault.New(
			"production deployment",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("stop is not allowed on production environments"),
			fault.Public("Production deployments cannot be stopped."),
		)
	}

	_, err = h.CtrlClient.StopDeployment(ctx, &ctrlv1.StopDeploymentRequest{
		DeploymentId: dep.ID,
	})
	if err != nil {
		return deployment.MapCtrlError(err, "stop deployment",
			"The deployment could not be stopped. It must be running and belong to a non-production environment.")
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
