package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2PermissionsListRolesRequestBody
	Response = openapi.V2PermissionsListRolesResponseBody
)

// Handler implements zen.Route interface for the v2 permissions list roles endpoint
type Handler struct {
	// Services as public fields
	Logger logging.Logger
	DB     db.Database
	Keys   keys.KeyService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/permissions.listRoles"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.listRoles")

	// 1. Authentication
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	cursor := ptr.SafeDeref(req.Cursor, "")
	limit := ptr.SafeDeref(req.Limit, 100)

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.ReadRole,
		}),
	)))
	if err != nil {
		return err
	}

	roles, err := db.Query.ListRoles(
		ctx,
		h.DB.RO(),
		db.ListRolesParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			IDCursor:    cursor,
			Limit:       int32(limit) + 1,
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve roles."),
		)
	}

	var nextCursor *string
	hasMore := len(roles) > limit
	if hasMore {
		nextCursor = ptr.P(roles[limit].ID)
		roles = roles[:limit]
	}

	roleResponses := make([]openapi.Role, 0, len(roles))
	for _, role := range roles {
		roleResponse := openapi.Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: nil,
			Permissions: nil,
		}

		if role.Description.Valid {
			roleResponse.Description = &role.Description.String
		}

		rolePermissions := make([]db.Permission, 0)
		if permBytes, ok := role.Permissions.([]byte); ok && permBytes != nil {
			//nolint:musttag
			_ = json.Unmarshal(permBytes, &rolePermissions) // Ignore error, default to empty array
		}
		perms := make([]openapi.Permission, 0)

		for _, perm := range rolePermissions {
			permission := openapi.Permission{
				Id:          perm.ID,
				Name:        perm.Name,
				Slug:        perm.Slug,
				Description: nil,
			}

			if perm.Description.Valid {
				permission.Description = &perm.Description.String
			}

			perms = append(perms, permission)
		}

		if len(perms) > 0 {
			roleResponse.Permissions = ptr.P(perms)
		}

		roleResponses = append(roleResponses, roleResponse)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: roleResponses,
		Pagination: &openapi.Pagination{
			Cursor:  nextCursor,
			HasMore: hasMore,
		},
	})
}
