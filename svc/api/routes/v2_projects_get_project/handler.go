package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ProjectsGetProjectRequestBody
	Response = openapi.V2ProjectsGetProjectResponseBody
)

// Handler implements zen.Route interface for the v2 projects get project endpoint
type Handler struct {
	DB db.Database
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/projects.getProject"
}

// Handle processes the HTTP request
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
			Action:       rbac.ReadProject,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   project.ID,
			Action:       rbac.ReadProject,
		}),
	))
	if err != nil {
		// Mirror the missing-slug 404 so an unauthorized key can't probe which
		// slugs exist or read the project ID from the authorization error.
		return fault.New(
			"project not found",
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking project existence"),
			fault.Public("The requested project does not exist."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.Project{
			Id:               project.ID,
			Name:             project.Name,
			Slug:             project.Slug,
			CreatedAt:        project.CreatedAt,
			UpdatedAt:        project.UpdatedAt.Int64,
			DeleteProtection: project.DeleteProtection.Bool,
		},
	})
}
