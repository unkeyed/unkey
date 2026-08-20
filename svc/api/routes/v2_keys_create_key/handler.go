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

	"github.com/unkeyed/unkey/gen/rpc/vault"
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
	"github.com/unkeyed/unkey/svc/api/internal/projects"
)

type (
	Request  = openapi.V2KeysCreateKeyRequestBody
	Response = openapi.V2KeysCreateKeyResponseBody
)

type Handler struct {
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Vault     vault.VaultServiceClient
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
	// Mint a correlation ID for this user action so the dashboard can drill
	// from any one of the audit events (key.create + N permission binds + N
	// role binds) to the rest. Nested helpers that call Auditlogs.Insert
	// pick this up via auditlog.CorrelationFrom(ctx).
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	// 1. Authentication
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	apiAndKeySpace, err := db.Query.FindApiAndKeySpaceByID(ctx, h.DB.RO(), req.ApiId)
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
	if apiAndKeySpace.WorkspaceID != principal.WorkspaceID {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api belongs to different workspace"), fault.Public("The specified API was not found."),
		)
	}

	// 3. Permission check. Creating a key is authorized against the keyspace
	// because the key does not exist until after the operation succeeds. The
	// tuple legs accept legacy API-scoped grants until those are migrated.
	err = principal.Authorize(rbac.Or(
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
		rbac.U(
			urn.New().Workspace(principal.WorkspaceID).Keyspace(apiAndKeySpace.KeyAuthID.String),
			permissions.CreateKey{},
		),
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Api.NotFound.URN(),
			"The specified API was not found.",
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

	if apiAndKeySpace.KeySpaceID == "" {
		return fault.New("api not set up for keys",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("api not set up for keys, keyspace not found"), fault.Public("The requested API is not set up to handle keys."),
		)
	}

	// Per-request values win; otherwise fall back to the keyspace
	// defaults (`default_prefix` / `default_bytes` configured in the
	// dashboard, persisted on key_auth). Without these fallbacks the
	// columns round-trip through the DB for nothing.
	var prefix string
	switch {
	case req.Prefix != nil:
		prefix = *req.Prefix
	case apiAndKeySpace.DefaultPrefix.Valid:
		prefix = apiAndKeySpace.DefaultPrefix.String
	}

	byteLength := ptr.SafeDeref(req.ByteLength)
	if byteLength == 0 && apiAndKeySpace.DefaultBytes.Valid {
		byteLength = int(apiAndKeySpace.DefaultBytes.Int32)
	}
	if byteLength == 0 {
		byteLength = 16
	}

	// keyID is assigned at the start of each retry attempt below; on a
	// duplicate-entry collision on the keys.id unique index we regenerate
	// it and retry. The DB is the source of truth for uniqueness.
	var keyID string
	keyResult, err := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     prefix,
		ByteLength: byteLength,
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

		err = principal.Authorize(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.EncryptKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   apiAndKeySpace.ApiID,
				Action:       rbac.EncryptKey,
			}),
			rbac.U(urn.New().Workspace(principal.WorkspaceID).Keyspace(apiAndKeySpace.KeySpaceID).Key("*"), permissions.EncryptKey{}),
		))
		if err != nil {
			return apierrors.MaskInsufficientPermissionsAsNotFound(
				err,
				codes.Data.Api.NotFound.URN(),
				"The specified API was not found.",
			)
		}

		if !apiAndKeySpace.StoreEncryptedKeys {
			return fault.New("api not set up for key encryption",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("api not set up for key encryption"), fault.Public("This API does not support key encryption."),
			)
		}

		encryption, err = h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: principal.WorkspaceID,
			Data:    keyResult.Key,
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("vault error"), fault.Public("Failed to encrypt key in vault."),
			)
		}
	}

	if req.Credits != nil && req.Credits.Refill != nil {
		if !req.Credits.Remaining.IsSpecified() || req.Credits.Remaining.IsNull() {
			return fault.New("missing credits.remaining",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("credits.remaining required when refill is set"),
				fault.Public("`credits.remaining` must be provided when `credits.refill` is set."),
			)
		}
		if req.Credits.Refill.Interval == openapi.KeyCreditsRefillIntervalMonthly && req.Credits.Refill.RefillDay == 0 {
			return fault.New("missing refillDay",
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("refillDay required for monthly interval"),
				fault.Public("`refillDay` must be provided when the refill interval is `monthly`."),
			)
		}
	}

	now := time.Now().UnixMilli()

	// Creates without identity, permission, or role lookups have no
	// data-dependent writes after authorization. Send all their writes and the
	// commit in one MySQL multi-statement query.
	auditPreparer, canPrepareAudit := h.Auditlogs.(auditlogs.OutboxPreparer)
	canBatch := canPrepareAudit &&
		req.ExternalId == nil &&
		(req.Permissions == nil || len(*req.Permissions) == 0) &&
		(req.Roles == nil || len(*req.Roles) == 0)
	if canBatch {
		batchErr := retry.New(
			retry.Attempts(5),
			retry.ShouldRetry(db.IsDuplicateKeyError),
		).DoContext(ctx, func() error {
			// A duplicate response proves the INSERT failed before COMMIT, so it is
			// safe to regenerate IDs. Never retry an ambiguous transport failure.
			keyID = uid.New(uid.KeyPrefix)
			insertKey := db.InsertKeyParams{
				ID:                 keyID,
				KeySpaceID:         apiAndKeySpace.KeyAuthID.String,
				Hash:               keyResult.Hash,
				Start:              keyResult.Start,
				WorkspaceID:        principal.WorkspaceID,
				ForWorkspaceID:     sql.NullString{},
				Name:               sql.NullString{},
				IdentityID:         sql.NullString{},
				Meta:               sql.NullString{},
				Expires:            sql.NullTime{},
				CreatedAtM:         now,
				Enabled:            true,
				RemainingRequests:  sql.NullInt64{},
				RefillDay:          sql.NullInt16{},
				RefillAmount:       sql.NullInt64{},
				PendingMigrationID: sql.NullString{},
			}
			if req.Name != nil {
				insertKey.Name = sql.NullString{String: *req.Name, Valid: true}
			}
			if req.Meta != nil {
				meta, marshalErr := json.Marshal(*req.Meta)
				if marshalErr != nil {
					return fault.Wrap(marshalErr,
						fault.Code(codes.App.Validation.InvalidInput.URN()),
						fault.Internal("failed to marshal meta"), fault.Public("Invalid metadata format."),
					)
				}
				insertKey.Meta = sql.NullString{String: string(meta), Valid: true}
			}
			if req.Expires != nil {
				insertKey.Expires = sql.NullTime{Time: time.UnixMilli(*req.Expires), Valid: true}
			}
			if req.Credits != nil {
				if req.Credits.Remaining.IsSpecified() && !req.Credits.Remaining.IsNull() {
					insertKey.RemainingRequests = sql.NullInt64{Int64: req.Credits.Remaining.MustGet(), Valid: true}
				}
				if req.Credits.Refill != nil {
					insertKey.RefillAmount = sql.NullInt64{Int64: req.Credits.Refill.Amount, Valid: true}
					if req.Credits.Refill.Interval == openapi.KeyCreditsRefillIntervalMonthly {
						insertKey.RefillDay = sql.NullInt16{Int16: int16(req.Credits.Refill.RefillDay), Valid: true} //nolint:gosec
					}
				}
			}
			if req.Enabled != nil {
				insertKey.Enabled = *req.Enabled
			}

			var encryptionParams *db.InsertKeyEncryptionParams
			if encryption != nil {
				encryptionParams = &db.InsertKeyEncryptionParams{
					WorkspaceID:     principal.WorkspaceID,
					KeyID:           keyID,
					Encrypted:       encryption.GetEncrypted(),
					EncryptionKeyID: encryption.GetKeyId(),
					CreatedAt:       now,
				}
			}

			ratelimits := []db.InsertKeyRatelimitParams{}
			if req.Ratelimits != nil {
				ratelimits = make([]db.InsertKeyRatelimitParams, len(*req.Ratelimits))
				for i, ratelimit := range *req.Ratelimits {
					ratelimits[i] = db.InsertKeyRatelimitParams{
						ID:          uid.New(uid.RatelimitPrefix),
						WorkspaceID: principal.WorkspaceID,
						KeyID:       sql.NullString{String: keyID, Valid: true},
						Name:        ratelimit.Name,
						Limit:       uint64(ratelimit.Limit),
						Duration:    uint64(ratelimit.Duration),
						CreatedAt:   now,
						UpdatedAt:   sql.NullInt64{},
						AutoApply:   ratelimit.AutoApply,
					}
				}
			}
			outboxRows, prepareErr := auditPreparer.PrepareOutboxRows(ctx, []auditlog.AuditLog{{
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
						Type:        auditlog.APIResourceType,
						ID:          req.ApiId,
						DisplayName: apiAndKeySpace.ApiName,
						Name:        apiAndKeySpace.ApiName,
						Meta:        map[string]any{},
					},
				},
			}})
			if prepareErr != nil {
				return prepareErr
			}
			if len(outboxRows) != 1 {
				return fault.New("invalid audit outbox batch size", fault.Internal("create key batch requires exactly one audit outbox row"))
			}

			return h.DB.BatchRW().CreateKeyWithAuditBatch(ctx, insertKey, encryptionParams, ratelimits, outboxRows[0])
		})
		if batchErr != nil {
			return fault.Wrap(batchErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to create key."),
			)
		}

		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{RequestId: s.RequestID()},
			Data: openapi.V2KeysCreateKeyResponseData{
				KeyId: keyID,
				Key:   keyResult.Key,
			},
		})
	}

	txErr := retry.New(
		retry.Attempts(5),
		retry.ShouldRetry(func(err error) bool {
			return db.IsDuplicateKeyError(err) || db.IsTransientError(err)
		}),
	).DoContext(ctx, func() error {
		// Fresh keyID per attempt so a duplicate-entry collision on the
		// keys.id unique index can be recovered by regenerating the ID.
		keyID = uid.New(uid.KeyPrefix)
		return db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			projectID := ""
			attemptIdentityID := sql.NullString{}
			existingPermissionsBySlug := make(map[string]db.Permission)
			existingRolesByName := make(map[string]db.FindRolesByNamesRow)
			permissionSlugs := []string{}
			if req.Permissions != nil {
				permissionSlugs = *req.Permissions
			}
			roleNames := []string{}
			if req.Roles != nil {
				roleNames = *req.Roles
			}
			findIdentity := int64(0)
			externalID := ""
			if req.ExternalId != nil {
				findIdentity = 1
				externalID = *req.ExternalId
			}
			if findIdentity == 1 || len(permissionSlugs) > 0 || len(roleNames) > 0 {
				resources, findErr := db.Query.FindKeyMutationResources(ctx, tx, db.FindKeyMutationResourcesParams{
					FindIdentity:    findIdentity,
					WorkspaceID:     principal.WorkspaceID,
					ExternalID:      externalID,
					PermissionSlugs: permissionSlugs,
					RoleNames:       roleNames,
				})
				if findErr != nil {
					return fault.Wrap(findErr,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("failed to find key mutation resources"),
						fault.Public("Failed to retrieve key resources."),
					)
				}
				for _, resource := range resources {
					switch resource.ResourceType {
					case "identity":
						attemptIdentityID = sql.NullString{String: resource.IdentityID, Valid: true}
					case "permission":
						existingPermissionsBySlug[resource.PermissionSlug] = db.Permission{ID: resource.PermissionID, Slug: resource.PermissionSlug} //nolint:exhaustruct
					case "role":
						existingRolesByName[resource.RoleName] = db.FindRolesByNamesRow{ID: resource.RoleID, Name: resource.RoleName}
					case "project":
						projectID = resource.ProjectID
					}
				}
			}

			insertKeyParams := db.InsertKeyParams{
				ID:                 keyID,
				KeySpaceID:         apiAndKeySpace.KeyAuthID.String,
				Hash:               keyResult.Hash,
				Start:              keyResult.Start,
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

				if !attemptIdentityID.Valid {
					if projectID == "" {
						projectID, err = projects.EnsureDefaultProject(ctx, tx, principal.WorkspaceID)
						if err != nil {
							return err
						}
					}

					identityID := uid.New(uid.IdentityPrefix)
					err = db.Query.InsertIdentity(ctx, tx, db.InsertIdentityParams{
						ID:          identityID,
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
							fault.Internal("failed to insert identity"),
							fault.Public("Failed to create identity."),
						)
					}
					attemptIdentityID = sql.NullString{Valid: true, String: identityID}
				}

				insertKeyParams.IdentityID = attemptIdentityID
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
			if req.Permissions != nil && len(*req.Permissions) > 0 {
				requestedPermissions := []db.Permission{}
				permissionSlugsToCreate := []string{}

				for _, requestedSlug := range *req.Permissions {
					existingPerm, exists := existingPermissionsBySlug[requestedSlug]
					if exists {
						requestedPermissions = append(requestedPermissions, existingPerm)
						continue
					}

					permissionSlugsToCreate = append(permissionSlugsToCreate, requestedSlug)
				}

				if len(permissionSlugsToCreate) > 0 && projectID == "" {
					projectID, err = projects.EnsureDefaultProject(ctx, tx, principal.WorkspaceID)
					if err != nil {
						return err
					}
				}

				permissionsToCreate := make([]db.InsertPermissionParams, 0, len(permissionSlugsToCreate))
				for _, permissionSlug := range permissionSlugsToCreate {
					newPermID := uid.New(uid.PermissionPrefix)
					permissionsToCreate = append(permissionsToCreate, db.InsertPermissionParams{
						PermissionID: newPermID,
						WorkspaceID:  principal.WorkspaceID,
						ProjectID:    projectID,
						Name:         permissionSlug,
						Slug:         permissionSlug,
						Description:  dbtype.NullString{String: "", Valid: false},
						CreatedAtM:   now,
					})

					requestedPermissions = append(requestedPermissions, db.Permission{
						Pk:          0, // only here to make the linter happy
						ID:          newPermID,
						Name:        permissionSlug,
						Slug:        permissionSlug,
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

			if req.Roles != nil && len(*req.Roles) > 0 {
				// Create missing roles in bulk and build final list
				requestedRoles := []db.FindRolesByNamesRow{}

				for _, requestedName := range *req.Roles {
					existingRole, exists := existingRolesByName[requestedName]
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
						Type:        auditlog.APIResourceType,
						ID:          req.ApiId,
						DisplayName: apiAndKeySpace.ApiName,
						Name:        apiAndKeySpace.ApiName,
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
