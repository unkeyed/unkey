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
	"github.com/unkeyed/unkey/svc/api/internal/ctrlclient"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ProjectsDeleteProjectRequestBody
	Response = openapi.V2ProjectsDeleteProjectResponseBody
)

type Handler struct {
	DB         db.Database
	CtrlClient ctrl.ProjectServiceClient
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

	project, err := db.Query.FindProjectByWorkspaceAndSlug(ctx, h.DB.RO(), db.FindProjectByWorkspaceAndSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Slug:        req.Slug,
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
	// deployments via a durable Restate workflow. The control plane enqueues
	// the workflow and returns immediately; teardown is eventually consistent.
	_, err = h.CtrlClient.DeleteProject(ctx, &ctrlv1.DeleteProjectRequest{
		ProjectId: project.ID,
		Actor:     actor,
	})
	if err != nil {
		return ctrlclient.HandleError(err, "delete project")
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
