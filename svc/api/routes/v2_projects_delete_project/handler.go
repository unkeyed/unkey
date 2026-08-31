package handler

import (
	"context"
	"net/http"

	restateingress "github.com/restatedev/sdk-go/ingress"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/projectgate"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ProjectsDeleteProjectRequestBody
	Response = openapi.V2ProjectsDeleteProjectResponseBody
)

type Handler struct {
	DB      db.Database
	Restate *restateingress.Client
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/projects.deleteProject"
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

	project, err := db.Query.FindProjectByIdOrSlug(ctx, h.DB.RW(), db.FindProjectByIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"project not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project not found"),
				fault.Public("The requested project does not exist."),
			)
		}

		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve project."),
		)
	}

	if project.Slug == projectgate.DefaultSlug {
		return fault.New(
			"project not found",
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal("default project is not exposed by the projects API"),
			fault.Public("The requested project does not exist."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.DeleteProject,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   project.ID,
			Action:       rbac.DeleteProject,
		}),
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Project(project.ID),
			permissions.Delete{},
		),
	))
	if err != nil {
		return err
	}

	if project.DeleteProtection.Valid && project.DeleteProtection.Bool {
		return fault.New(
			"delete protected",
			fault.Code(codes.App.Protection.ProtectedResource.URN()),
			fault.Internal("project is protected from deletion"),
			fault.Public("This project has delete protection enabled. Disable it before attempting to delete."),
		)
	}

	actor, err := ctrlclient.Actor(s)
	if err != nil {
		return err
	}

	// Deletion cascades through the project's apps, environments, and
	// deployments. Send returns once Restate has accepted the durable workflow;
	// teardown remains eventually consistent.
	_, err = hydrav1.NewProjectServiceIngressClient(h.Restate, project.ID).
		Delete().
		Send(ctx, &hydrav1.DeleteProjectRequest{
			Actor:         actor,
			CorrelationId: auditlog.NewCorrelationID(),
		})
	if err != nil {
		// Restate authentication, service discovery, capacity, and transport
		// failures are all internal. Preserve the cause for operators without
		// exposing infrastructure details or implying the caller can fix them.
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("failed to submit project deletion to Restate"),
			fault.Public("Failed to delete project."),
		)
	}

	return s.JSON(http.StatusAccepted, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
