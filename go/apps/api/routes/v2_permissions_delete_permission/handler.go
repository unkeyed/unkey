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

type Request = openapi.V2PermissionsDeletePermissionRequestBody
type Response = openapi.V2PermissionsDeletePermissionResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.deletePermission", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.deletePermission")

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
		err = svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Permission,
					ResourceID:   "*",
					Action:       rbac.DeletePermission,
				}),
			),
		)
		if err != nil {
			return err
		}

		// 4. Check if permission exists and belongs to authorized workspace
		permission, err := db.Query.FindPermissionById(ctx, svc.DB.RO(), req.PermissionId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("permission not found",
					fault.Code(codes.Data.Permission.NotFound.URN()),
					fault.Internal("permission not found"), fault.Public("The requested permission does not exist."),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve permission information."),
			)
		}

		// Check if permission belongs to authorized workspace
		if permission.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Internal("permission not found"), fault.Public("The requested permission does not exist."),
			)
		}

		// 5. Delete the permission in a transaction
		err = svc.DB.WithTransaction(ctx, func(ctx context.Context, tx db.Tx) error {
			// Delete role-permission relationships
			_, err := db.Query.DeleteRolePermissionsByPermissionId(ctx, tx, req.PermissionId)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to delete role-permission relationships."),
				)
			}

			// Delete key-permission relationships
			_, err = db.Query.DeleteKeyPermissionsByPermissionId(ctx, tx, req.PermissionId)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to delete key-permission relationships."),
				)
			}

			// Delete the permission itself
			_, err = db.Query.DeletePermission(ctx, tx, req.PermissionId)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to delete permission."),
				)
			}

			// Create audit log for permission deletion
			err = svc.Auditlogs.CreateAuditLog(ctx, tx, auditlogs.CreateAuditLogParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       "permission.delete",
				ActorType:   "key",
				ActorID:     auth.KeyID,
				Description: "Deleted " + req.PermissionId,
				Resources: []auditlogs.Resource{
					{
						Type: "permission",
						ID:   req.PermissionId,
						Meta: map[string]interface{}{
							"name":        permission.Name,
							"description": permission.Description.String,
						},
					},
				},
				Context: auditlogs.Context{
					Location:  s.Location(),
					UserAgent: s.UserAgent(),
				},
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("audit log error"), fault.Public("Failed to create audit log for permission deletion."),
				)
			}

			return nil
		})
		if err != nil {
			return err
		}

		// 6. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
		})
	})
}
