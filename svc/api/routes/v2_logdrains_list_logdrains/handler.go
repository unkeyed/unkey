package logdrains

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/logdrainconfig"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Handler struct{ DB db.Database }

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/logdrains.listLogdrains" }
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[openapi.ListLogdrainsRequest](s)
	if err != nil {
		return err
	}
	p := pagination.Parse(req.Limit, req.Cursor, 100)
	rows, err := pagination.FetchAuthorized(ctx, p, func(ctx context.Context, cursor string, limit int32) ([]db.Logdrain, error) {
		return db.Query.ListLogdrains(ctx, h.DB.RO(), db.ListLogdrainsParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			IDCursor:    cursor,
			Limit:       limit,
		})
	}, func(row db.Logdrain) bool {
		return rbac.Check(rbac.U(urn.New().Workspace(row.WorkspaceID).Logdrain(row.ID), permissions.Read), principal.Permissions) == nil
	}, func(row db.Logdrain) string { return row.ID })
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve log drains."))
	}
	rows, pg := pagination.Paginate(rows, p, func(row db.Logdrain) string { return row.ID })
	data, err := array.MapErr(rows, logdrainconfig.ToPublic)
	if err != nil {
		return err
	}
	response := openapi.ListLogdrainsResponse{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pg,
	}
	return s.JSON(http.StatusOK, response)
}
