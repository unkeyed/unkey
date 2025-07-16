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

	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.Verify(ctx, keys.WithPermissions(
		rbac.And(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.UpdateKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Rbac,
				ResourceID:   "*",
				Action:       rbac.AddPermissionToKey,
			}),
		),
	))
	if err != nil {
		return err
	}

	key, err := db.Query.FindKeyByID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key not found"),
				fault.Public("The specified key was not found."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve key."),
		)
	}

	// Validate key belongs to authorized workspace
	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve current permissions."),
		)
	}

	foundPermissions, err := db.Query.FindPermissionsBySlugs(ctx, h.DB.RO(), db.FindPermissionsBySlugsParams{
		Slugs:       req.Permissions,
		WorkspaceID: auth.AuthorizedWorkspaceID,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve permissions."),
		)
	}

	requestedPermissions := make(map[string]db.FindPermissionsBySlugsRow)
	permissionsToCreate := make([]db.InsertPermissionParams, 0)

	for _, foundPermission := range foundPermissions {
		requestedPermissions[foundPermission.Slug] = foundPermission
	}

	for _, permissionsToSet := range req.Permissions {
		_, ok := requestedPermissions[permissionsToSet]
		if ok {
			continue
		}

		permissionID := uid.New(uid.PermissionPrefix)
		permissionsToCreate = append(permissionsToCreate, db.InsertPermissionParams{
			PermissionID: permissionID,
			WorkspaceID:  auth.AuthorizedWorkspaceID,
			Name:         permissionsToSet,
			Slug:         permissionsToSet,
			Description:  sql.NullString{String: "", Valid: false},
			CreatedAtM:   time.Now().UnixMilli(),
		})

		requestedPermissions[permissionsToSet] = db.FindPermissionsBySlugsRow{
			ID:   permissionID,
			Slug: permissionsToSet,
		}
	}

	if len(permissionsToCreate) > 0 {
		err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Rbac,
				ResourceID:   "*",
				Action:       rbac.CreatePermission,
			}),
		)))
		if err != nil {
			return err
		}
	}

	currentPermissionIDs := make(map[string]bool)
	requestedPermissionIDs := make(map[string]bool)
	permissionsToAdd := make([]db.FindPermissionsBySlugsRow, 0)
	permissionsToRemove := make(map[string]db.FindPermissionsBySlugsRow, 0)

	for _, permission := range currentPermissions {
		currentPermissionIDs[permission.Slug] = true
	}

	for _, permission := range requestedPermissions {
		requestedPermissionIDs[permission.ID] = true
		if !currentPermissionIDs[permission.ID] {
			permissionsToAdd = append(permissionsToAdd, permission)
		}
	}

	for _, permission := range currentPermissions {
		_, found := requestedPermissionIDs[permission.ID]
		if found {
			continue
		}

		permissionsToRemove[permission.ID] = db.FindPermissionsBySlugsRow{
			ID:   permission.ID,
			Slug: permission.Slug,
			Name: permission.Name,
		}
	}

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		var auditLogs []auditlog.AuditLog

		if len(permissionsToCreate) > 0 {
			for _, permission := range permissionsToCreate {
				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.PermissionCreateEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Created %s (%s)", permission.Slug, permission.PermissionID),
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type:        "permission",
							ID:          permission.PermissionID,
							Name:        permission.Name,
							DisplayName: permission.Name,
							Meta: map[string]interface{}{
								"name":        permission.Name,
								"slug":        permission.Name,
								"description": permission.Description.String,
							},
						},
					},
				})
			}

			err = db.BulkInsert(ctx, tx, "INSERT INTO permissions (id, workspace_id, name, slug, description, created_at_m) VALUES (?, ?, ?, ?, ?, ?)", permissionsToCreate)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to add permission assignments."),
				)
			}
		}

		permIds := make([]string, 0)
		for id, perm := range permissionsToRemove {
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.AuthDisconnectPermissionKeyEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     fmt.Sprintf("Removed permission %s from key %s", perm.Name, req.KeyId),
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
						ID:          perm.ID,
						Name:        perm.Name,
						DisplayName: perm.Name,
						Meta:        map[string]any{},
					},
				},
			})

			permIds = append(permIds, id)
		}

		if len(permissionsToRemove) > 0 {
			err = db.Query.DeleteKeyPermissionByKeyAndPermissionIDs(ctx, tx, db.DeleteKeyPermissionByKeyAndPermissionIDsParams{
				KeyID:         req.KeyId,
				PermissionIds: permIds,
			})

			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to remove permission assignment."),
				)
			}
		}

		permissionsToInsert := make([]db.InsertKeyPermissionParams, 0)
		for _, permission := range permissionsToAdd {
			permissionsToInsert = append(permissionsToInsert, db.InsertKeyPermissionParams{
				KeyID:        req.KeyId,
				PermissionID: permission.ID,
				WorkspaceID:  auth.AuthorizedWorkspaceID,
				CreatedAt:    time.Now().UnixMilli(),
			})

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

		if len(permissionsToInsert) > 0 {
			err = db.BulkInsert(ctx, tx, "INSERT INTO key_permissions (key_id, permission_id, workspace_id, created_at_m) VALUES (?, ?, ?, ?)", permissionsToInsert)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to add permission assignments."),
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

	finalPermissions := make([]db.FindPermissionsBySlugsRow, 0)
	for _, perm := range requestedPermissions {
		finalPermissions = append(finalPermissions, perm)
	}

	slices.SortFunc(finalPermissions, func(a, b db.FindPermissionsBySlugsRow) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})

	responseData := make(openapi.V2KeysSetPermissionsResponseData, len(finalPermissions))
	for i, permission := range finalPermissions {
		res := struct {
			Description *string `json:"description,omitempty"`
			Name        string  `json:"name"`
			Slug        string  `json:"slug"`
		}{
			Description: nil,
			Name:        permission.Name,
			Slug:        permission.Slug,
		}

		if permission.Description.Valid {
			res.Description = &permission.Description.String
		}

		responseData[i] = res
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
