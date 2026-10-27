package handler

import (
	"cmp"
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

	// A caller without create permission gets 403 whether or not the slugs
	// resolve, so it cannot probe which ones exist
	scope, err := db.Query.ResolveDeploymentScope(ctx, h.DB.RO(), db.ResolveDeploymentScopeParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(err, fault.Internal("failed to resolve environment"))
	}
	notFound := err != nil || !scope.AppID.Valid || !scope.EnvironmentID.Valid

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.CreateDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   scope.EnvironmentID.String,
			Action:       rbac.CreateDeployment,
		}),
		rbac.U(
			// A level that did not resolve becomes "*", so only a wildcard
			// permission can reach the 404
			urn.New().Workspace(principal.AuthorizedWorkspaceID).
				Project(cmp.Or(scope.ProjectID, "*")).
				App(cmp.Or(scope.AppID.String, "*")).
				Environment(cmp.Or(scope.EnvironmentID.String, "*")).
				Deployment("*"),
			permissions.Write,
		),
	))
	if err != nil {
		return err
	}
	if notFound {
		return fault.New(
			"environment not found",
			fault.Code(codes.Data.Environment.NotFound.URN()),
			fault.Internal("project, app, or environment did not resolve"),
			fault.Public("The requested project, app, or environment does not exist."),
		)
	}

	environmentID := scope.EnvironmentID.String

	actorInfo, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	createReq := &hydrav1.DeployCreateRequest{
		ProjectId:     scope.ProjectID,
		AppId:         scope.AppID.String,
		EnvironmentId: environmentID,
		Decision:      hydrav1.CreateDecision_CREATE_DECISION_DEPLOY,
		Trigger:       &hydrav1.Trigger{Source: deployment.TriggerFromClient(s), Actor: actorInfo, Reason: ""},
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
		if err := deployment.RequireSourceInEnvironment(ctx, h.DB, principal.AuthorizedWorkspaceID, scope.AppID.String, environmentID, req.Deployment.DeploymentId); err != nil {
			return err
		}
		createReq.Source = &hydrav1.DeployCreateRequest_ExistingDeployment{
			ExistingDeployment: &hydrav1.CreateExistingDeploymentSource{
				DeploymentId:  req.Deployment.DeploymentId,
				RequireLatest: false,
			},
		}
	}

	deploymentID, err := deployment.Create(ctx, h.Restate, createReq)
	if err != nil {
		return err
	}

	return s.JSON(http.StatusCreated, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V3DeploymentsCreateDeploymentResponseData{DeploymentId: deploymentID},
	})
}

func hasValue(value *string) bool {
	return value != nil && strings.TrimSpace(*value) != ""
}
