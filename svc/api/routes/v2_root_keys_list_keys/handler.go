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
	permissionsByKey := make(map[string][]string, len(rows))
	if len(rows) > 0 {
		ids := make([]string, 0, len(rows))
		for _, row := range rows {
			ids = append(ids, row.ID)
			permissionsByKey[row.ID] = []string{}
		}
		storedPermissions, err := db.Query.ListRootKeyPermissions(ctx, h.DB.RO(), db.ListRootKeyPermissionsParams{
			WorkspaceID: p.AuthorizedWorkspaceID,
			KeyIds:      ids,
		})
		if err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve root key permissions."))
		}
		for _, permission := range storedPermissions {
			permissionsByKey[permission.KeyID] = append(permissionsByKey[permission.KeyID], permission.Slug)
		}
	}
	data := make([]openapi.V2RootKeysListKeysResponseData, 0, len(rows))
	for _, row := range rows {
		slices.Sort(permissionsByKey[row.ID])
		name := nullable.NewNullNullable[string]()
		if row.Name.Valid {
			name = nullable.NewNullableWithValue(row.Name.String)
		}
		expires := nullable.NewNullNullable[int64]()
		if row.Expires.Valid {
			expires = nullable.NewNullableWithValue(row.Expires.Time.UnixMilli())
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
			Expires:     expires,
			Permissions: slices.Compact(permissionsByKey[row.ID]),
		})
	}
	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pg,
	})
}
