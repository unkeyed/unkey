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
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysAddPermissionsRequestBody
type Response = openapi.V2KeysAddPermissionsResponseBody

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
	return "/v2/keys.addPermissions"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := db.Query.FindKeyByIdOrHash(ctx,
		h.DB.RO(),
		db.FindKeyByIdOrHashParams{
			ID:   sql.NullString{String: req.KeyId, Valid: true},
			Hash: sql.NullString{String: "", Valid: false},
		},
	)
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

	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"), fault.Public("The specified key was not found."),
		)
	}

	if key.Api.DeletedAtM.Valid {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to deleted api"),
			fault.Public("The specified key was not found."),
		)
	}

	currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve current permissions."),
		)
	}

	err = auth.Verify(ctx, keys.WithPermissions(
		rbac.And(
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.UpdateKey,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   key.Api.ID,
					Action:       rbac.UpdateKey,
				}),
			),
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Rbac,
					ResourceID:   "*",
					Action:       rbac.AddPermissionToKey,
				}),
			),
		),
	))
	if err != nil {
		return err
	}

	// Convert current permissions to a map for efficient lookup
	currentPermissionIDs := make(map[string]bool)
	for _, permission := range currentPermissions {
		currentPermissionIDs[permission.ID] = true
	}

	// 6. Resolve and validate requested permissions
	requestedPermissions := make([]db.Permission, 0, len(req.Permissions))
	for _, permRef := range req.Permissions {
		var permission db.Permission

		if permRef.Id != nil {
			// Find by ID
			permission, err = db.Query.FindPermissionByID(ctx, h.DB.RO(), *permRef.Id)
			if err != nil {
				if db.IsNotFound(err) {
					return fault.New("permission not found",
						fault.Code(codes.Data.Permission.NotFound.URN()),
						fault.Internal("permission not found"), fault.Public(fmt.Sprintf("Permission with ID '%s' was not found.", *permRef.Id)),
					)
				}
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve permission."),
				)
			}
		} else if permRef.Slug != nil {
			// Find by name
			permission, err = db.Query.FindPermissionBySlugAndWorkspaceID(ctx, h.DB.RO(), db.FindPermissionBySlugAndWorkspaceIDParams{
				Slug:        *permRef.Slug,
				WorkspaceID: auth.AuthorizedWorkspaceID,
			})
			if err != nil {
				if db.IsNotFound(err) {
					return fault.New("permission not found",
						fault.Code(codes.Data.Permission.NotFound.URN()),
						fault.Internal("permission not found"), fault.Public(fmt.Sprintf("Permission with slug '%s' was not found.", *permRef.Slug)),
					)
				}
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve permission."),
				)
			}
		} else {
			return fault.New("invalid permission reference",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("permission missing id and name"), fault.Public("Each permission must specify either 'id' or 'slug'."),
			)
		}

		// Validate permission belongs to the same workspace
		if permission.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Internal("permission belongs to different workspace"), fault.Public(fmt.Sprintf("Permission '%s' was not found.", permission.Name)),
			)
		}

		requestedPermissions = append(requestedPermissions, permission)
	}

	// 7. Determine which permissions to add (only add permissions that aren't already assigned)
	permissionsToAdd := make([]db.Permission, 0)
	for _, permission := range requestedPermissions {
		if !currentPermissionIDs[permission.ID] {
			permissionsToAdd = append(permissionsToAdd, permission)
		}
	}

	// 8. Apply changes in transaction (only if there are permissions to add)
	if len(permissionsToAdd) > 0 {
		err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			var auditLogs []auditlog.AuditLog

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
							Type:        auditlog.KeyResourceType,
							ID:          req.KeyId,
							Name:        key.Name.String,
							DisplayName: key.Name.String,
							Meta:        map[string]any{},
						},
						{
							Type:        auditlog.PermissionResourceType,
							ID:          permission.ID,
							Name:        permission.Slug,
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
	}

	// 9. Get final state of direct permissions and build response
	finalPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RW(), req.KeyId)
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
	responseData := make(openapi.V2KeysAddPermissionsResponseData, len(finalPermissions))
	for i, permission := range finalPermissions {
		responseData[i] = struct {
			Id   string `json:"id"`
			Name string `json:"name"`
			Slug string `json:"slug"`
		}{
			Id:   permission.ID,
			Name: permission.Name,
			Slug: permission.Slug,
		}
	}

	// 10. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
