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

	if dep.Status != db.DeploymentsStatusStopped {
		return fault.New(
			"deployment not stopped",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("start target is not in stopped status"),
			fault.Public("The deployment is not stopped."),
		)
	}

	// Production deployments are never stopped, so this action does not apply.
	if dep.EnvironmentSlug == "production" {
		return fault.New(
			"production deployment",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("start is not allowed on production environments"),
			fault.Public("Production deployments cannot be started."),
		)
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
