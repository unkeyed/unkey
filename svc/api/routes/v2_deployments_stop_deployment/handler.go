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

	dep, err := deployment.FindAuthorized(ctx, h.DB, principal, req.DeploymentId, rbac.StopDeployment)
	if err != nil {
		return err
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

	if err := deployment.RequireNonProduction(ctx, h.DB, dep.EnvironmentID,
		"Production deployments cannot be stopped."); err != nil {
		return err
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
