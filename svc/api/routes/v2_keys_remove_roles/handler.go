package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2KeysRemoveRolesRequestBody
	Response = openapi.V2KeysRemoveRolesResponseBody
)

// Handler implements zen.Route interface for the v2 keys remove roles endpoint
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
	return "/v2/keys.removeRoles"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.removeRoles")

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

	currentRoleIDs := make(map[string]db.ListRolesByKeyIDRow)
	for _, role := range currentRoles {
		currentRoleIDs[role.ID] = role
	}

	foundRoles, err := db.Query.FindManyRolesByNamesWithPerms(ctx, h.DB.RO(), db.FindManyRolesByNamesWithPermsParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Names:       req.Roles,
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve current roles."),
		)
	}

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

	rolesToRemove := make([]db.FindManyRolesByNamesWithPermsRow, 0)
	for _, role := range foundRoles {
		_, exists := currentRoleIDs[role.ID]
		if !exists {
			continue
		}

		rolesToRemove = append(rolesToRemove, role)
		delete(currentRoleIDs, role.ID)
	}

	if len(rolesToRemove) > 0 {
		err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			var auditLogs []auditlog.AuditLog
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

	responseData := make(openapi.V2KeysRemoveRolesResponseData, 0)
	for _, role := range currentRoleIDs {
		r := openapi.Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			Permissions: nil,
		}

		rolePermissions, err := db.UnmarshalNullableJSONTo[[]db.Permission](role.Permissions)
		if err != nil {
			h.Logger.Error("Failed to unmarshal role permissions", "error", err)
		}

		for _, permission := range rolePermissions {
			perm := openapi.Permission{
				Id:          permission.ID,
				Name:        permission.Name,
				Slug:        permission.Slug,
				Description: permission.Description.String,
			}

			r.Permissions = append(r.Permissions, perm)
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
