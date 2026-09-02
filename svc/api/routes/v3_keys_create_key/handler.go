package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/svc/api/openapi"

	"github.com/unkeyed/unkey/pkg/auditlog"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/auditactor"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
)

type (
	Request  = openapi.V3KeysCreateKeyRequestBody
	Response = openapi.V2KeysCreateKeyResponseBody
)

// Handler creates keys through the v3 API route.
type Handler struct {
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Vault     vault.VaultServiceClient
}

// Method returns the HTTP method this route responds to.
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches.
func (h *Handler) Path() string {
	return "/v3/keys.createKey"
}

// Handle resolves the keyspace and creates its key.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// Use one correlation ID for all audit logs from this request.
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	keySpace, err := db.Query.FindKeySpaceByID(ctx, h.DB.RO(), req.Keyspace)
	if err != nil {
		if db.IsNotFound(err) {
			return keyspaceNotFound("keyspace not found")
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve the keyspace."),
		)
	}

	if keySpace.WorkspaceID != principal.WorkspaceID {
		return keyspaceNotFound("keyspace belongs to a different workspace")
	}
	if keySpace.DeletedAtM.Valid {
		return keyspaceNotFound("keyspace is deleted")
	}

	liveAPIs, err := db.Query.FindApisByKeyAuthIds(ctx, h.DB.RO(), db.FindApisByKeyAuthIdsParams{
		WorkspaceID: principal.WorkspaceID,
		KeyAuthIds:  []string{keySpace.ID},
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve the keyspace."),
		)
	}
	if len(liveAPIs) == 0 {
		return keyspaceNotFound("keyspace does not belong to a live API")
	}

	err = principal.Authorize(rbac.U(
		urn.New().Workspace(principal.WorkspaceID).Project(keySpace.ProjectID).Keyspace(keySpace.ID).Key("*"),
		permissions.Write,
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.KeyAuth.NotFound.URN(),
			"The specified keyspace was not found.",
		)
	}

	// Portal sessions are scoped to a single external identity. Force the
	// externalId on the request so the created key is always owned by the
	// session's identity, regardless of what the client sends.
	//
	// Portal-authenticated actions are attributed to a portalEndUser actor so
	// customers can see end-user activity in their audit logs.
	switch src := principal.Source.(type) {
	case authprincipal.PortalSessionSource:
		if src.ExternalID == "" {
			return fault.New("portal session missing identity",
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("portal session externalId is empty"),
				fault.Public("An internal error occurred."),
			)
		}
		req.ExternalId = &src.ExternalID
	}
	actor := auditactor.FromPrincipal(principal)

	var prefix string
	switch {
	case req.Prefix != nil:
		prefix = *req.Prefix
	case keySpace.DefaultPrefix.Valid:
		prefix = keySpace.DefaultPrefix.String
	}

	// keyID is assigned at the start of each retry attempt below; on a
	// duplicate-entry collision on the keys.id unique index we regenerate
	// it and retry. The DB is the source of truth for uniqueness.
	var keyID string
	keyResult, err := h.Keys.CreateKeyV1(ctx, keys.CreateKeyV1Request{Prefix: prefix})
	if err != nil {
		return err
	}
	keyValue := keyResult.Key

	encrypt := ptr.SafeDeref(req.Recoverable, false)
	var encryption *vaultv1.EncryptResponse
	if encrypt {
		if h.Vault == nil {
			return fault.New("vault missing",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Public("Vault hasn't been set up."),
			)
		}

		if !keySpace.StoreEncryptedKeys {
			return fault.New("keyspace does not support key recovery",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("keyspace does not store encrypted keys"),
				fault.Public("This keyspace does not support key recovery."),
			)
		}

		encryption, err = h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: principal.WorkspaceID,
			Data:    keyValue,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("vault error"), fault.Public("Failed to encrypt key in vault."),
			)
		}
	}

	projectID := keySpace.ProjectID
	now := time.Now().UnixMilli()

	txErr := retry.New(
		retry.Attempts(5),
		retry.ShouldRetry(db.IsDuplicateKeyError),
	).DoContext(ctx, func() error {
		// Fresh keyID per attempt so a duplicate-entry collision on the
		// keys.id unique index can be recovered by regenerating the ID.
		keyID = uid.New(uid.KeyPrefix)
		return db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			insertKeyParams := db.InsertKeyParams{
				ID:                 keyID,
				KeySpaceID:         keySpace.ID,
				Hash:               keyResult.Hash,
				Prefix:             keyResult.Prefix,
				Start:              keyResult.Start,
				End:                keyResult.End,
				WorkspaceID:        principal.WorkspaceID,
				ForWorkspaceID:     sql.NullString{String: "", Valid: false},
				CreatedAtM:         now,
				Enabled:            true,
				RemainingRequests:  sql.NullInt64{Int64: 0, Valid: false},
				RefillDay:          sql.NullInt16{Int16: 0, Valid: false},
				RefillAmount:       sql.NullInt64{Int64: 0, Valid: false},
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
					WorkspaceID: principal.WorkspaceID,
					ProjectID:   projectID,
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
					WorkspaceID: principal.WorkspaceID,
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
					insertKeyParams.RemainingRequests = sql.NullInt64{
						Int64: req.Credits.Remaining.MustGet(),
						Valid: true,
					}
				}

				if req.Credits.Refill != nil {
					insertKeyParams.RefillAmount = sql.NullInt64{
						Int64: req.Credits.Refill.Amount,
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
					WorkspaceID:     principal.WorkspaceID,
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
						WorkspaceID: principal.WorkspaceID,
						KeyID:       sql.NullString{String: keyID, Valid: true},
						Name:        ratelimit.Name,
						Limit:       uint64(ratelimit.Limit),
						Duration:    uint64(ratelimit.Duration),
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
					WorkspaceID: principal.WorkspaceID,
					ProjectID:   projectID,
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
						WorkspaceID:  principal.WorkspaceID,
						ProjectID:    projectID,
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
						WorkspaceID: principal.WorkspaceID,
						ProjectID:   projectID,
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
						WorkspaceID:  principal.WorkspaceID,
						CreatedAt:    now,
						UpdatedAt:    sql.NullInt64{Valid: false, Int64: 0},
					})

					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID:   principal.WorkspaceID,
						Event:         auditlog.AuthConnectPermissionKeyEvent,
						ActorType:     actor.Type,
						ActorID:       actor.ID,
						ActorName:     actor.Name,
						ActorMeta:     actor.Meta,
						Display:       fmt.Sprintf("Added permission %s to key %s", reqPerm.Slug, keyID),
						RemoteIP:      s.Location(),
						UserAgent:     s.UserAgent(),
						CorrelationID: "",
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
					WorkspaceID: principal.WorkspaceID,
					ProjectID:   projectID,
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
						WorkspaceID: principal.WorkspaceID,
						CreatedAtM:  now,
					})

					auditLogs = append(auditLogs, auditlog.AuditLog{
						WorkspaceID:   principal.WorkspaceID,
						Event:         auditlog.AuthConnectRoleKeyEvent,
						ActorType:     actor.Type,
						ActorID:       actor.ID,
						ActorName:     actor.Name,
						ActorMeta:     actor.Meta,
						Display:       fmt.Sprintf("Connected role %s to key %s", reqRole.Name, keyID),
						RemoteIP:      s.Location(),
						UserAgent:     s.UserAgent(),
						CorrelationID: "",
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
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.KeyCreateEvent,
				ActorType:     actor.Type,
				ActorID:       actor.ID,
				ActorName:     actor.Name,
				ActorMeta:     actor.Meta,
				Display:       fmt.Sprintf("Created key %s", keyID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.KeyResourceType,
						ID:          keyID,
						DisplayName: keyID,
						Name:        keyID,
						Meta:        map[string]any{},
					},
					{
						Type:        auditlog.KeySpaceResourceType,
						ID:          keySpace.ID,
						DisplayName: "",
						Name:        "",
						Meta:        nil,
					},
				},
			})

			err = h.Auditlogs.Insert(ctx, tx, auditLogs)
			if err != nil {
				return err
			}

			return nil
		})
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
			Key:   keyValue,
		},
	})
}

// keyspaceNotFound hides whether a keyspace does not exist or belongs to a different workspace.
func keyspaceNotFound(internal string) error {
	return fault.New("keyspace not found",
		fault.Code(codes.Data.KeyAuth.NotFound.URN()),
		fault.Internal(internal),
		fault.Public("The specified keyspace was not found."),
	)
}
