package handler

import (
	"context"
	"database/sql"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsCreatePermissionRequestBody
type Response = openapi.V2PermissionsCreatePermissionResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.createPermission", func(ctx context.Context, s *zen.Session) error {

		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		req, err := zen.BindBody[Request](s)
		if err != nil {
			return err
		}

		err = svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Rbac,
				ResourceID:   "*",
				Action:       rbac.CreatePermission,
			}),
		)

		if err != nil {
			return err
		}

		permissionID := uid.New(uid.PermissionPrefix)

		description := ptr.SafeDeref(req.Description)

		// Create permission in a transaction with audit log
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

		// Insert the permission
		err = db.Query.InsertPermission(ctx, tx, db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  auth.AuthorizedWorkspaceID,
			Name:         req.Name,
			Description:  sql.NullString{Valid: description != "", String: description},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		if err != nil {
			if db.IsDuplicateKeyError(err) {

				return fault.New("permission already exists",
					fault.Code(codes.UnkeyDataErrorsIdentityDuplicate), // Reuse the identity duplicate code for conflict status
					fault.Internal("already exists"), fault.Public("A permission with name \""+req.Name+"\" already exists in this workspace"),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to create permission."),
			)
		}

		// Create audit log
		err = svc.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       "permission.create",
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				Display:     "Created " + permissionID,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        "permission",
						ID:          permissionID,
						Name:        req.Name,
						DisplayName: req.Name,
						Meta: map[string]interface{}{
							"name":        req.Name,
							"description": description,
						},
					},
				},
			},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("audit log error"), fault.Public("Failed to create audit log for permission creation."),
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
		if err != nil {
			return err
		}

		// 5. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.PermissionsCreatePermissionResponseData{
				PermissionId: permissionID,
			},
		})
	})
}
