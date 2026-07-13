package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/mysql"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
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
	// 1. Authentication
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	p := pagination.Parse(req.Limit, req.Cursor, 100)
	search := mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.ReadPermission,
		}),
	))
	if err != nil {
		return err
	}

	permissions, err := db.Query.ListPermissions(
		ctx,
		h.DB.RO(),
		db.ListPermissionsParams{
			WorkspaceID:       principal.WorkspaceID,
			IDCursor:          p.Cursor,
			Search:            search,
			DescriptionSearch: dbtype.NullString(search),
			Limit:             p.FetchLimit(),
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve permissions."),
		)
	}

	permissions, pg := pagination.Paginate(permissions, p, func(r db.Permission) string { return r.ID })

	responsePermissions := array.Map(permissions, func(perm db.Permission) openapi.Permission {
		return openapi.Permission{
			Id:          perm.ID,
			Name:        perm.Name,
			Slug:        perm.Slug,
			Description: perm.Description.String,
		}
	})

	// 7. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       responsePermissions,
		Pagination: pg,
	})
}
