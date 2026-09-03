package handler

import (
	"context"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deploygate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsPromoteDeploymentRequestBody
	Response = openapi.V2DeploymentsPromoteDeploymentResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
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
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(dep.ProjectID).App(dep.AppID).Environment(dep.EnvironmentID),
			permissions.Write,
		),
	))
	if err != nil {
		return fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking deployment existence"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	// The current live pointer decides promote eligibility. See
	// deploygate.CheckPromoteTarget for the invariant the API, ctrl service, and
	// worker all enforce.
	app, err := db.Query.FindAppById(ctx, h.DB.RW(), dep.AppID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to load app for promotion eligibility"),
			fault.Public("Failed to resolve the current live deployment."),
		)
	}

	if err := deploygate.CheckPromoteTarget(deploygate.PromoteInput{
		Status:              dep.Status,
		DesiredState:        dep.DesiredState,
		EnvironmentKind:     dep.EnvironmentKind,
		CurrentDeploymentID: app.CurrentDeploymentID.String,
		DeploymentID:        dep.ID,
		IsRolledBack:        app.IsRolledBack,
	}); err != nil {
		return err
	}

	if err := deployment.EnsureWorkspaceCanDeploy(ctx, h.DB, principal.WorkspaceID); err != nil {
		return err
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = hydrav1.NewDeployServiceIngressClient(h.Restate, dep.ID).
		Promote().
		Send(ctx, &hydrav1.PromoteRequest{
			TargetDeploymentId: dep.ID,
			Actor:              actor,
			CorrelationId:      auditlog.NewCorrelationID(),
		})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment promotion to Restate"),
			fault.Public("Failed to promote deployment."),
		)
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
