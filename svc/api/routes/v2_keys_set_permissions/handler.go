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
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2KeysSetPermissionsRequestBody
	Response = openapi.V2KeysSetPermissionsResponseBody
)

// Handler implements zen.Route interface for the v2 keys set permissions endpoint
type Handler struct {
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	KeyCache  cache.Cache[string, db.CachedKeyData]
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
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), req.KeyId)
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

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(
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
			rbac.And(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Rbac,
					ResourceID:   "*",
					Action:       rbac.AddPermissionToKey,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Rbac,
					ResourceID:   "*",
					Action:       rbac.RemovePermissionFromKey,
				}),
			),
		),
	))
	if err != nil {
		return err
	}

	currentPermissions, err := db.Query.ListDirectPermissionsByKeyID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve current permissions."),
		)
	}

	foundPermissions, err := db.Query.FindPermissionsBySlugs(ctx, h.DB.RO(), db.FindPermissionsBySlugsParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Slugs:       req.Permissions,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to lookup permissions to set."),
		)
	}

	missingPermissions := make(map[string]struct{})
	permissionsToSet := make([]db.Permission, 0)
	permissionsToInsert := make([]db.InsertPermissionParams, 0)

	for _, permission := range req.Permissions {
		missingPermissions[permission] = struct{}{}
	}

	for _, permission := range foundPermissions {
		delete(missingPermissions, permission.ID)
		delete(missingPermissions, permission.Slug)
	}

	permissionsToSet = append(permissionsToSet, foundPermissions...)

	if len(missingPermissions) > 0 {
		err = auth.VerifyRootKey(ctx, keys.WithPermissions(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Rbac,
				ResourceID:   "*",
				Action:       rbac.CreatePermission,
			}),
		))
		if err != nil {
			return err
		}
	}

	for perm := range missingPermissions {
		permissionID := uid.New(uid.PermissionPrefix)
		now := time.Now().UnixMilli()
		permissionsToInsert = append(permissionsToInsert, db.InsertPermissionParams{
			PermissionID: permissionID,
			Name:         perm,
			WorkspaceID:  auth.AuthorizedWorkspaceID,
			Slug:         perm,
			Description:  dbtype.NullString{String: "", Valid: false},
			CreatedAtM:   now,
		})

		permissionsToSet = append(permissionsToSet, db.Permission{
			Pk:          0, // only here to make the linter happy
			ID:          permissionID,
			Name:        perm,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Slug:        perm,
			Description: dbtype.NullString{String: "", Valid: false},
			CreatedAtM:  now,
			UpdatedAtM:  sql.NullInt64{Int64: now, Valid: true},
		})
	}

	currentPermissionMap := make(map[string]db.Permission)
	for _, permission := range currentPermissions {
		currentPermissionMap[permission.ID] = permission
	}

	requestedPermissionIDs := make(map[string]bool)
	requestedPermissionMap := make(map[string]db.Permission)
	for _, permission := range permissionsToSet {
		requestedPermissionIDs[permission.ID] = true
		requestedPermissionMap[permission.ID] = permission
	}

	permissionsToRemove := make([]string, 0)
	for _, permission := range currentPermissions {
		if !requestedPermissionIDs[permission.ID] {
			permissionsToRemove = append(permissionsToRemove, permission.ID)
		}
	}

	permissionsToAdd := make([]db.Permission, 0)
	for _, permission := range permissionsToSet {
		_, ok := currentPermissionMap[permission.ID]
		if !ok {
			permissionsToAdd = append(permissionsToAdd, permission)
		}
	}

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Lock the key row to prevent concurrent modifications and deadlocks
		_, err := db.Query.LockKeyForUpdate(ctx, tx, key.ID)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("unable to lock key"),
				fault.Public("We're unable to update the key."),
			)
		}

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

			for _, permissionID := range permissionsToRemove {
				perm := currentPermissionMap[permissionID]
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
							Type:        auditlog.KeyResourceType,
							ID:          req.KeyId,
							Name:        key.Name.String,
							DisplayName: key.Name.String,
							Meta:        map[string]any{},
						},
						{
							Type:        auditlog.PermissionResourceType,
							ID:          permissionID,
							Name:        perm.Slug,
							DisplayName: perm.Name,
							Meta:        map[string]any{},
						},
					},
				})
			}
		}

		if len(permissionsToInsert) > 0 {
			for _, toCreate := range permissionsToInsert {
				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.PermissionCreateEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Created %s (%s)", toCreate.Slug, toCreate.PermissionID),
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type:        auditlog.PermissionResourceType,
							ID:          toCreate.PermissionID,
							Name:        toCreate.Slug,
							DisplayName: toCreate.Name,
							Meta: map[string]any{
								"name": toCreate.Name,
								"slug": toCreate.Slug,
							},
						},
					},
				})
			}

			err = db.BulkQuery.InsertPermissions(ctx, tx, permissionsToInsert)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to insert permissions."),
				)
			}
		}

		if len(permissionsToAdd) > 0 {
			toAdd := make([]db.InsertKeyPermissionParams, len(permissionsToAdd))
			now := time.Now().UnixMilli()

			for idx, permission := range permissionsToAdd {
				toAdd[idx] = db.InsertKeyPermissionParams{
					KeyID:        req.KeyId,
					PermissionID: permission.ID,
					WorkspaceID:  auth.AuthorizedWorkspaceID,
					CreatedAt:    now,
					UpdatedAt:    sql.NullInt64{Valid: true, Int64: now},
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

			err = db.BulkQuery.InsertKeyPermissions(ctx, tx, toAdd)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to add permissions to key."),
				)
			}
		}

		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	h.KeyCache.Remove(ctx, key.Hash)

	responseData := make(openapi.V2KeysSetPermissionsResponseData, 0)
	for _, permission := range permissionsToSet {
		perm := openapi.Permission{
			Description: permission.Description.String,
			Id:          permission.ID,
			Name:        permission.Name,
			Slug:        permission.Slug,
		}

		responseData = append(responseData, perm)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
