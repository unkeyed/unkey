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
	Request  = openapi.V2DeploymentsStartDeploymentRequestBody
	Response = openapi.V2DeploymentsStartDeploymentResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
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
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(dep.ProjectID).App(dep.AppID).Environment(dep.EnvironmentID).Deployment(dep.ID),
			permissions.StartDeployment{},
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

	// A missing billing row means the workspace was never linked to billing and
	// is not suspended.
	// LoadBilling rather than EnsureWorkspaceCanDeploy: the spend check belongs
	// after the lifecycle gate below, so a deployment that cannot be started at
	// all reports that rather than a billing reason.
	billing, err := deployment.LoadBilling(ctx, h.DB, principal.WorkspaceID)
	if err != nil {
		return err
	}

	if err := deploygate.CheckWorkspacePlan(billing.Plan, billing.PlanOverride); err != nil {
		return err
	}

	// "Stopped" is keyed on desired_state, not status: stopping sets
	// desired_state=stopped immediately while status only flips once krane drains
	// the last instance.
	if err := deploygate.CheckStartTarget(deploygate.StartInput{
		DesiredState:    dep.DesiredState,
		EnvironmentKind: dep.EnvironmentKind,
		SpendSuspended:  billing.SpendSuspended,
	}); err != nil {
		return err
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	_, err = hydrav1.NewDeployServiceIngressClient(h.Restate, dep.ID).
		WakeDeployment().
		Send(ctx, &hydrav1.WakeDeploymentRequest{
			DeploymentId:  dep.ID,
			Actor:         actor,
			CorrelationId: auditlog.NewCorrelationID(),
		})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment start to Restate"),
			fault.Public("Failed to start deployment."),
		)
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
