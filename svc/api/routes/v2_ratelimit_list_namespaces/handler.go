package handler

import (
	"context"
	"errors"
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
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitListNamespacesRequestBody
	Response = openapi.V2RatelimitListNamespacesResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/ratelimit.listNamespaces"
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

	p := pagination.Parse(req.Limit, req.Cursor, 100)
	search := mysql.SearchContains(strings.TrimSpace(ptr.SafeDeref(req.Search)))

	allowed := func(row db.ListRatelimitNamespacesRow) bool {
		resource := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(row.ProjectID).RatelimitNamespace(row.ID)
		return rbac.Check(rbac.U(resource, permissions.Read), principal.Permissions) == nil
	}
	rows, err := pagination.FetchAuthorized(ctx, p, func(ctx context.Context, cursor string, limit int32) ([]db.ListRatelimitNamespacesRow, error) {
		if cursor != "" && cursor == p.Cursor {
			cursorRows, err := db.Query.ListRatelimitNamespaces(ctx, h.DB.RO(), db.ListRatelimitNamespacesParams{
				WorkspaceID: principal.AuthorizedWorkspaceID,
				CursorID:    cursor,
				Limit:       1,
			})
			if err != nil {
				return nil, err
			}
			if len(cursorRows) == 0 || cursorRows[0].ID != cursor || !allowed(cursorRows[0]) {
				return nil, nil
			}
		}
		return db.Query.ListRatelimitNamespaces(ctx, h.DB.RO(), db.ListRatelimitNamespacesParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			CursorID:    cursor,
			Search:      search,
			Limit:       limit,
		})
	}, allowed, func(row db.ListRatelimitNamespacesRow) string { return row.ID })
	if errors.Is(err, pagination.ErrScanLimit) {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("authorized namespace scan limit exceeded"),
			fault.Public("The namespace scan limit was reached. Narrow the search filter and retry."),
		)
	}
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve namespaces."),
		)
	}

	rows, pg := pagination.Paginate(rows, p, func(r db.ListRatelimitNamespacesRow) string { return r.ID })

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: array.Map(rows, func(row db.ListRatelimitNamespacesRow) openapi.RatelimitNamespace {
			return openapi.RatelimitNamespace{
				Id:        row.ID,
				Name:      row.Name,
				CreatedAt: row.CreatedAtM,
				UpdatedAt: row.UpdatedAtM.Int64,
			}
		}),
		Pagination: pg,
	})
}
