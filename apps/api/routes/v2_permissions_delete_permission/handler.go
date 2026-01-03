package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/apps/api/openapi"
	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
)

type Request = openapi.V2PermissionsDeletePermissionRequestBody
type Response = openapi.V2PermissionsDeletePermissionResponseBody

// Handler implements zen.Route interface for the v2 permissions delete permission endpoint
type Handler struct {
	// Services as public fields
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
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
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}
	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.DeletePermission,
		}),
	)))
	if err != nil {
		return err
	}

	permission, err := db.Query.FindPermissionByIdOrSlug(ctx, h.DB.RO(), db.FindPermissionByIdOrSlugParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Search:      req.Permission,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Internal("permission not found"),
				fault.Public("The requested permission does not exist."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve permission information."),
		)
	}

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.DeleteManyRolePermissionsByPermissionID(ctx, tx, permission.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete role-permission relationships."),
			)
		}

		err = db.Query.DeleteManyKeyPermissionsByPermissionID(ctx, tx, permission.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete key-permission relationships."),
			)
		}

		err = db.Query.DeletePermission(ctx, tx, permission.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete permission."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.PermissionDeleteEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     "Deleted " + permission.ID,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.PermissionResourceType,
						ID:          permission.ID,
						Name:        permission.Slug,
						DisplayName: permission.Name,
						Meta: map[string]interface{}{
							"name":        permission.Name,
							"slug":        permission.Slug,
							"description": permission.Description.String,
						},
					},
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
