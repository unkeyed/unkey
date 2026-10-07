package handler

import (
	"context"
	"net/http"
	"slices"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2RootKeysListKeysRequestBody
type Response = openapi.V2RootKeysListKeysResponseBody

// Handler lists readable root keys from the new credential store.
type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string { return http.MethodPost }

func (h *Handler) Path() string { return "/v2/rootKeys.listKeys" }

// Handle filters keys before pagination so denied keys never appear in cursors.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	p, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	params := pagination.Parse(req.Limit, req.Cursor, 100)
	rows, err := pagination.FetchAuthorized(ctx, params, func(ctx context.Context, cursor string, limit int32) ([]db.ListRootKeysRow, error) {
		return db.Query.ListRootKeys(ctx, h.DB.RO(), db.ListRootKeysParams{
			WorkspaceID: p.AuthorizedWorkspaceID,
			IDCursor:    cursor,
			Limit:       limit,
		})
	}, func(row db.ListRootKeysRow) bool {
		return rbac.Check(rbac.U(urn.New().Workspace(p.AuthorizedWorkspaceID).RootKey(row.ID), permissions.Read), p.Permissions) == nil
	}, func(row db.ListRootKeysRow) string { return row.ID })
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve root keys."))
	}
	rows, pg := pagination.Paginate(rows, params, func(row db.ListRootKeysRow) string { return row.ID })
	data := make([]openapi.V2RootKeysListKeysResponseData, 0, len(rows))
	for _, row := range rows {
		rootKeyPermissions, err := db.UnmarshalNullableJSONTo[[]string](row.Permissions)
		if err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve root key permissions."))
		}
		slices.Sort(rootKeyPermissions)
		name := nullable.NewNullNullable[string]()
		if row.Name.Valid {
			name = nullable.NewNullableWithValue(row.Name.String)
		}
		expires := nullable.NewNullNullable[int64]()
		if row.Expires.Valid {
			expires = nullable.NewNullableWithValue(row.Expires.Int64)
		}
		start := row.Start
		if row.Prefix != "" {
			start = row.Prefix + "_" + start
		}
		data = append(data, openapi.V2RootKeysListKeysResponseData{
			KeyId:       row.ID,
			Name:        name,
			Start:       start,
			End:         row.End,
			Enabled:     row.Enabled,
			CreatedAt:   row.CreatedAt,
			LastUsedAt:  int64(row.LastUsedAt),
			Expires:     expires,
			Permissions: slices.Compact(rootKeyPermissions),
		})
	}
	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pg,
	})
}
