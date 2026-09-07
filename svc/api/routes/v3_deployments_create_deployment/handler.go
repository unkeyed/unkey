package handler

import (
	"context"
	"net/http"
	"strings"

	restateingress "github.com/restatedev/sdk-go/ingress"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V3DeploymentsCreateDeploymentRequestBody
	Response = openapi.V3DeploymentsCreateDeploymentResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
}

func (h *Handler) Path() string {
	return "/v3/deployments.createDeployment"
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
	sourceCount := 0
	if req.Git != nil {
		sourceCount++
	}
	if req.Oci != nil {
		sourceCount++
	}
	if req.Deployment != nil {
		sourceCount++
	}
	if sourceCount > 1 {
		return fault.New(
			"multiple deployment sources provided",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("request contains more than one source override"),
			fault.Public("Provide at most one of git, oci, or deployment."),
		)
	}

	environment, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("project, app, or environment did not resolve"),
				fault.Public("The requested project, app, or environment does not exist."),
			)
		}
		return fault.Wrap(err, fault.Internal("failed to resolve environment"))
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.CreateDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   environment.ID,
			Action:       rbac.CreateDeployment,
		}),
		rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(environment.ProjectID).App(environment.AppID).Environment(environment.ID).Deployment("*"),
			permissions.Write,
		),
	))
	if err != nil {
		return err
	}

	client := s.Request().Header.Get("X-Unkey-Client")
	trigger := ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_API
	switch {
	case strings.HasPrefix(client, "unkey-cli/"):
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_CLI
	case strings.HasPrefix(client, "unkey-dashboard"):
		trigger = ctrlv1.DeploymentTrigger_DEPLOYMENT_TRIGGER_DASHBOARD
	}

	actorInfo, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	// The id is the Restate object key the create runs on, so minting it here
	// lets the response name the deployment without waiting on the worker.
	deploymentID := uid.New(uid.DeploymentPrefix)

	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:     environment.ProjectID,
		AppId:         environment.AppID,
		EnvironmentId: environment.ID,
		Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Trigger:       trigger,
		TriggeredBy:   principal.Subject.ID,
		TriggerReason: "",
		Actor:         actorInfo,
	}

	// No source leaves the oneof unset: the worker then deploys what the app
	// declares
	switch {
	case req.Oci != nil:
		createReq.Source = &hydrav1.DeployCreateRequest_Image{
			Image: &hydrav1.CreateImageSource{Image: req.Oci.Image},
		}

	case req.Git != nil:
		git := req.Git
		if hasValue(git.Repository) && !hasValue(git.CommitSha) {
			return fault.New(
				"repository requires commitSha",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("repository set without commitSha"),
				fault.Public("repository requires commitSha."),
			)
		}
		createReq.Source = &hydrav1.DeployCreateRequest_Git{
			// nolint: exhaustruct // the worker fills the commit metadata it resolves from git
			Git: &hydrav1.CreateGitSource{
				Commit: &ctrlv1.GitCommitInfo{
					Branch:         ptr.SafeDeref(git.Branch),
					CommitSha:      ptr.SafeDeref(git.CommitSha),
					ForkRepository: ptr.SafeDeref(git.Repository),
				},
				PrNumber: 0,
			},
		}

	case req.Deployment != nil:
		if err := h.requireRedeployableSource(ctx, principal.AuthorizedWorkspaceID, environment.AppID, environment.ID, req.Deployment.DeploymentId); err != nil {
			return err
		}
		createReq.Source = &hydrav1.DeployCreateRequest_ExistingDeployment{
			ExistingDeployment: &hydrav1.CreateExistingDeploymentSource{
				DeploymentId:   req.Deployment.DeploymentId,
				RequireNoNewer: false,
			},
		}
	}

	res, err := hydrav1.NewDeployServiceIngressClient(h.Restate, deploymentID).
		Create().
		Request(ctx, createReq)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit deployment create to Restate"),
			fault.Public("Failed to create deployment."),
		)
	}
	if err := deployment.OutcomeFault(res.GetOutcome()); err != nil {
		return err
	}

	return s.JSON(http.StatusCreated, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V3DeploymentsCreateDeploymentResponseData{DeploymentId: deploymentID},
	})
}

// requireRedeployableSource keeps the source deployment's ownership check in the
// API so a caller cannot learn whether a deployment it cannot read exists
func (h *Handler) requireRedeployableSource(ctx context.Context, workspaceID, appID, environmentID, deploymentID string) error {
	deployment, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), deploymentID)
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to find deployment"))
	}

	if db.IsNotFound(err) ||
		deployment.WorkspaceID != workspaceID ||
		deployment.AppID != appID ||
		deployment.EnvironmentID != environmentID {
		return fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("deployment does not exist or does not match this workspace, app, and environment"),
			fault.Public("The specified deployment does not exist."),
		)
	}

	return nil
}

func hasValue(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}
