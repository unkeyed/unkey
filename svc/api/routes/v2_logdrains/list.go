package logdrains

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type List struct{ DB db.Database }

func (h *List) Method() string { return http.MethodPost }
func (h *List) Path() string   { return "/v2/logdrains.listLogdrains" }
func (h *List) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.ListLogdrainsRequest](s)
	if err != nil {
		return err
	}
	if err := principal.Authorize(rbac.U(urn.V1{WorkspaceID: principal.AuthorizedWorkspaceID, Resource: "logdrains/*"}, permissions.Read)); err != nil {
		return err
	}
	limit := ptr.SafeDeref(req.Limit, 100)
	rows, err := db.Query.ListLogdrains(ctx, h.DB.RO(), db.ListLogdrainsParams{WorkspaceID: principal.AuthorizedWorkspaceID, AfterID: ptr.SafeDeref(req.Cursor), Limit: int32(limit + 1)})
	if err != nil {
		return err
	}
	response := openapi.ListLogdrainsResponse{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: []openapi.Logdrain{}, Pagination: openapi.Pagination{Cursor: nil, HasMore: false}}
	response.Pagination.HasMore = len(rows) > limit
	if response.Pagination.HasMore {
		rows = rows[:limit]
	}
	for _, row := range rows {
		data, err := toPublic(row)
		if err != nil {
			return err
		}
		response.Data = append(response.Data, data)
	}
	if response.Pagination.HasMore {
		response.Pagination.Cursor = &rows[len(rows)-1].ID
	}
	return s.JSON(http.StatusOK, response)
}
