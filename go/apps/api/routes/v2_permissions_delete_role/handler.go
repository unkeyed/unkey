package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsDeleteRoleRequestBody
type Response = openapi.V2PermissionsDeleteRoleResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.deleteRole", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.deleteRole")

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
				fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
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
					Action:       rbac.DeleteRole,
				}),
			),
		)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("unable to check permissions"), fault.Public("We're unable to check the permissions of your key."),
			)
		}

		if !permissionCheck.Valid {
			return fault.New("insufficient permissions",
				fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.Internal(permissionCheck.Message), fault.Public(permissionCheck.Message),
			)
		}

		// 4. Get role by ID to verify existence and workspace ownership
		role, err := db.Query.FindRoleById(ctx, svc.DB.RO(), req.RoleId)
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
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role not found"), fault.Public("The requested role does not exist."),
			)
		}

		// 6. Start transaction for deletion
		err = svc.DB.WithTx(ctx, func(tx db.Tx) error {
			// 6.1 Delete role_permissions
			_, err := db.Query.DeleteRolePermissionsByRoleId(ctx, tx, req.RoleId)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to delete role permissions."),
				)
			}

			// 6.2 Delete key_roles
			_, err = db.Query.DeleteKeyRolesByRoleId(ctx, tx, req.RoleId)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to delete key roles."),
				)
			}

			// 6.3 Delete the role
			_, err = db.Query.DeleteRoleById(ctx, tx, req.RoleId)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to delete the role."),
				)
			}

			return nil
		})

		if err != nil {
			return err
		}

		// 7. Create audit log for role deletion
		_, err = svc.Auditlogs.Create(ctx, auditlogs.CreateParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			ActorID:     auth.KeyID,
			Action:      "role.delete",
			Resources: map[string]string{
				"roleId": req.RoleId,
			},
			Metadata: map[string]interface{}{
				"roleName":        role.Name,
				"roleDescription": role.Description.String,
			},
		})
		if err != nil {
			svc.Logger.Error("failed to create audit log", "error", err)
			// We don't fail the request on audit log failure
		}

		// 8. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
		})
	})
}
