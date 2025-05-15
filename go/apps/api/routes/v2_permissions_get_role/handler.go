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
		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		// 3. Permission check
		permissionCheck, err := svc.Permissions.Check(
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
			return fault.Wrap(err,
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissionCheck.Valid {
			return fault.New("insufficient permissions",
				fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.WithDesc(permissionCheck.Message, permissionCheck.Message),
			)
		}

		// 4. Get role by ID
		role, err := db.Query.FindRoleById(ctx, svc.DB.RO(), req.RoleId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("role not found",
					fault.WithCode(codes.Data.Role.NotFound.URN()),
					fault.WithDesc("role not found", "The requested role does not exist."),
				)
			}
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to retrieve role information."),
			)
		}

		// 5. Check if role belongs to authorized workspace
		if role.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("role not found",
				fault.WithCode(codes.Data.Role.NotFound.URN()),
				fault.WithDesc("role not found", "The requested role does not exist."),
			)
		}

		// 6. Fetch permissions associated with the role
		rolePermissions, err := db.Query.FindPermissionsByRoleId(ctx, svc.DB.RO(), req.RoleId)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to retrieve role permissions."),
			)
		}

		// 7. Transform permissions to the response format
		permissions := make([]openapi.Permission, 0, len(rolePermissions))
		for _, perm := range rolePermissions {
			permissions = append(permissions, openapi.Permission{
				Id:          perm.ID,
				Name:        perm.Name,
				WorkspaceId: perm.WorkspaceID,
				Description: perm.Description.String,
				CreatedAt:   perm.CreatedAtM.Time.Format(http.TimeFormat),
			})
		}

		// 8. Return the role with its permissions
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.PermissionsGetRoleResponseData{
				Role: openapi.RoleWithPermissions{
					Id:          role.ID,
					Name:        role.Name,
					WorkspaceId: role.WorkspaceID,
					Description: role.Description.String,
					CreatedAt:   role.CreatedAtM.Time.Format(http.TimeFormat),
					Permissions: permissions,
				},
			},
		})
	})
}
