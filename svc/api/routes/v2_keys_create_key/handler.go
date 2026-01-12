package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/svc/api/openapi"

	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/zen"
)

type (
	Request  = openapi.V2KeysCreateKeyRequestBody
	Response = openapi.V2KeysCreateKeyResponseBody
)

type Handler struct {
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Vault     *vault.Service
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
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// 3. Permission check
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
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
	)))
	if err != nil {
		return err
	}

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

	if api.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api belongs to different workspace"), fault.Public("The specified API was not found."),
		)
	}

	keySpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), api.KeyAuthID.String)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("api not set up for keys",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("api not set up for keys, keyspace not found"), fault.Public("The requested API is not set up to handle keys."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve API information."),
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

	encrypt := ptr.SafeDeref(req.Recoverable, false)
	var encryption *vaultv1.EncryptResponse
	if encrypt {
		if h.Vault == nil {
			return fault.New("vault missing",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Public("Vault hasn't been set up."),
			)
		}

		err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.EncryptKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   api.ID,
				Action:       rbac.EncryptKey,
			}),
		)))
		if err != nil {
			return err
		}

		if !keySpace.StoreEncryptedKeys {
			return fault.New("api not set up for key encryption",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("api not set up for key encryption"), fault.Public("This API does not support key encryption."),
			)
		}

		encryption, err = h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: s.AuthorizedWorkspaceID(),
			Data:    keyResult.Key,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("vault error"), fault.Public("Failed to encrypt key in vault."),
			)
		}
	}

	now := time.Now().UnixMilli()

	txErr := db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		insertKeyParams := db.InsertKeyParams{
			ID:                 keyID,
			KeySpaceID:         api.KeyAuthID.String,
			Hash:               keyResult.Hash,
			Start:              keyResult.Start,
			WorkspaceID:        auth.AuthorizedWorkspaceID,
			ForWorkspaceID:     sql.NullString{String: "", Valid: false},
			CreatedAtM:         now,
			Enabled:            true,
			RemainingRequests:  sql.NullInt32{Int32: 0, Valid: false},
			RefillDay:          sql.NullInt16{Int16: 0, Valid: false},
			RefillAmount:       sql.NullInt32{Int32: 0, Valid: false},
			Name:               sql.NullString{String: "", Valid: false},
			IdentityID:         sql.NullString{String: "", Valid: false},
			Meta:               sql.NullString{String: "", Valid: false},
			Expires:            sql.NullTime{Time: time.Time{}, Valid: false},
			PendingMigrationID: sql.NullString{Valid: false, String: ""},
		}

		// Set optional fields
		if req.Name != nil {
			insertKeyParams.Name = sql.NullString{String: *req.Name, Valid: true}
		}

		// Handle identity creation/lookup from externalId
		if req.ExternalId != nil {
			externalID := *req.ExternalId

			// Upsert identity - inserts if not exists, no-op if exists
			err = db.Query.UpsertIdentity(ctx, tx, db.UpsertIdentityParams{
				ID:          uid.New(uid.IdentityPrefix),
				ExternalID:  externalID,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Environment: "default",
				CreatedAt:   now,
				Meta:        []byte("{}"),
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("failed to upsert identity"),
					fault.Public("Failed to create identity."),
				)
			}

			// Fetch the identity ID (either just created or already existed)
			identity, err := db.Query.FindIdentityByExternalID(ctx, tx, db.FindIdentityByExternalIDParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				ExternalID:  externalID,
				Deleted:     false,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("failed to find identity after upsert"),
					fault.Public("Failed to find identity."),
				)
			}

			insertKeyParams.IdentityID = sql.NullString{Valid: true, String: identity.ID}
		}

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
			// If refill is set, remaining must be specified and not null
			if req.Credits.Refill != nil {
				if !req.Credits.Remaining.IsSpecified() || req.Credits.Remaining.IsNull() {
					return fault.New("missing credits.remaining",
						fault.Code(codes.App.Validation.InvalidInput.URN()),
						fault.Internal("credits.remaining required when refill is set"),
						fault.Public("`credits.remaining` must be provided when `credits.refill` is set."),
					)
				}
			}

			if req.Credits.Remaining.IsSpecified() && !req.Credits.Remaining.IsNull() {
				insertKeyParams.RemainingRequests = sql.NullInt32{
					Int32: int32(req.Credits.Remaining.MustGet()), // nolint:gosec
					Valid: true,
				}
			}

			if req.Credits.Refill != nil {
				insertKeyParams.RefillAmount = sql.NullInt32{
					Int32: int32(req.Credits.Refill.Amount), // nolint:gosec
					Valid: true,
				}

				if req.Credits.Refill.Interval == openapi.KeyCreditsRefillIntervalMonthly {
					// 0 is the zero value of int16
					if req.Credits.Refill.RefillDay == 0 {
						return fault.New("missing refillDay",
							fault.Code(codes.App.Validation.InvalidInput.URN()),
							fault.Internal("refillDay required for monthly interval"),
							fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
						)
					}

					insertKeyParams.RefillDay = sql.NullInt16{
						Int16: int16(req.Credits.Refill.RefillDay), // nolint:gosec
						Valid: true,
					}
				}
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

		if encryption != nil {
			err = db.Query.InsertKeyEncryption(ctx, tx, db.InsertKeyEncryptionParams{
				WorkspaceID:     auth.AuthorizedWorkspaceID,
				KeyID:           keyID,
				CreatedAt:       now,
				Encrypted:       encryption.GetEncrypted(),
				EncryptionKeyID: encryption.GetKeyId(),
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to create key encryption."),
				)
			}
		}

		if req.Ratelimits != nil && len(*req.Ratelimits) > 0 {
			ratelimitsToInsert := make([]db.InsertKeyRatelimitParams, len(*req.Ratelimits))
			for i, ratelimit := range *req.Ratelimits {
				ratelimitID := uid.New(uid.RatelimitPrefix)
				ratelimitsToInsert[i] = db.InsertKeyRatelimitParams{
					ID:          ratelimitID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					KeyID:       sql.NullString{String: keyID, Valid: true},
					Name:        ratelimit.Name,
					Limit:       int32(ratelimit.Limit), // nolint:gosec
					Duration:    ratelimit.Duration,
					CreatedAt:   now,
					UpdatedAt:   sql.NullInt64{Int64: 0, Valid: false},
					AutoApply:   ratelimit.AutoApply,
				}
			}

			err = db.BulkQuery.InsertKeyRatelimits(ctx, tx, ratelimitsToInsert)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to create rate limit."),
				)
			}
		}

		var auditLogs []auditlog.AuditLog
		if req.Permissions != nil {
			var existingPermissions []db.Permission
			existingPermissions, err = db.Query.FindPermissionsBySlugs(ctx, tx, db.FindPermissionsBySlugsParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Slugs:       *req.Permissions,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to retrieve permissions."),
				)
			}

			existingPermMap := make(map[string]db.Permission)
			for _, p := range existingPermissions {
				existingPermMap[p.Slug] = p
			}

			permissionsToCreate := []db.InsertPermissionParams{}
			requestedPermissions := []db.Permission{}

			for _, requestedSlug := range *req.Permissions {
				existingPerm, exists := existingPermMap[requestedSlug]
				if exists {
					requestedPermissions = append(requestedPermissions, existingPerm)
					continue
				}

				newPermID := uid.New(uid.PermissionPrefix)
				permissionsToCreate = append(permissionsToCreate, db.InsertPermissionParams{
					PermissionID: newPermID,
					WorkspaceID:  auth.AuthorizedWorkspaceID,
					Name:         requestedSlug,
					Slug:         requestedSlug,
					Description:  dbtype.NullString{String: "", Valid: false},
					CreatedAtM:   now,
				})

				requestedPermissions = append(requestedPermissions, db.Permission{
					Pk:          0, // only here to make the linter happy
					ID:          newPermID,
					Name:        requestedSlug,
					Slug:        requestedSlug,
					CreatedAtM:  now,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Description: dbtype.NullString{String: "", Valid: false},
					UpdatedAtM:  sql.NullInt64{Int64: 0, Valid: false},
				})
			}

			if len(permissionsToCreate) > 0 {
				err = db.BulkQuery.InsertPermissions(ctx, tx, permissionsToCreate)
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to create permissions."),
					)
				}
			}

			permissionsToInsert := []db.InsertKeyPermissionParams{}
			for _, reqPerm := range requestedPermissions {
				permissionsToInsert = append(permissionsToInsert, db.InsertKeyPermissionParams{
					KeyID:        keyID,
					PermissionID: reqPerm.ID,
					WorkspaceID:  auth.AuthorizedWorkspaceID,
					CreatedAt:    now,
					UpdatedAt:    sql.NullInt64{Valid: false, Int64: 0},
				})

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.AuthConnectPermissionKeyEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Added permission %s to key %s", reqPerm.Slug, keyID),
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type:        auditlog.KeyResourceType,
							ID:          keyID,
							Name:        insertKeyParams.Name.String,
							DisplayName: insertKeyParams.Name.String,
							Meta:        map[string]any{},
						},
						{
							Type:        auditlog.PermissionResourceType,
							ID:          reqPerm.ID,
							Name:        reqPerm.Slug,
							DisplayName: reqPerm.Slug,
							Meta:        map[string]any{},
						},
					},
				})
			}

			if len(permissionsToInsert) > 0 {
				err = db.BulkQuery.InsertKeyPermissions(ctx, tx, permissionsToInsert)
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to assign permissions."),
					)
				}
			}
		}

		if req.Roles != nil {
			var existingRoles []db.FindRolesByNamesRow
			existingRoles, err = db.Query.FindRolesByNames(ctx, tx, db.FindRolesByNamesParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Names:       *req.Roles,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to retrieve roles."),
				)
			}

			// Find which roles need to be created
			existingRoleMap := make(map[string]db.FindRolesByNamesRow)
			for _, r := range existingRoles {
				existingRoleMap[r.Name] = r
			}

			// Create missing roles in bulk and build final list
			requestedRoles := []db.FindRolesByNamesRow{}

			for _, requestedName := range *req.Roles {
				existingRole, exists := existingRoleMap[requestedName]
				if exists {
					requestedRoles = append(requestedRoles, existingRole)
					continue
				}

				return fault.New("role not found",
					fault.Code(codes.Data.Role.NotFound.URN()),
					fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role '%s' was not found.", requestedName)),
				)
			}

			// Insert all requested roles
			rolesToInsert := []db.InsertKeyRoleParams{}
			for _, reqRole := range requestedRoles {
				rolesToInsert = append(rolesToInsert, db.InsertKeyRoleParams{
					KeyID:       keyID,
					RoleID:      reqRole.ID,
					WorkspaceID: auth.AuthorizedWorkspaceID,
					CreatedAtM:  now,
				})

				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Event:       auditlog.AuthConnectRoleKeyEvent,
					ActorType:   auditlog.RootKeyActor,
					ActorID:     auth.Key.ID,
					ActorName:   "root key",
					ActorMeta:   map[string]any{},
					Display:     fmt.Sprintf("Connected role %s to key %s", reqRole.Name, keyID),
					RemoteIP:    s.Location(),
					UserAgent:   s.UserAgent(),
					Resources: []auditlog.AuditLogResource{
						{
							Type:        auditlog.KeyResourceType,
							ID:          keyID,
							DisplayName: insertKeyParams.Name.String,
							Name:        insertKeyParams.Name.String,
							Meta:        map[string]any{},
						},
						{
							Type:        auditlog.RoleResourceType,
							ID:          reqRole.ID,
							DisplayName: reqRole.Name,
							Name:        reqRole.Name,
							Meta:        map[string]any{},
						},
					},
				})
			}

			if len(rolesToInsert) > 0 {
				err = db.BulkQuery.InsertKeyRoles(ctx, tx, rolesToInsert)
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"),
						fault.Public("Failed to assign roles."),
					)
				}
			}
		}

		auditLogs = append(auditLogs, auditlog.AuditLog{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       auditlog.KeyCreateEvent,
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.Key.ID,
			ActorName:   "root key",
			ActorMeta:   map[string]any{},
			Display:     fmt.Sprintf("Created key %s", keyID),
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
			Resources: []auditlog.AuditLogResource{
				{
					Type:        auditlog.KeyResourceType,
					ID:          keyID,
					DisplayName: keyID,
					Name:        keyID,
					Meta:        map[string]any{},
				},
				{
					Type:        auditlog.APIResourceType,
					ID:          req.ApiId,
					DisplayName: api.Name,
					Name:        api.Name,
					Meta:        map[string]any{},
				},
			},
		})

		err = h.Auditlogs.Insert(ctx, tx, auditLogs)
		if err != nil {
			return err
		}

		return nil
	})

	if txErr != nil {
		return txErr
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2KeysCreateKeyResponseData{
			KeyId: keyID,
			Key:   keyResult.Key,
		},
	})
}
