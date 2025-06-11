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

// Handler implements zen.Route interface for the v2 permissions create role endpoint
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
	return "/v2/permissions.createRole"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.createRole")

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
	_, err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (interface{}, error) {
		// Insert the role
		err := db.Query.InsertRole(ctx, tx, db.InsertRoleParams{
			RoleID:      roleID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Name:        req.Name,
			Description: sql.NullString{Valid: description != "", String: description},
		})
		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return nil, fault.New("role already exists",
					fault.Code(codes.UnkeyDataErrorsIdentityDuplicate),
					fault.Internal("role already exists"), fault.Public("A role with name \""+req.Name+"\" already exists in this workspace"),
				)
			}
			return nil, fault.Wrap(err,
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
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("audit log error"), fault.Public("Failed to create audit log for role creation."),
			)
		}

		return nil, nil
	})
	if err != nil {
		return err
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
}
