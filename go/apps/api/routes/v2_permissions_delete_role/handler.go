package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
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

// Handler implements zen.Route interface for the v2 permissions delete role endpoint
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
	return "/v2/permissions.deleteRole"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.deleteRole")

	// 1. Authentication
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.DeleteRole,
		}),
	)))
	if err != nil {
		return err
	}

	role, err := db.Query.FindRoleByIdOrNameWithPerms(ctx, h.DB.RO(), db.FindRoleByIdOrNameWithPermsParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Search:      req.Role,
	})
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

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.DeleteManyRolePermissionsByRoleID(ctx, tx, role.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete role-permission relationships."),
			)
		}

		err = db.Query.DeleteManyKeyRolesByRoleID(ctx, tx, role.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete key-role relationships."),
			)
		}

		err = db.Query.DeleteRoleByID(ctx, tx, role.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to delete role."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RoleDeleteEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     "Deleted " + role.ID,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.RoleResourceType,
						ID:          role.ID,
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
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 7. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
