package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2PermissionsCreatePermissionRequestBody
type Response = openapi.V2PermissionsCreatePermissionResponseBody

// Handler implements zen.Route interface for the v2 permissions create permission endpoint
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
	return "/v2/permissions.createPermission"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.T(rbac.Tuple{
		ResourceType: rbac.Rbac,
		ResourceID:   "*",
		Action:       rbac.CreatePermission,
	})))
	if err != nil {
		return err
	}

	permissionID := uid.New(uid.PermissionPrefix)

	description := ptr.SafeDeref(req.Description)

	// Create permission in a transaction with audit log
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Insert the permission
		err = db.Query.InsertPermission(ctx, tx, db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  auth.AuthorizedWorkspaceID,
			Name:         req.Name,
			Slug:         req.Slug,
			Description:  dbtype.NullString{Valid: description != "", String: description},
			CreatedAtM:   time.Now().UnixMilli(),
		})
		if err != nil {
			if db.IsDuplicateKeyError(err) {
				return fault.New("permission already exists",
					fault.Code(codes.Data.Permission.Duplicate.URN()),
					fault.Internal("already exists"),
					fault.Public(fmt.Sprintf("A permission with slug '%s' already exists in this workspace", req.Slug)),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to create permission."),
			)
		}

		// Create audit log
		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.PermissionCreateEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     "Created " + permissionID,
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.PermissionResourceType,
						ID:          permissionID,
						Name:        req.Slug,
						DisplayName: req.Name,
						Meta: map[string]interface{}{
							"name":        req.Name,
							"slug":        req.Slug,
							"description": description,
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

	// 5. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2PermissionsCreatePermissionResponseData{
			PermissionId: permissionID,
		},
	})
}
