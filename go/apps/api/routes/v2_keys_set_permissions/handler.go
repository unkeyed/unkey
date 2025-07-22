package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysSetPermissionsRequestBody
type Response = openapi.V2KeysSetPermissionsResponseBody

// Handler implements zen.Route interface for the v2 keys set permissions endpoint
type Handler struct {
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	KeyCache  cache.Cache[string, db.FindKeyForVerificationRow]
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.setPermissions"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.setPermissions")

	// 1. Authentication
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// 3. Permission check
	err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.UpdateKey,
		}),
	)))
	if err != nil {
		return err
	}

	// 4. Validate key exists and belongs to workspace
	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key not found"), fault.Public("The specified key was not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve key."),
		)
	}

	// Validate key belongs to authorized workspace
	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"), fault.Public("The specified key was not found."),
		)
	}

	// 5. Get current direct permissions for the key
	currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve current permissions."),
		)
	}

	permissions, err := db.Query.FindManyPermissionsByIdOrSlug(ctx, h.DB.RO(), db.FindManyPermissionsByIdOrSlugParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Ids:         req.Permissions,
	})

	missingPermissions := make(map[string]struct{})
	for _, permission := range req.Permissions {
		missingPermissions[permission] = struct{}{}
	}

	for _, permission := range permissions {
		if _, ok := missingPermissions[permission.ID]; ok {
			delete(missingPermissions, permission.ID)
		}

		if _, ok := missingPermissions[permission.Slug]; ok {
			delete(missingPermissions, permission.Slug)
		}
	}

	permissionsToSet := make([]db.Permission, 0)
	permissionsToInsert := make([]db.InsertPermissionParams, 0)

	for _, permission := range permissions {
		permissionsToSet = append(permissionsToSet, permission)
	}

	for perm := range missingPermissions {
		if strings.HasPrefix(perm, "perm_") {
			return fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Internal("permission not found"),
				fault.Public(fmt.Sprintf("Permission with ID '%s' was not found.", perm)),
			)
		}

		permissionID := uid.New(uid.PermissionPrefix)
		now := time.Now().UnixMilli()
		permissionsToInsert = append(permissionsToInsert, db.InsertPermissionParams{
			PermissionID: permissionID,
			Name:         perm,
			WorkspaceID:  auth.AuthorizedWorkspaceID,
			Slug:         perm,
			Description:  sql.NullString{String: "", Valid: false},
			CreatedAtM:   now,
		})

		permissionsToSet = append(permissionsToSet, db.Permission{
			ID:          permissionID,
			Name:        perm,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Slug:        perm,
			Description: sql.NullString{String: "", Valid: false},
			CreatedAtM:  now,
		})
	}

	// 7. Calculate differential update
	// Create maps for efficient lookup
	currentPermissionIDs := make(map[string]bool)
	for _, permission := range currentPermissions {
		currentPermissionIDs[permission.ID] = true
	}

	requestedPermissionIDs := make(map[string]bool)
	requestedPermissionMap := make(map[string]db.Permission)
	for _, permission := range requestedPermissions {
		requestedPermissionIDs[permission.ID] = true
		requestedPermissionMap[permission.ID] = permission
	}

	// Determine permissions to remove and add
	permissionsToRemove := make([]string, 0)
	for _, permission := range currentPermissions {
		if !requestedPermissionIDs[permission.ID] {
			permissionsToRemove = append(permissionsToRemove, permission.ID)
		}
	}

	permissionsToAdd := make([]db.Permission, 0)
	for _, permission := range requestedPermissions {
		if !currentPermissionIDs[permission.ID] {
			permissionsToAdd = append(permissionsToAdd, permission)
		}
	}

	// 8. Apply changes in transaction
	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		var auditLogs []auditlog.AuditLog

		if len(permissionsToRemove) > 0 {
			err = db.Query.DeleteManyKeyPermissionByKeyAndPermissionIDs(ctx, tx, db.DeleteManyKeyPermissionByKeyAndPermissionIDsParams{
				KeyID: req.KeyId,
				Ids:   permissionsToRemove,
			})

			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to remove permission assignment."),
				)
			}

			for _, perm := range permissionsToRemove {
				// auditLogs = append(auditLogs, auditlog.AuditLog{
				// 	WorkspaceID: auth.AuthorizedWorkspaceID,
				// 	Event:       auditlog.AuthDisconnectPermissionKeyEvent,
				// 	ActorType:   auditlog.RootKeyActor,
				// 	ActorID:     auth.Key.ID,
				// 	ActorName:   "root key",
				// 	ActorMeta:   map[string]any{},
				// 	Display:     fmt.Sprintf("Removed permission %s from key %s", permissionName, req.KeyId),
				// 	RemoteIP:    s.Location(),
				// 	UserAgent:   s.UserAgent(),
				// 	Resources: []auditlog.AuditLogResource{
				// 		{
				// 			Type:        "key",
				// 			ID:          req.KeyId,
				// 			Name:        key.Name.String,
				// 			DisplayName: key.Name.String,
				// 			Meta:        map[string]any{},
				// 		},
				// 		{
				// 			Type:        "permission",
				// 			ID:          permissionID,
				// 			Name:        permissionName,
				// 			DisplayName: permissionName,
				// 			Meta:        map[string]any{},
				// 		},
				// 	},
				// })
			}
		}

		if len(permissionsToAdd) > 0 {
			toAdd := make([]db.InsertKeyPermissionParams, len(permissionsToAdd))
			for idx, permission := range permissionsToAdd {
				toAdd[idx] = db.InsertKeyPermissionParams{
					KeyID:        req.KeyId,
					PermissionID: permission.ID,
					WorkspaceID:  auth.AuthorizedWorkspaceID,
					CreatedAt:    time.Now().UnixMilli(),
				}

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.AuthConnectPermissionKeyEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Added permission %s to key %s", permission.Name, req.KeyId),
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type:        "key",
							ID:          req.KeyId,
							Name:        key.Name.String,
							DisplayName: key.Name.String,
							Meta:        map[string]any{},
						},
						{
							Type:        "permission",
							ID:          permission.ID,
							Name:        permission.Slug,
							DisplayName: permission.Name,
							Meta:        map[string]any{},
						},
					},
				})
			}

			err = db.BulkQuery.InsertKeyPermissions(ctx, h.DB.RW(), toAdd)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to add permissions to key."),
				)
			}
		}

		// Insert audit logs
		if len(auditLogs) > 0 {
			err = h.Auditlogs.Insert(ctx, tx, auditLogs)
			if err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return err
	}

	h.KeyCache.Remove(ctx, key.Hash)

	// Build response data
	responseData := make(openapi.V2KeysSetPermissionsResponseData, 0)

	// 11. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
