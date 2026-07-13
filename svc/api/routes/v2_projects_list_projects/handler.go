package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ProjectsListProjectsRequestBody
	Response = openapi.V2ProjectsListProjectsResponseBody
)

// Handler implements zen.Route interface for the v2 projects list projects endpoint
type Handler struct {
	DB db.Database
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/projects.listProjects"
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

	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Project,
		ResourceID:   "*",
		Action:       rbac.ReadProject,
	}))
	if err != nil {
		return err
	}

	p := pagination.Parse(req.Limit, req.Cursor, 100)
	search := mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))

	rows, err := db.Query.ListProjectsByWorkspaceId(ctx, h.DB.RO(), db.ListProjectsByWorkspaceIdParams{
		WorkspaceID: principal.WorkspaceID,
		IDCursor:    p.Cursor,
		Search:      search,
		Limit:       p.FetchLimit(),
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve projects."),
		)
	}

	rows, pg := pagination.Paginate(rows, p, func(r db.ListProjectsByWorkspaceIdRow) string { return r.ID })

	data := array.Map(rows, func(row db.ListProjectsByWorkspaceIdRow) openapi.Project {
		return openapi.Project{
			Id:               row.ID,
			Name:             row.Name,
			Slug:             row.Slug,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt.Int64,
			DeleteProtection: row.DeleteProtection.Bool,
		}
	})

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       data,
		Pagination: pg,
	})
}
