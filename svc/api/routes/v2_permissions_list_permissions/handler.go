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
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PermissionsListPermissionsRequestBody
	Response = openapi.V2PermissionsListPermissionsResponseBody
)

// Handler implements zen.Route interface for the v2 permissions list permissions endpoint
type Handler struct {
	DB db.Database
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/permissions.listPermissions"
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

	p := pagination.Parse(req.Limit, req.Cursor, 100)
	search := mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))

	projectID, err := projects.EnsureDefaultProject(ctx, h.DB.RW(), principal.WorkspaceID)
	if err != nil {
		return err
	}

	rows, err := db.Query.ListPermissions(
		ctx,
		h.DB.RO(),
		db.ListPermissionsParams{
			WorkspaceID:       principal.WorkspaceID,
			ProjectID:         projectID,
			IDCursor:          p.Cursor,
			Search:            search,
			DescriptionSearch: search,
			Limit:             p.FetchLimit(),
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve permissions."),
		)
	}

	readPermissions := rbac.U(
		urn.New().Workspace(principal.WorkspaceID).Project(projectID).RBAC().Permission("*"),
		permissions.Read{},
	)
	err = principal.Authorize(rbac.Or(
		readPermissions,
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.ReadPermission,
		}),
	))
	if err != nil {
		return err
	}

	rows, pg := pagination.Paginate(rows, p, func(r db.Permission) string { return r.ID })

	responsePermissions := array.Map(rows, func(perm db.Permission) openapi.Permission {
		return openapi.Permission{
			Id:          perm.ID,
			Name:        perm.Name,
			Slug:        perm.Slug,
			Description: perm.Description.String,
		}
	})

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       responsePermissions,
		Pagination: pg,
	})
}
