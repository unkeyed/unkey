package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeployGetDeploymentRequestBody
	Response = openapi.V2DeployGetDeploymentResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Path() string {
	return "/v2/deploy.getDeployment"
}

func (h *Handler) Method() string {
	return "POST"
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

	deployment, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), req.DeploymentId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("deployment not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("deployment not found"),
				fault.Public("The requested deployment does not exist or has been deleted."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to find deployment"))
	}

	// Verify deployment belongs to the authenticated workspace
	if deployment.WorkspaceID != principal.AuthorizedWorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"),
			fault.Public("The requested deployment does not exist or has been deleted."),
		)
	}

	// Extract projectID from deployment
	projectID := deployment.ProjectID

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.ReadDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   projectID,
			Action:       rbac.ReadDeployment,
		}),
	))
	if err != nil {
		return err
	}

	// Build response directly from database model
	responseData := openapi.V2DeployGetDeploymentResponseData{
		Id:           deployment.ID,
		Status:       dbStatusToOpenAPI(deployment.Status),
		Steps:        nil,
		ErrorMessage: nil,
		Hostnames:    nil,
	}

	// Fetch hostnames from frontline routes
	routes, routesErr := db.Query.FindFrontlineRoutesByDeploymentID(ctx, h.DB.RO(), req.DeploymentId)
	if routesErr != nil {
		logger.Warn("failed to fetch frontline routes for deployment", "error", routesErr, "deployment_id", deployment.ID)
	} else if len(routes) > 0 {
		hostnames := make([]string, len(routes))
		for i, route := range routes {
			hostnames[i] = route.FullyQualifiedDomainName
		}
		responseData.Hostnames = &hostnames
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}

func dbStatusToOpenAPI(status mysqltype.DeploymentsStatus) openapi.V2DeployGetDeploymentResponseDataStatus {
	switch status {
	case mysqltype.DeploymentsStatusPending:
		return openapi.PENDING
	case mysqltype.DeploymentsStatusStarting:
		return openapi.STARTING
	case mysqltype.DeploymentsStatusBuilding:
		return openapi.BUILDING
	case mysqltype.DeploymentsStatusDeploying:
		return openapi.DEPLOYING
	case mysqltype.DeploymentsStatusNetwork:
		return openapi.NETWORK
	case mysqltype.DeploymentsStatusFinalizing:
		return openapi.FINALIZING
	case mysqltype.DeploymentsStatusReady:
		return openapi.READY
	case mysqltype.DeploymentsStatusFailed:
		return openapi.FAILED
	case mysqltype.DeploymentsStatusSkipped:
		return openapi.SKIPPED
	case mysqltype.DeploymentsStatusAwaitingApproval:
		return openapi.AWAITINGAPPROVAL
	case mysqltype.DeploymentsStatusStopped:
		return openapi.STOPPED
	case mysqltype.DeploymentsStatusSuperseded:
		return openapi.SUPERSEDED
	case mysqltype.DeploymentsStatusCancelled:
		return openapi.CANCELLED
	default:
		return openapi.UNSPECIFIED
	}
}
