package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsListRolesRequestBody
type Response = openapi.V2PermissionsListRolesResponseBody

// Handler implements zen.Route interface for the v2 permissions list roles endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
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
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Handle null cursor - use empty string to start from beginning
	cursor := ptr.SafeDeref(req.Cursor, "")

	// 3. Permission check
	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Rbac,
				ResourceID:   "*",
				Action:       rbac.ReadRole,
			}),
		),
	)
	if err != nil {
		return err
	}

	// 4. Query roles with pagination
	roles, err := db.Query.ListRoles(
		ctx,
		h.DB.RO(),
		db.ListRolesParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			IDCursor:    cursor,
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve roles."),
		)
	}

	// Check if we have more results by seeing if we got 101 roles
	hasMore := len(roles) > 100
	var nextCursor *string

	// If we have more than 100, truncate to 100
	if hasMore {
		nextCursor = ptr.P(roles[100].ID)
		roles = roles[:100]
	}

	// 5. Get permissions for each role
	roleResponses := make([]openapi.RoleWithPermissions, 0, len(roles))
	for _, role := range roles {
		// Get permissions for this role
		rolePermissions, err := db.Query.ListPermissionsByRoleID(ctx, h.DB.RO(), role.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve role permissions."),
			)
		}

		// Transform permissions
		permissions := make([]openapi.Permission, 0, len(rolePermissions))
		for _, perm := range rolePermissions {
			permission := openapi.Permission{
				Id:          perm.ID,
				Name:        perm.Name,
				Description: nil,
				WorkspaceId: perm.WorkspaceID,
				CreatedAt:   perm.CreatedAtM,
			}

			// Add description only if it's valid
			if perm.Description.Valid {
				permission.Description = &perm.Description.String
			}

			permissions = append(permissions, permission)
		}

		// Transform role
		roleResponse := openapi.RoleWithPermissions{
			Id:          role.ID,
			Name:        role.Name,
			Description: nil,
			WorkspaceId: role.WorkspaceID,
			CreatedAt:   role.CreatedAtM,
			Permissions: permissions,
		}

		// Add description only if it's valid
		if role.Description.Valid {
			roleResponse.Description = &role.Description.String
		}

		roleResponses = append(roleResponses, roleResponse)
	}

	// 6. Return success response
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
