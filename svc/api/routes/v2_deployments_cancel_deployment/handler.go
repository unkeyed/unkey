package handler

import (
	"context"
	"fmt"
	"net/http"
	"slices"

	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsCancelDeploymentRequestBody
	Response = openapi.V2DeploymentsCancelDeploymentResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.cancelDeployment"
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

	dep, err := deployment.FindDeployment(ctx, h.DB, principal.AuthorizedWorkspaceID, req.DeploymentId)
	if err != nil {
		return err
	}

	err = principal.Authorize(rbac.U(
		urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(dep.ProjectID).App(dep.AppID).Environment(dep.EnvironmentID).Deployment(dep.ID),
		permissions.Write,
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Deployment.NotFound.URN(),
			"The requested deployment does not exist.",
		)
	}

	if !slices.Contains(mysqltype.ProgressingDeploymentStatuses, dep.Status) {
		return fault.New(
			"deployment not in progress",
			fault.Code(codes.App.Precondition.DeploymentNotInProgress.URN()),
			fault.Internal(fmt.Sprintf("deployment status is %s", dep.Status)),
			fault.Public(fmt.Sprintf("The deployment is not in progress. Its status is %s.", dep.Status)),
		)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = hydrav1.NewDeploymentServiceIngressClient(h.Restate, dep.ID).
		CancelDeployment().
		Send(ctx, &hydrav1.CancelDeploymentRequest{
			Actor:         actor,
			CorrelationId: auditlog.NewCorrelationID(),
		})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment cancellation to Restate"),
			fault.Public("Failed to cancel deployment."),
		)
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
