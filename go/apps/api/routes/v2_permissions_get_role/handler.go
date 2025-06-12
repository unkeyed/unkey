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
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsGetRoleRequestBody
type Response = openapi.V2PermissionsGetRoleResponseBody

// Handler implements zen.Route interface for the v2 permissions get role endpoint
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
	return "/v2/permissions.getRole"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.getRole")

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

	// 4. Get role by ID
	role, err := db.Query.FindRoleByID(ctx, h.DB.RO(), req.RoleId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role not found"), fault.Public("The requested role does not exist."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve role information."),
		)
	}

	// 5. Check if role belongs to authorized workspace
	if role.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("role does not belong to authorized workspace",
			fault.Code(codes.Data.Role.NotFound.URN()),
			fault.Public("The requested role does not exist."),
		)
	}

	// 6. Fetch permissions associated with the role
	rolePermissions, err := db.Query.ListPermissionsByRoleID(ctx, h.DB.RO(), req.RoleId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve role permissions."),
		)
	}

	// 7. Transform permissions to the response format
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

	// 8. Return the role with its permissions
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

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.PermissionsGetRoleResponseData{
			Role: roleResponse,
		},
	})
}
