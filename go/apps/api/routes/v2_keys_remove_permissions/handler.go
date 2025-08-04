package handler

import (
	"context"
	"fmt"
	"net/http"

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

type Request = openapi.V2KeysRemovePermissionsRequestBody
type Response = openapi.V2KeysRemovePermissionsResponseBody

// Handler implements zen.Route interface for the v2 keys remove permissions endpoint
type Handler struct {
	// Services as public fields
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
	return "/v2/keys.removePermissions"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.removePermissions")

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

	// Validate key belongs to authorized workspace
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
			fault.Internal("database error"),
			fault.Public("Failed to retrieve current permissions."),
		)
	}

	currentPermissionIDs := make(map[string]db.Permission)
	for _, permission := range currentPermissions {
		currentPermissionIDs[permission.ID] = permission
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

	existingPermissions := make(map[string]db.Permission)
	for _, permission := range foundPermissions {
		existingPermissions[permission.ID] = permission
		existingPermissions[permission.Slug] = permission
	}

	for _, toRemove := range req.Permissions {
		_, exists := existingPermissions[toRemove]

		if !exists {
			return fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Public(fmt.Sprintf("Permission '%s' was not found.", toRemove)),
			)
		}
	}

	permissionsToRemove := make([]db.Permission, 0)
	for _, permission := range foundPermissions {
		_, exists := currentPermissionIDs[permission.ID]
		if !exists {
			continue
		}

		delete(currentPermissionIDs, permission.ID)
		permissionsToRemove = append(permissionsToRemove, permission)
	}

	if len(permissionsToRemove) > 0 {
		err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			var auditLogs []auditlog.AuditLog
			var idsToRemove []string

			for _, permission := range permissionsToRemove {
				idsToRemove = append(idsToRemove, permission.ID)
				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.AuthDisconnectPermissionKeyEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Removed permission %s from key %s", permission.Name, req.KeyId),
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

			err = db.Query.DeleteManyKeyPermissionByKeyAndPermissionIDs(ctx, tx, db.DeleteManyKeyPermissionByKeyAndPermissionIDsParams{
				KeyID: req.KeyId,
				Ids:   idsToRemove,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to remove permission assignment."),
				)
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
	}

	responseData := make(openapi.V2KeysRemovePermissionsResponseData, 0)
	for _, permission := range currentPermissionIDs {
		perm := openapi.Permission{
			Id:          permission.ID,
			Slug:        permission.Slug,
			Name:        permission.Name,
			Description: nil,
		}

		if permission.Description.Valid {
			perm.Description = &permission.Description.String
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
