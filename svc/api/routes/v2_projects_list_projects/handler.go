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

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")

	rows, err := db.Query.ListProjectsByWorkspaceId(ctx, h.DB.RO(), db.ListProjectsByWorkspaceIdParams{
		WorkspaceID: principal.WorkspaceID,
		IDCursor:    cursor,
		Limit:       int32(limit + 1), // nolint:gosec
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve projects."),
		)
	}

	hasMore := len(rows) > limit
	var nextCursor *string
	if hasMore {
		nextCursor = ptr.P(rows[limit].ID)
		rows = rows[:limit]
	}

	data := make([]openapi.V2ProjectsListProjectsResponseData, len(rows))
	for i, row := range rows {
		data[i] = openapi.V2ProjectsListProjectsResponseData{
			Id:               row.ID,
			Name:             row.Name,
			Slug:             row.Slug,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt.Int64,
			DeleteProtection: row.DeleteProtection.Bool,
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
