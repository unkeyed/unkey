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

type Request = openapi.V2PermissionsDeletePermissionRequestBody
type Response = openapi.V2PermissionsDeletePermissionResponseBody

// Handler implements zen.Route interface for the v2 permissions delete permission endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/permissions.deletePermission"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
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
				Action:       rbac.DeletePermission,
			}),
		),
	)
	if err != nil {
		return err
	}

	// 4. Check if permission exists and belongs to authorized workspace
	permission, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), req.PermissionId)
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
	tx, err := h.DB.RW().Begin(ctx)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to create transaction"), fault.Public("Unable to start database transaction."),
		)
	}

	defer func() {
		if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
			h.Logger.Error("failed to rollback transaction", "error", err)
		}
	}()

	// Delete role-permission relationships
	err = db.Query.DeleteManyRolePermissionsByPermissionID(ctx, tx, req.PermissionId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to delete role-permission relationships."),
		)
	}

	// Delete key-permission relationships
	err = db.Query.DeleteManyKeyPermissionsByPermissionID(ctx, tx, req.PermissionId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to delete key-permission relationships."),
		)
	}

	// Delete the permission itself
	err = db.Query.DeletePermission(ctx, tx, req.PermissionId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to delete permission."),
		)
	}

	// Create audit log for permission deletion
	err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
		{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       "permission.delete",
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.KeyID,
			ActorName:   "root key",
			Display:     "Deleted " + req.PermissionId,
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
			Resources: []auditlog.AuditLogResource{
				{
					Type:        "permission",
					ID:          req.PermissionId,
					Name:        permission.Name,
					DisplayName: permission.Name,
					Meta: map[string]interface{}{
						"name":        permission.Name,
						"description": permission.Description.String,
					},
				},
			},
		},
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("audit log error"), fault.Public("Failed to create audit log for permission deletion."),
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

	// 6. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
	})
}
