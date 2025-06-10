package handler

import (
	"context"
	"net/http"
	"time"

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

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.getRole", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.getRole")

		// 1. Authentication
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// 2. Request validation
		req, err := zen.BindBody[Request](s)
		if err != nil {
			return err
		}

		// 3. Permission check
		err = svc.Permissions.Check(
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
		role, err := db.Query.FindRoleByID(ctx, svc.DB.RO(), req.RoleId)
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
		rolePermissions, err := db.Query.ListPermissionsByRoleID(ctx, svc.DB.RO(), req.RoleId)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve role permissions."),
			)
		}

		// 7. Transform permissions to the response format
		permissions := make([]openapi.Permission, 0, len(rolePermissions))
		for _, perm := range rolePermissions {
			permCreatedAt := time.UnixMilli(perm.CreatedAtM).UTC()
			permission := openapi.Permission{
				Id:          perm.ID,
				Name:        perm.Name,
				WorkspaceId: perm.WorkspaceID,
				CreatedAt:   &permCreatedAt,
			}

			// Add description only if it's valid
			if perm.Description.Valid {
				permission.Description = &perm.Description.String
			}

			permissions = append(permissions, permission)
		}

		// 8. Return the role with its permissions
		roleCreatedAt := time.UnixMilli(role.CreatedAtM).UTC()
		roleResponse := openapi.RoleWithPermissions{
			Id:          role.ID,
			Name:        role.Name,
			WorkspaceId: role.WorkspaceID,
			CreatedAt:   &roleCreatedAt,
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
	})
}
