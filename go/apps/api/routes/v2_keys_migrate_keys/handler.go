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
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	dbtype "github.com/unkeyed/unkey/go/pkg/db/types"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2KeysMigrateKeysRequestBody
	Response = openapi.V2KeysMigrateKeysResponseBody
)

const (
	ChunkSize = 1_000
)

type Handler struct {
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	ApiCache  cache.Cache[cache.ScopedKey, db.FindLiveApiByIDRow]
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.migrateKeys"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.migrateKeys")

	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

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

	api, hit, err := h.ApiCache.SWR(ctx, cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: req.ApiId}, func(ctx context.Context) (db.FindLiveApiByIDRow, error) {
		return db.Query.FindLiveApiByID(ctx, h.DB.RO(), req.ApiId)
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("api does not exist"),
				fault.Public("The requested API does not exist or has been deleted."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve API information."),
		)
	}

	if hit == cache.Null {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api not found"),
			fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	// Check if API belongs to the authorized workspace
	if api.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"),
			fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	migration, err := db.Query.FindKeyMigrationByID(ctx, h.DB.RO(), db.FindKeyMigrationByIDParams{ID: req.MigrationId, WorkspaceID: auth.AuthorizedWorkspaceID})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Migration.NotFound.URN()),
				fault.Internal("migration does not exist"),
				fault.Public("The requested Migration does not exist or has been deleted."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve migration information."),
		)
	}

	now := time.Now().UnixMilli()

	var hashes []string
	var identitiesToFind []string
	var permissionsToFind []string
	var rolesToFind []string

	var keysArray []db.InsertKeyParams
	var ratelimitsToInsert []db.InsertKeyRatelimitParams
	var identitiesToInsert []db.InsertIdentityParams
	var keyRolesToInsert []db.InsertKeyRoleParams
	var keyPermissionsToInsert []db.InsertKeyPermissionParams
	var rolesToInsert []db.InsertRoleParams
	var permissionsToInsert []db.InsertPermissionParams

	var keysToInsert = make(map[string]db.InsertKeyParams)
	var externalIdToIdentityId = make(map[string]*string)
	var permissionSlugToPermissionId = make(map[string]*string)
	var roleNameToRoleId = make(map[string]*string)

	var auditLogs []auditlog.AuditLog
	var failedHashes = make([]string, 0)

	for _, key := range req.Keys {
		hashes = append(hashes, key.Hash)
		name := ptr.SafeDeref(key.Name)

		newKey := db.InsertKeyParams{
			ID:                 uid.New(uid.KeyPrefix),
			Hash:               key.Hash,
			KeySpaceID:         api.KeyAuth.ID,
			Start:              "", // Unknown at this point
			WorkspaceID:        auth.AuthorizedWorkspaceID,
			Name:               sql.NullString{Valid: name != "", String: name},
			Meta:               sql.NullString{Valid: false, String: ""},
			PendingMigrationID: sql.NullString{Valid: true, String: migration.ID},
			ForWorkspaceID:     sql.NullString{Valid: false, String: ""},
			IdentityID:         sql.NullString{Valid: false, String: ""},
			Expires:            sql.NullTime{Valid: false, Time: time.Time{}},
			CreatedAtM:         now,
			Enabled:            ptr.SafeDeref(key.Enabled, true),
			RemainingRequests:  sql.NullInt32{Valid: false, Int32: 0},
			RefillDay:          sql.NullInt16{Valid: false, Int16: 0},
			RefillAmount:       sql.NullInt32{Valid: false, Int32: 0},
		} // nolint:exhaustruct

		if key.Meta != nil {
			metaBytes, marshalErr := json.Marshal(*key.Meta)
			if marshalErr != nil {
				return fault.Wrap(marshalErr,
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("failed to marshal meta"), fault.Public("Invalid metadata format."),
				)
			}

			newKey.Meta = sql.NullString{String: string(metaBytes), Valid: true}
		}

		if key.Expires != nil {
			newKey.Expires = sql.NullTime{Time: time.UnixMilli(*key.Expires), Valid: true}
		}

		if key.Credits != nil {
			if key.Credits.Remaining.IsSpecified() {
				newKey.RemainingRequests = sql.NullInt32{
					Int32: int32(key.Credits.Remaining.MustGet()), // nolint:gosec
					Valid: true,
				}
			}

			if key.Credits.Refill != nil {
				newKey.RefillAmount = sql.NullInt32{
					Int32: int32(key.Credits.Refill.Amount), // nolint:gosec
					Valid: true,
				}

				if key.Credits.Refill.Interval == openapi.KeyCreditsRefillIntervalMonthly {
					if key.Credits.Refill.RefillDay == 0 {
						return fault.New("missing refillDay",
							fault.Code(codes.App.Validation.InvalidInput.URN()),
							fault.Internal("refillDay required for monthly interval"),
							fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
						)
					}

					newKey.RefillDay = sql.NullInt16{
						Int16: int16(key.Credits.Refill.RefillDay), // nolint:gosec
						Valid: true,
					}
				}
			}
		}

		if key.ExternalId != nil {
			identitiesToFind = append(identitiesToFind, *key.ExternalId)

			externalIdToIdentityId[*key.ExternalId] = nil
		}

		if key.Permissions != nil {
			permissionsToFind = append(permissionsToFind, *key.Permissions...)

			for _, permission := range *key.Permissions {
				permissionSlugToPermissionId[permission] = nil
			}
		}

		if key.Roles != nil {
			rolesToFind = append(rolesToFind, *key.Roles...)

			for _, role := range *key.Roles {
				roleNameToRoleId[role] = nil
			}
		}

		// Any other data of the key will be set later down the line.
		keysToInsert[key.Hash] = newKey
	}

	hashes = deduplicate(hashes)
	identitiesToFind = deduplicate(identitiesToFind)
	permissionsToFind = deduplicate(permissionsToFind)
	rolesToFind = deduplicate(rolesToFind)

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		usedHashes, err := db.Query.FindKeysByHash(ctx, tx, hashes)
		if err != nil && !db.IsNotFound(err) {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to check for duplicate keys."),
			)
		}

		// Respond with that in the response, so the customer knows which keys were already in use.
		// and can contact us maybe we can get rid of them.
		for _, hash := range usedHashes {
			delete(keysToInsert, hash.Hash)
			failedHashes = append(failedHashes, hash.Hash)
		}

		if len(identitiesToFind) > 0 {
			identities, err := db.Query.FindIdentitiesByExternalId(ctx, tx, db.FindIdentitiesByExternalIdParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				ExternalIds: identitiesToFind,
				Deleted:     false,
			})
			if err != nil && !db.IsNotFound(err) {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to check for duplicate identities."),
				)
			}

			for _, identity := range identities {
				externalIdToIdentityId[identity.ExternalID] = &identity.ID
			}
		}

		if len(permissionsToFind) > 0 {
			permissions, err := db.Query.FindPermissionsBySlugs(ctx, tx, db.FindPermissionsBySlugsParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Slugs:       permissionsToFind,
			})
			if err != nil && !db.IsNotFound(err) {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to check for duplicate permissions."),
				)
			}

			for _, permission := range permissions {
				permissionSlugToPermissionId[permission.Slug] = &permission.ID
			}
		}

		if len(rolesToFind) > 0 {
			roles, err := db.Query.FindRolesByNames(ctx, tx, db.FindRolesByNamesParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Names:       rolesToFind,
			})
			if err != nil && !db.IsNotFound(err) {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to check for duplicate roles."),
				)
			}

			for _, role := range roles {
				roleNameToRoleId[role.Name] = &role.ID
			}
		}

		// We found the stuff that we could find, now we can just upsert everything that doesn't exist.
		for externalId, identityId := range externalIdToIdentityId {
			if identityId != nil {
				continue
			}

			id := uid.New(uid.IdentityPrefix)
			identitiesToInsert = append(identitiesToInsert, db.InsertIdentityParams{
				ID:          id,
				ExternalID:  externalId,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Environment: "default",
				CreatedAt:   now,
				Meta:        []byte("{}"),
			})

			externalIdToIdentityId[externalId] = &id
		}

		for slug, permissionId := range permissionSlugToPermissionId {
			if permissionId != nil {
				continue
			}

			id := uid.New(uid.PermissionPrefix)
			permissionsToInsert = append(permissionsToInsert, db.InsertPermissionParams{
				PermissionID: id,
				WorkspaceID:  auth.AuthorizedWorkspaceID,
				Name:         slug,
				Slug:         slug,
				Description:  dbtype.NullString{Valid: false, String: ""},
				CreatedAtM:   now,
			})

			permissionSlugToPermissionId[slug] = &id
		}

		for name, roleId := range roleNameToRoleId {
			if roleId != nil {
				continue
			}

			id := uid.New(uid.RolePrefix)
			rolesToInsert = append(rolesToInsert, db.InsertRoleParams{
				RoleID:      id,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Name:        name,
				Description: sql.NullString{Valid: false, String: ""},
				CreatedAt:   now,
			})

			roleNameToRoleId[name] = &id
		}

		// Now the fun begins.
		for _, key := range req.Keys {
			keyParams, ok := keysToInsert[key.Hash]
			if !ok {
				continue
			}

			if key.Ratelimits != nil {
				for _, ratelimit := range *key.Ratelimits {
					ratelimitsToInsert = append(ratelimitsToInsert, db.InsertKeyRatelimitParams{
						ID:          uid.New(uid.RatelimitPrefix),
						WorkspaceID: auth.AuthorizedWorkspaceID,
						KeyID:       sql.NullString{String: keyParams.ID, Valid: true},
						Name:        ratelimit.Name,
						Limit:       int32(ratelimit.Limit), // nolint:gosec
						Duration:    ratelimit.Duration,
						CreatedAt:   now,
						AutoApply:   ratelimit.AutoApply,
					})
				}
			}

			if key.ExternalId != nil {
				identityID, ok := externalIdToIdentityId[*key.ExternalId]
				if ok {
					keyParams.IdentityID = sql.NullString{Valid: true, String: *identityID}
				}
			}

			if key.Permissions != nil {
				for _, permission := range *key.Permissions {
					permissionID, ok := permissionSlugToPermissionId[permission]
					if !ok {
						continue
					}

					keyPermissionsToInsert = append(keyPermissionsToInsert, db.InsertKeyPermissionParams{
						KeyID:        keyParams.ID,
						PermissionID: *permissionID,
						WorkspaceID:  auth.AuthorizedWorkspaceID,
						CreatedAt:    now,
					})

					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID: auth.AuthorizedWorkspaceID,
						Event:       auditlog.AuthConnectPermissionKeyEvent,
						ActorType:   auditlog.RootKeyActor,
						ActorID:     auth.Key.ID,
						ActorName:   "root key",
						ActorMeta:   map[string]any{},
						Display:     fmt.Sprintf("Added permission %s to key %s", permission, keyParams.ID),
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Resources: []auditlog.AuditLogResource{
							{
								Type:        auditlog.KeyResourceType,
								ID:          keyParams.ID,
								Name:        keyParams.Name.String,
								DisplayName: keyParams.Name.String,
								Meta:        map[string]any{},
							},
							{
								Type:        auditlog.PermissionResourceType,
								ID:          *permissionID,
								Name:        permission,
								DisplayName: permission,
								Meta:        map[string]any{},
							},
						},
					})
				}
			}

			if key.Roles != nil {
				for _, role := range *key.Roles {
					roleID, ok := roleNameToRoleId[role]
					if !ok {
						continue
					}

					keyRolesToInsert = append(keyRolesToInsert, db.InsertKeyRoleParams{
						KeyID:       keyParams.ID,
						RoleID:      *roleID,
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
						Display:     fmt.Sprintf("Connected role %s to key %s", role, keyParams.ID),
						RemoteIP:    s.Location(),
						UserAgent:   s.UserAgent(),
						Resources: []auditlog.AuditLogResource{
							{
								Type:        auditlog.KeyResourceType,
								ID:          keyParams.ID,
								Name:        keyParams.Name.String,
								DisplayName: keyParams.Name.String,
								Meta:        map[string]any{},
							},
							{
								Type:        auditlog.RoleResourceType,
								ID:          *roleID,
								DisplayName: role,
								Name:        role,
								Meta:        map[string]any{},
							},
						},
					})
				}
			}

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.KeyCreateEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.Key.ID,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				Display:     fmt.Sprintf("Created key %s in migration %s", keyParams.ID, migration.ID),
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.KeyResourceType,
						ID:          keyParams.ID,
						DisplayName: keyParams.Name.String,
						Name:        keyParams.Name.String,
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

			keysArray = append(keysArray, keyParams)
		}

		if len(permissionsToInsert) > 0 {
			chunks := chunk(permissionsToInsert, ChunkSize)
			for _, chunk := range chunks {
				if err := db.BulkQuery.InsertPermissions(ctx, tx, chunk); err != nil {
					return err
				}
			}
		}

		if len(identitiesToInsert) > 0 {
			chunks := chunk(identitiesToInsert, ChunkSize)
			for _, chunk := range chunks {
				if err := db.BulkQuery.InsertIdentities(ctx, tx, chunk); err != nil {
					return err
				}
			}
		}

		if len(rolesToInsert) > 0 {
			chunks := chunk(rolesToInsert, ChunkSize)
			for _, chunk := range chunks {
				if err := db.BulkQuery.InsertRoles(ctx, tx, chunk); err != nil {
					return err
				}
			}
		}

		if len(keysArray) > 0 {
			chunks := chunk(keysArray, ChunkSize)
			for _, chunk := range chunks {
				if err := db.BulkQuery.InsertKeys(ctx, tx, chunk); err != nil {
					return err
				}
			}
		}

		if len(keyRolesToInsert) > 0 {
			chunks := chunk(keyRolesToInsert, ChunkSize)
			for _, chunk := range chunks {
				if err := db.BulkQuery.InsertKeyRoles(ctx, tx, chunk); err != nil {
					return err
				}
			}
		}

		if len(keyPermissionsToInsert) > 0 {
			chunks := chunk(keyPermissionsToInsert, ChunkSize)
			for _, chunk := range chunks {
				if err := db.BulkQuery.InsertKeyPermissions(ctx, tx, chunk); err != nil {
					return err
				}
			}
		}

		if len(ratelimitsToInsert) > 0 {
			chunks := chunk(ratelimitsToInsert, ChunkSize)
			for _, chunk := range chunks {
				if err := db.BulkQuery.InsertKeyRatelimits(ctx, tx, chunk); err != nil {
					return err
				}
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

	// Build the response with migrated keys and failed hashes
	migratedKeys := []openapi.V2KeysMigrateKeysMigration{}
	for _, key := range keysArray {
		migratedKeys = append(migratedKeys, openapi.V2KeysMigrateKeysMigration{
			Hash:  key.Hash,
			KeyId: key.ID,
		})
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2KeysMigrateKeysResponseData{
			Migrated: migratedKeys,
			Failed:   failedHashes,
		},
	})
}

func deduplicate[T comparable](items []T) []T {
	seen := make(map[T]bool)
	result := []T{}

	for _, item := range items {
		if !seen[item] {
			seen[item] = true
			result = append(result, item)
		}
	}

	return result
}

func chunk[T any](items []T, size int) [][]T {
	var chunks [][]T
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}

	return chunks
}
