package handler

import (
	"context"
	"database/sql"
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
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysAddRolesRequestBody
type Response = openapi.V2KeysAddRolesResponseBody

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
	return "/v2/keys.addRoles"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.addRoles")

	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
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

	// Validate key belongs to authorized workspace
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
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Rbac,
				ResourceID:   "*",
				Action:       rbac.AddRoleToKey,
			}),
		),
	))
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

	foundMap := make(map[string]db.FindManyRolesByNamesWithPermsRow)
	for _, role := range foundRoles {
		foundMap[role.ID] = role
		foundMap[role.Name] = role
	}

	for _, role := range req.Roles {
		_, ok := foundMap[role]
		if ok {
			continue
		}

		return fault.New("role not found",
			fault.Code(codes.Data.Role.NotFound.URN()),
			fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role %q was not found.", role)),
		)
	}

	// 7. Determine which roles to add (only add roles that aren't already assigned)
	currentRoleIDs := make(map[string]bool)
	for _, role := range currentRoles {
		currentRoleIDs[role.ID] = true
	}

	rolesToAdd := make([]db.FindManyRolesByNamesWithPermsRow, 0)
	for _, role := range foundRoles {
		if !currentRoleIDs[role.ID] {
			rolesToAdd = append(rolesToAdd, role)
		}
	}

	if len(rolesToAdd) > 0 {
		err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			var auditLogs []auditlog.AuditLog
			rolesToInsert := make([]db.InsertKeyRoleParams, 0)

			for _, role := range rolesToAdd {
				rolesToInsert = append(rolesToInsert, db.InsertKeyRoleParams{
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

			err = db.BulkQuery.InsertKeyRoles(ctx, tx, rolesToInsert)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to assign roles."),
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

	responseData := make(openapi.V2KeysAddRolesResponseData, 0)
	// Wrap row so we don't have to do the same logic twice.
	for _, role := range rolesToAdd {
		row := db.ListRolesByKeyIDRow{
			ID:          role.ID,
			WorkspaceID: role.WorkspaceID,
			Name:        role.Name,
			Description: role.Description,
			CreatedAtM:  role.CreatedAtM,
			UpdatedAtM:  role.UpdatedAtM,
			Permissions: role.Permissions,
		}

		currentRoles = append(currentRoles, row)
	}

	for _, role := range currentRoles {
		r := openapi.Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: nil,
		}

		if role.Description.Valid {
			r.Description = &role.Description.String
		}

		rolePermissions := make([]db.Permission, 0)
		json.Unmarshal(role.Permissions.([]byte), &rolePermissions)
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
