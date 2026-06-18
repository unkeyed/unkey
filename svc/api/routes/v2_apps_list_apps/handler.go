package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsListAppsRequestBody
	Response = openapi.V2AppsListAppsResponseBody
)

// Handler implements zen.Route interface for the v2 apps list apps endpoint
type Handler struct {
	DB db.Database
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/apps.listApps"
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

	project, err := db.Query.FindProjectByWorkspaceAndId(ctx, h.DB.RO(), db.FindProjectByWorkspaceAndIdParams{
		WorkspaceID: principal.WorkspaceID,
		ID:          req.ProjectId,
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
			Action:       rbac.ReadApp,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   project.ID,
			Action:       rbac.ReadApp,
		}),
	))
	if err != nil {
		// Mirror the missing-project 404 so an unauthorized key can't probe which
		// project slugs exist or read the project ID from the authorization error.
		return fault.New(
			"project not found",
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking project existence"),
			fault.Public("The requested project does not exist."),
		)
	}

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")

	rows, err := db.Query.ListAppsByProject(ctx, h.DB.RO(), db.ListAppsByProjectParams{
		ProjectID: project.ID,
		IDCursor:  cursor,
		Limit:     int32(limit + 1), // nolint:gosec
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve apps."),
		)
	}

	hasMore := len(rows) > limit
	var nextCursor *string
	if hasMore {
		nextCursor = ptr.P(rows[limit].App.ID)
		rows = rows[:limit]
	}

	data := make([]openapi.App, len(rows))
	for i, row := range rows {
		data[i] = openapi.App{
			Id:                  row.App.ID,
			Name:                row.App.Name,
			Slug:                row.App.Slug,
			ProjectId:           row.App.ProjectID,
			DefaultBranch:       row.App.DefaultBranch,
			CurrentDeploymentId: row.App.CurrentDeploymentID.String,
			IsRolledBack:        row.App.IsRolledBack,
			DeleteProtection:    row.App.DeleteProtection.Bool,
			CreatedAt:           row.App.CreatedAt,
			UpdatedAt:           row.App.UpdatedAt.Int64,
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
		Pagination: &openapi.Pagination{
			Cursor:  nextCursor,
			HasMore: hasMore,
		},
	})
}
