package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2PermissionsCreateRoleRequestBody
type Response = openapi.V2PermissionsCreateRoleResponseBody

// Handler implements zen.Route interface for the v2 permissions create role endpoint
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
	return "/v2/permissions.createRole"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.createRole")

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

	// 3. Permission check
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.CreateRole,
		}),
	)))
	if err != nil {
		return err
	}

	// 4. Prepare role creation
	roleID := uid.New(uid.RolePrefix)
	description := ptr.SafeDeref(req.Description)

	// 5. Create role in a transaction with audit log
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Insert the role
		err = db.Query.InsertRole(ctx, tx, db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Name:        req.Name,
			Description: sql.NullString{Valid: description != "", String: description},
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return fault.New("role already exists",
					fault.Code(codes.Data.Role.Duplicate.URN()),
					fault.Internal("role already exists"), fault.Public(fmt.Sprintf("A role with name '%s' already exists in this workspace", req.Name)),
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

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RoleCreateEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     "Created " + roleID,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.RoleResourceType,
						ID:          roleID,
						Name:        req.Name,
						DisplayName: req.Name,
						Meta:        metaData,
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
		Data: openapi.V2PermissionsCreateRoleResponseData{
			RoleId: roleID,
		},
	})
}
