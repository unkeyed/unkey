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
	"github.com/unkeyed/unkey/go/internal/services/permissions"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysCreateKeyRequestBody
type Response = openapi.V2KeysCreateKeyResponseBody

type Handler struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.createKey"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.createKey")

	// 1. Authentication
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// 3. Permission check
	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   req.ApiId,
				Action:       rbac.CreateKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.CreateKey,
			}),
		),
	)
	if err != nil {
		return err
	}

	// 4. Validate API exists and belongs to workspace
	api, err := db.Query.FindApiByID(ctx, h.DB.RO(), req.ApiId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("api not found",
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("api not found"), fault.Public("The specified API was not found."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve API."),
		)
	}

	// Validate API belongs to authorized workspace
	if api.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api belongs to different workspace"), fault.Public("The specified API was not found."),
		)
	}

	// 5. Generate key using key service
	keyID := uid.New(uid.KeyPrefix)
	keyResult, err := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     ptr.SafeDeref(req.Prefix),
		ByteLength: ptr.SafeDeref(req.ByteLength, 16),
	})
	if err != nil {
		return err
	}

	now := time.Now().UnixMilli()

	// 6. Resolve permissions if provided
	var resolvedPermissions []db.Permission
	if req.Permissions != nil {
		for _, permName := range *req.Permissions {
			permission, findErr := db.Query.FindPermissionByNameAndWorkspaceID(ctx, h.DB.RO(), db.FindPermissionByNameAndWorkspaceIDParams{
				Name:        permName,
				WorkspaceID: auth.AuthorizedWorkspaceID,
			})
			if findErr != nil {
				if db.IsNotFound(findErr) {
					return fault.New("permission not found",
						fault.Code(codes.Data.Permission.NotFound.URN()),
						fault.Internal("permission not found"), fault.Public(fmt.Sprintf("Permission '%s' was not found.", permName)),
					)
				}
				return fault.Wrap(findErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve permission."),
				)
			}
			resolvedPermissions = append(resolvedPermissions, permission)
		}
	}

	// 7. Resolve roles if provided
	var resolvedRoles []db.Role
	if req.Roles != nil {
		for _, roleName := range *req.Roles {
			role, findErr := db.Query.FindRoleByNameAndWorkspaceID(ctx, h.DB.RO(), db.FindRoleByNameAndWorkspaceIDParams{
				Name:        roleName,
				WorkspaceID: auth.AuthorizedWorkspaceID,
			})
			if findErr != nil {
				if db.IsNotFound(findErr) {
					return fault.New("role not found",
						fault.Code(codes.Data.Role.NotFound.URN()),
						fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role '%s' was not found.", roleName)),
					)
				}
				return fault.Wrap(findErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve role."),
				)
			}
			resolvedRoles = append(resolvedRoles, role)
		}
	}

	// 8. Execute all database operations in a transaction
	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		// 9. Insert the key
		insertKeyParams := db.InsertKeyParams{
			ID:                keyID,
			KeyringID:         api.KeyAuthID.String,
			Hash:              keyResult.Hash,
			Start:             keyResult.Start,
			WorkspaceID:       auth.AuthorizedWorkspaceID,
			ForWorkspaceID:    sql.NullString{String: "", Valid: false},
			CreatedAtM:        now,
			Enabled:           true,
			RemainingRequests: sql.NullInt32{Int32: 0, Valid: false},
			Name:              sql.NullString{String: "", Valid: false},
			IdentityID:        sql.NullString{String: "", Valid: false},
			Meta:              sql.NullString{String: "", Valid: false},
			Expires:           sql.NullTime{Time: time.Time{}, Valid: false},
			RatelimitAsync:    sql.NullBool{Bool: false, Valid: false},
			RatelimitLimit:    sql.NullInt32{Int32: 0, Valid: false},
			RatelimitDuration: sql.NullInt64{Int64: 0, Valid: false},
			Environment:       sql.NullString{String: "", Valid: false},
		}

		// Set optional fields
		if req.Name != nil {
			insertKeyParams.Name = sql.NullString{String: *req.Name, Valid: true}
		}

		// Note: owner_id is set to null in the SQL query, so we skip setting it here

		if req.Meta != nil {
			metaBytes, marshalErr := json.Marshal(*req.Meta)
			if marshalErr != nil {
				return fault.Wrap(marshalErr,
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("failed to marshal meta"), fault.Public("Invalid metadata format."),
				)
			}
			insertKeyParams.Meta = sql.NullString{String: string(metaBytes), Valid: true}
		}

		if req.Expires != nil {
			insertKeyParams.Expires = sql.NullTime{Time: time.UnixMilli(*req.Expires), Valid: true}
		}

		if req.Credits != nil {
			insertKeyParams.RemainingRequests = sql.NullInt32{
				Int32: int32(req.Credits.Remaining), // nolint:gosec
				Valid: true,
			}
		}

		// Set enabled status (default true)
		if req.Enabled != nil {
			insertKeyParams.Enabled = *req.Enabled
		}

		err = db.Query.InsertKey(ctx, tx, insertKeyParams)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to create key."),
			)
		}

		// 10. Handle rate limits if provided
		if req.Ratelimits != nil {
			for _, ratelimit := range *req.Ratelimits {
				ratelimitID := uid.New(uid.RatelimitPrefix)
				err = db.Query.InsertKeyRatelimit(ctx, tx, db.InsertKeyRatelimitParams{
					ID:          ratelimitID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					KeyID:       sql.NullString{String: keyID, Valid: true},
					Name:        ratelimit.Name,
					Limit:       int32(ratelimit.Limit), // nolint:gosec
					Duration:    int64(ratelimit.Duration),
					CreatedAt:   now,
				})
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to create rate limit."),
					)
				}
			}
		}

		// 11. Handle permissions if provided
		var auditLogs []auditlog.AuditLog
		for _, permission := range resolvedPermissions {
			err = db.Query.InsertKeyPermission(ctx, tx, db.InsertKeyPermissionParams{
				KeyID:        keyID,
				PermissionID: permission.ID,
				WorkspaceID:  auth.AuthorizedWorkspaceID,
				CreatedAt:    now,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to assign permission."),
				)
			}

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.AuthConnectPermissionKeyEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     fmt.Sprintf("Added permission %s to key %s", permission.Name, keyID),
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        "key",
						ID:          keyID,
						Name:        insertKeyParams.Name.String,
						DisplayName: insertKeyParams.Name.String,
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

		// 12. Handle roles if provided
		for _, role := range resolvedRoles {
			err = db.Query.InsertKeyRole(ctx, tx, db.InsertKeyRoleParams{
				KeyID:       keyID,
				RoleID:      role.ID,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				CreatedAtM:  now,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to assign role."),
				)
			}

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.AuthConnectRoleKeyEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     fmt.Sprintf("Connected role %s to key %s", role.Name, keyID),
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        "key",
						ID:          keyID,
						DisplayName: insertKeyParams.Name.String,
						Name:        insertKeyParams.Name.String,
						Meta:        map[string]any{},
					},
					{
						Type:        "role",
						ID:          role.ID,
						DisplayName: role.Name,
						Name:        role.Name,
						Meta:        map[string]any{},
					},
				},
			})
		}

		// 13. Create main audit log for key creation
		auditLogs = append(auditLogs, auditlog.AuditLog{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       auditlog.KeyCreateEvent,
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.KeyID,
			ActorName:   "root key",
			ActorMeta:   map[string]any{},
			Display:     fmt.Sprintf("Created key %s", keyID),
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
			Resources: []auditlog.AuditLogResource{
				{
					Type:        "key",
					ID:          keyID,
					DisplayName: keyID,
					Name:        keyID,
					Meta:        map[string]any{},
				},
				{
					Type:        "api",
					ID:          req.ApiId,
					DisplayName: api.Name,
					Name:        api.Name,
					Meta:        map[string]any{},
				},
			},
		})

		// 14. Insert audit logs
		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("audit log error"), fault.Public("Failed to create audit log."),
			)
		}

		return nil
	})
	if err != nil {
		return err
	}

	// 16. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.KeysCreateKeyResponseData{
			KeyId: keyID,
			Key:   keyResult.Key,
		},
	})
}
