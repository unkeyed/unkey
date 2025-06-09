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
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsCreateRoleRequestBody
type Response = openapi.V2PermissionsCreateRoleResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.createRole", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.createRole")

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
					Action:       rbac.CreateRole,
				}),
			),
		)
		if err != nil {
			return err
		}

		// 4. Prepare role creation
		roleID := uid.New(uid.RolePrefix)
		description := ptr.SafeDeref(req.Description)

		// 5. Create role in a transaction with audit log
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

		// Insert the role
		err = db.Query.InsertRole(ctx, tx, db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Name:        req.Name,
			Description: sql.NullString{Valid: description != "", String: description},
		})
		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return fault.New("role already exists",
					fault.Code(codes.UnkeyDataErrorsIdentityDuplicate),
					fault.Internal("role already exists"), fault.Public("A role with name \""+req.Name+"\" already exists in this workspace"),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to create role."),
			)
		}

		// Create audit log
		metaData := map[string]interface{}{
			"name":        req.Name,
			"description": description,
		}

		err = svc.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       "role.create",
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				Display:     "Created " + roleID,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        "role",
						ID:          roleID,
						Name:        req.Name,
						DisplayName: req.Name,
						Meta:        metaData,
					},
				},
			},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("audit log error"), fault.Public("Failed to create audit log for role creation."),
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
			Data: openapi.PermissionsCreateRoleResponseData{
				RoleId: roleID,
			},
		})
	})
}
