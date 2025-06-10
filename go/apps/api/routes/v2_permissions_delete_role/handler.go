package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
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
					Action:       rbac.DeleteRole,
				}),
			),
		)
		if err != nil {
			return err
		}

		// 4. Get role by ID to verify existence and workspace ownership
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
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role not found"), fault.Public("The requested role does not exist."),
			)
		}

		// 6. Delete the role in a transaction
		tx, err := svc.DB.RW().Begin(ctx)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to create transaction"), fault.Public("Unable to start database transaction."),
			)
		}

		defer func() {
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				svc.Logger.Error("failed to rollback transaction", "error", err)
			}
		}()

		// Delete role-permission relationships
		err = db.Query.DeleteManyRolePermissionsByRoleID(ctx, tx, req.RoleId)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete role-permission relationships."),
			)
		}

		// Delete key-role relationships
		err = db.Query.DeleteManyKeyRolesByRoleID(ctx, tx, req.RoleId)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete key-role relationships."),
			)
		}

		// Delete the role itself
		err = db.Query.DeleteRoleByID(ctx, tx, req.RoleId)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete role."),
			)
		}

		// Create audit log for role deletion
		err = svc.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       "role.delete",
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				Display:     "Deleted " + req.RoleId,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        "role",
						ID:          req.RoleId,
						Name:        role.Name,
						DisplayName: role.Name,
						Meta: map[string]interface{}{
							"name":        role.Name,
							"description": role.Description.String,
						},
					},
				},
			},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("audit log error"), fault.Public("Failed to create audit log for role deletion."),
			)
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to commit transaction"), fault.Public("Failed to commit changes."),
			)
		}

		// 7. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
		})
	})
}
