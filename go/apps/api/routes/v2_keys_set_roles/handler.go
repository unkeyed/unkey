package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysSetRolesRequestBody
type Response = openapi.V2KeysSetRolesResponseBody

// Handler implements zen.Route interface for the v2 keys set roles endpoint
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
	return "/v2/keys.setRoles"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.setRoles")

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
			fault.Internal("key belongs to different workspace"), fault.Public("The specified key was not found."),
		)
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
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
	)))
	if err != nil {
		return err
	}

	currentRoles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve current roles."),
		)
	}

	currentRoleIDs := make(map[string]bool)
	for _, role := range currentRoles {
		currentRoleIDs[role.ID] = true
	}

	foundRoles, err := db.Query.FindManyRolesByNamesWithPerms(ctx, h.DB.RO(), db.FindManyRolesByNamesWithPermsParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Names:       req.Roles,
	})

	foundMap := make(map[string]struct{})
	for _, role := range foundRoles {
		foundMap[role.ID] = struct{}{}
		foundMap[role.Name] = struct{}{}
	}

	for _, role := range req.Roles {
		_, exists := foundMap[role]
		if !exists {
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Public(fmt.Sprintf("Role '%s' was not found.", role)),
			)
		}
	}

	requestedRoleIDs := make(map[string]bool)
	requestedRoleMap := make(map[string]db.FindManyRolesByNamesWithPermsRow)
	for _, role := range foundRoles {
		requestedRoleIDs[role.ID] = true
		requestedRoleMap[role.ID] = role
	}

	// Determine roles to remove and add
	rolesToRemove := make([]db.ListRolesByKeyIDRow, 0)
	for _, role := range currentRoles {
		if !requestedRoleIDs[role.ID] {
			rolesToRemove = append(rolesToRemove, role)
		}
	}

	rolesToAdd := make([]db.FindManyRolesByNamesWithPermsRow, 0)
	for _, role := range foundRoles {
		if !currentRoleIDs[role.ID] {
			rolesToAdd = append(rolesToAdd, role)
		}
	}

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		var auditLogs []auditlog.AuditLog

		if len(rolesToRemove) > 0 {
			var roleIds []string

			for _, role := range rolesToRemove {
				roleIds = append(roleIds, role.ID)

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.AuthDisconnectRoleKeyEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Removed role %s from key %s", role.Name, req.KeyId),
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
							Type:        auditlog.RoleResourceType,
							ID:          role.ID,
							Name:        role.Name,
							DisplayName: role.Name,
							Meta:        map[string]any{},
						},
					},
				})
			}

			err = db.Query.DeleteManyKeyRolesByKeyAndRoleIDs(ctx, tx, db.DeleteManyKeyRolesByKeyAndRoleIDsParams{
				KeyID:   req.KeyId,
				RoleIds: roleIds,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to remove role assignment."),
				)
			}
		}

		if len(rolesToAdd) > 0 {
			var keyRolesToInsert []db.InsertKeyRoleParams

			for _, role := range rolesToAdd {
				keyRolesToInsert = append(keyRolesToInsert, db.InsertKeyRoleParams{
					KeyID:       req.KeyId,
					RoleID:      role.ID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					CreatedAtM:  time.Now().UnixMilli(),
				})

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.AuthConnectRoleKeyEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Added role %s to key %s", role.Name, req.KeyId),
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
							Type:        auditlog.RoleResourceType,
							ID:          role.ID,
							Name:        role.Name,
							DisplayName: role.Name,
							Meta:        map[string]any{},
						},
					},
				})
			}

			err = db.BulkQuery.InsertKeyRoles(ctx, tx, keyRolesToInsert)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to add role assignment."),
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

	responseData := make(openapi.V2KeysSetRolesResponseData, 0)
	for _, role := range foundRoles {
		r := openapi.Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: nil,
			Permissions: nil,
		}

		if role.Description.Valid {
			r.Description = &role.Description.String
		}

		rolePermissions := make([]db.Permission, 0)
		if permBytes, ok := role.Permissions.([]byte); ok && permBytes != nil {
			// AIDEV-SAFETY: On JSON parse failure, we default to empty permissions list
			// to maintain least-privilege security posture rather than failing open
			if err := json.Unmarshal(permBytes, &rolePermissions); err != nil {
				h.Logger.Debug("failed to parse role permissions JSON, defaulting to empty list",
					"roleId", role.ID,
					"rawBytes", string(permBytes),
					"error", err.Error())
			}
		}

		perms := make([]openapi.Permission, 0)
		for _, permission := range rolePermissions {
			perm := openapi.Permission{
				Id:          permission.ID,
				Name:        permission.Name,
				Slug:        permission.Slug,
				Description: nil,
			}

			if permission.Description.Valid {
				perm.Description = &permission.Description.String
			}

			perms = append(perms, perm)
		}

		if len(perms) > 0 {
			r.Permissions = ptr.P(perms)
		}

		responseData = append(responseData, r)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
	})
}
