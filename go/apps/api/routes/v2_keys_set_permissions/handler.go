package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"
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
type Response = openapi.V2KeysSetPermissionsResponse

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

	// 6. Resolve and validate requested permissions
	requestedPermissions := make([]db.Permission, 0, len(req.Permissions))
	for _, permissionRef := range req.Permissions {
		var permission db.Permission

		// nolint:nestif
		if permissionRef.Id != nil && *permissionRef.Id != "" {
			// Find by ID
			permission, err = db.Query.FindPermissionByID(ctx, h.DB.RO(), *permissionRef.Id)
			if err != nil {
				if db.IsNotFound(err) {
					return fault.New("permission not found",
						fault.Code(codes.Data.Permission.NotFound.URN()),
						fault.Internal("permission not found"), fault.Public(fmt.Sprintf("Permission with ID '%s' was not found.", *permissionRef.Id)),
					)
				}
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve permission."),
				)
			}
		} else if permissionRef.Slug != nil && *permissionRef.Slug != "" {
			// Find by slug
			permission, err = db.Query.FindPermissionBySlugAndWorkspaceID(ctx, h.DB.RO(), db.FindPermissionBySlugAndWorkspaceIDParams{
				Slug:        *permissionRef.Slug,
				WorkspaceID: auth.AuthorizedWorkspaceID,
			})
			if err != nil {
				if db.IsNotFound(err) {
					// Check if we should create the permission
					if permissionRef.Create != nil && *permissionRef.Create {
						// Create permission using slug as both name and slug
						permissionID := uid.New(uid.PermissionPrefix)
						err = db.Query.InsertPermission(ctx, h.DB.RW(), db.InsertPermissionParams{
							PermissionID: permissionID,
							WorkspaceID:  auth.AuthorizedWorkspaceID,
							Name:         *permissionRef.Slug,
							Slug:         *permissionRef.Slug,
							Description:  sql.NullString{String: "", Valid: false},
							CreatedAtM:   time.Now().UnixMilli(),
						})
						if err != nil {
							return fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Internal("database error"), fault.Public("Failed to create permission."),
							)
						}

						// Fetch the newly created permission
						permission, err = db.Query.FindPermissionByID(ctx, h.DB.RO(), permissionID)
						if err != nil {
							return fault.Wrap(err,
								fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
								fault.Internal("database error"), fault.Public("Failed to retrieve created permission."),
							)
						}
					} else {
						return fault.New("permission not found",
							fault.Code(codes.Data.Permission.NotFound.URN()),
							fault.Internal("permission not found"), fault.Public(fmt.Sprintf("Permission with slug '%s' was not found.", *permissionRef.Slug)),
						)
					}
				} else {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to retrieve permission."),
					)
				}
			}
		} else {
			return fault.New("invalid permission reference",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("permission missing id and slug"), fault.Public("Each permission must specify either 'id' or 'slug'."),
			)
		}

		// Validate permission belongs to the same workspace
		if permission.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Internal("permission belongs to different workspace"), fault.Public(fmt.Sprintf("Permission '%s' was not found.", permission.Slug)),
			)
		}

		requestedPermissions = append(requestedPermissions, permission)
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

		// Remove permissions that are no longer needed
		for _, permissionID := range permissionsToRemove {
			err = db.Query.DeleteKeyPermissionByKeyAndPermissionID(ctx, tx, db.DeleteKeyPermissionByKeyAndPermissionIDParams{
				KeyID:        req.KeyId,
				PermissionID: permissionID,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to remove permission assignment."),
				)
			}

			// Find the permission for audit log
			var permissionName string
			for _, p := range currentPermissions {
				if p.ID == permissionID {
					permissionName = p.Name
					break
				}
			}

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.AuthDisconnectPermissionKeyEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     fmt.Sprintf("Removed permission %s from key %s", permissionName, req.KeyId),
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
						ID:          permissionID,
						Name:        permissionName,
						DisplayName: permissionName,
						Meta:        map[string]any{},
					},
				},
			})
		}

		// Add new permissions
		for _, permission := range permissionsToAdd {
			err = db.Query.InsertKeyPermission(ctx, tx, db.InsertKeyPermissionParams{
				KeyID:        req.KeyId,
				PermissionID: permission.ID,
				WorkspaceID:  auth.AuthorizedWorkspaceID,
				CreatedAt:    time.Now().UnixMilli(),
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to add permission assignment."),
				)
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
						Name:        permission.Name,
						DisplayName: permission.Name,
						Meta:        map[string]any{},
					},
				},
			})
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

	// 10. Get final state of permissions and build response
	finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve final permission state."),
		)
	}

	// Sort permissions alphabetically by name for consistent response
	slices.SortFunc(finalPermissions, func(a, b db.Permission) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})

	// Build response data
	responseData := make(openapi.V2KeysSetPermissionsResponseData, len(finalPermissions))
	for i, permission := range finalPermissions {
		responseData[i] = struct {
			Id   string `json:"id"`
			Name string `json:"name"`
		}{
			Id:   permission.ID,
			Name: permission.Name,
		}
	}

	// 11. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
