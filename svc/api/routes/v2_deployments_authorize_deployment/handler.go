package handler

import (
	"context"
	"fmt"
	"net/http"

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
	Request  = openapi.V2DeploymentsAuthorizeDeploymentRequestBody
	Response = openapi.V2DeploymentsAuthorizeDeploymentResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.authorizeDeployment"
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

	permissionQuery := rbac.U(
		urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(dep.ProjectID).App(dep.AppID).Environment(dep.EnvironmentID).Deployment(dep.ID),
		permissions.Write,
	)
	err = principal.Authorize(permissionQuery)
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			principal.Permissions,
			permissionQuery,
			codes.Data.Deployment.NotFound.URN(),
			"The requested deployment does not exist.",
		)
	}

	if dep.Status != mysqltype.DeploymentsStatusAwaitingApproval {
		return fault.New(
			"deployment not awaiting approval",
			fault.Code(codes.App.Precondition.DeploymentNotAwaitingApproval.URN()),
			fault.Internal(fmt.Sprintf("deployment status is %s", dep.Status)),
			fault.Public(fmt.Sprintf("The deployment is not awaiting approval. Its status is %s.", dep.Status)),
		)
	}

	if err := deployment.EnsureWorkspaceCanDeploy(ctx, h.DB, principal.AuthorizedWorkspaceID); err != nil {
		return err
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = hydrav1.NewDeploymentServiceIngressClient(h.Restate, dep.ID).
		AuthorizeDeployment().
		Send(ctx, &hydrav1.AuthorizeDeploymentRequest{
			Actor:         actor,
			CorrelationId: auditlog.NewCorrelationID(),
		})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment authorization to Restate"),
			fault.Public("Failed to authorize deployment."),
		)
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
