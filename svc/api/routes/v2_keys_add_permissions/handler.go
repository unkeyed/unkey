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
	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2KeysAddPermissionsRequestBody
type Response = openapi.V2KeysAddPermissionsResponseBody

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
	return "/v2/keys.addPermissions"
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

	key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), req.KeyId)
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

	wide.Set(ctx, wide.FieldKeyID, key.ID)

	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
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
			fault.Internal("database error"), fault.Public("Failed to lookup permissions to add."),
		)
	}

	missingPermissions := make(map[string]struct{})
	permissionsToSet := make([]db.Permission, 0)
	permissionsToInsert := make([]db.InsertPermissionParams, 0)
	currentPermissionIDs := make(map[string]db.Permission)

	for _, permission := range currentPermissions {
		currentPermissionIDs[permission.ID] = permission
	}

	for _, permission := range req.Permissions {
		missingPermissions[permission] = struct{}{}
	}

	for _, permission := range foundPermissions {
		delete(missingPermissions, permission.ID)
		delete(missingPermissions, permission.Slug)
	}

	for _, permission := range foundPermissions {
		_, ok := currentPermissionIDs[permission.ID]
		if ok {
			continue
		}

		permissionsToSet = append(permissionsToSet, permission)
	}

	for perm := range missingPermissions {
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

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		// Lock the key row to prevent concurrent modifications and deadlocks
		_, err := db.Query.LockKeyForUpdate(ctx, tx, req.KeyId)
		if err != nil {
			return fault.Wrap(err,
				fault.Internal("unable to lock key"),
				fault.Public("We're unable to update the key."),
			)
		}

		var auditLogs []auditlog.AuditLog

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
							Meta: map[string]interface{}{
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

		if len(permissionsToSet) > 0 {
			toAdd := make([]db.InsertKeyPermissionParams, len(permissionsToSet))
			now := time.Now().UnixMilli()

			for idx, permission := range permissionsToSet {
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
					fault.Public("Failed to add key permissions."),
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

	responseData := make(openapi.V2KeysAddPermissionsResponseData, 0)

	for _, permission := range currentPermissions {
		perm := openapi.Permission{
			Id:          permission.ID,
			Name:        permission.Name,
			Slug:        permission.Slug,
			Description: permission.Description.String,
		}

		responseData = append(responseData, perm)
	}

	for _, permission := range permissionsToSet {
		perm := openapi.Permission{
			Id:          permission.ID,
			Name:        permission.Name,
			Slug:        permission.Slug,
			Description: permission.Description.String,
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
