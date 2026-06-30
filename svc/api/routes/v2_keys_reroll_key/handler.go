package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
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
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/retry"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
)

type (
	Request  = openapi.V2KeysRerollKeyRequestBody
	Response = openapi.V2KeysRerollKeyResponseBody
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
	return "/v2/keys.rerollKey"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Public("The specified key was not found."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve key."),
		)
	}

	// Validate key belongs to authorized workspace
	if key.WorkspaceID != principal.WorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	// Portal sessions are scoped to a single external identity and may only
	// reroll keys belonging to that identity. Mismatches return 404 so we never
	// leak the existence of keys owned by another externalId.
	//
	// Identity scoping is intentionally separate from the RBAC permission system.
	// Permissions gate what operations a principal can perform; identity scoping
	// gates which keys are reachable. Portal sessions carry a fixed externalId.
	switch src := principal.Source.(type) {
	case authprincipal.PortalSessionSource:
		// Fail closed: a portal session without an externalId can't be scoped.
		if src.ExternalID == "" {
			return fault.New("portal session missing identity",
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("portal session externalId is empty"),
				fault.Public("An internal error occurred."),
			)
		}

		if !key.IdentityExternalID.Valid || key.IdentityExternalID.String != src.ExternalID {
			return fault.New("key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key identity externalId does not match portal session"),
				fault.Public("The specified key was not found."),
			)
		}
	}

	// Portal-authenticated rerolls are attributed to a portalEndUser actor so
	// customers can see end-user activity in their audit logs. The actor
	// metadata records which end user acted.
	actor := auditlog.ActorFromPrincipal(principal)

	keyData := db.ToKeyData(key)

	checks := rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.Api.ID,
			Action:       rbac.CreateKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.CreateKey,
		}),
	)

	if keyData.EncryptionKeyID.Valid {
		checks = rbac.And(
			checks,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   key.Api.ID,
					Action:       rbac.EncryptKey,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.EncryptKey,
				}),
			),
		)
	}

	err = principal.Authorize(checks)
	if err != nil {
		return err
	}

	length := 16
	prefix := ""

	split := strings.Split(key.Start, "_")
	if len(split) > 1 {
		prefix = split[0]
	}

	if prefix == "" && key.KeyAuth.DefaultPrefix.Valid {
		prefix = key.KeyAuth.DefaultPrefix.String
	}

	if key.KeyAuth.DefaultBytes.Valid {
		length = int(key.KeyAuth.DefaultBytes.Int32)
	}

	// keyID is assigned at the start of each retry attempt below; on a
	// duplicate-entry collision on the keys.id unique index we regenerate
	// it and retry. The DB is the source of truth for uniqueness.
	var keyID string
	keyResult, err := h.Keys.CreateKey(ctx, keys.CreateKeyRequest{
		Prefix:     prefix,
		ByteLength: length,
	})
	if err != nil {
		return err
	}

	var encryption *vaultv1.EncryptResponse
	if keyData.EncryptionKeyID.Valid {
		if h.Vault == nil {
			return fault.New("vault missing",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Public("Vault hasn't been set up."),
			)
		}

		if !key.KeyAuth.StoreEncryptedKeys {
			return fault.New("api not set up for key encryption",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("api not set up for key encryption"), fault.Public("This API does not support key encryption."),
			)
		}

		encryption, err = h.Vault.Encrypt(ctx, &vaultv1.EncryptRequest{
			Keyring: key.WorkspaceID,
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

	err = retry.New(
		retry.Attempts(5),
		retry.ShouldRetry(db.IsDuplicateKeyError),
	).DoContext(ctx, func() error {
		// Fresh keyID per attempt so a duplicate-entry collision on the
		// keys.id unique index can be recovered by regenerating the ID.
		keyID = uid.New(uid.KeyPrefix)
		return db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			err = db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
				ID:                 keyID,
				KeySpaceID:         key.KeyAuthID,
				Hash:               keyResult.Hash,
				Start:              keyResult.Start,
				WorkspaceID:        key.WorkspaceID,
				ForWorkspaceID:     key.ForWorkspaceID,
				CreatedAtM:         now,
				Enabled:            key.Enabled,
				RemainingRequests:  key.RemainingRequests,
				RefillDay:          key.RefillDay,
				RefillAmount:       key.RefillAmount,
				Name:               key.Name,
				IdentityID:         key.IdentityID,
				Meta:               key.Meta,
				Expires:            key.Expires,
				PendingMigrationID: sql.NullString{Valid: false, String: ""},
			})
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

			if len(keyData.Ratelimits) > 0 {
				ratelimitsToInsert := make([]db.InsertKeyRatelimitParams, 0)

				for _, ratelimit := range keyData.Ratelimits {
					if ratelimit.IdentityID.Valid {
						continue
					}

					ratelimitsToInsert = append(ratelimitsToInsert, db.InsertKeyRatelimitParams{
						ID:          uid.New(uid.RatelimitPrefix),
						WorkspaceID: key.WorkspaceID,
						KeyID:       sql.NullString{String: keyID, Valid: true},
						Name:        ratelimit.Name,
						Limit:       ratelimit.Limit,
						Duration:    ratelimit.Duration,
						AutoApply:   ratelimit.AutoApply,
						CreatedAt:   now,
						UpdatedAt:   sql.NullInt64{Int64: 0, Valid: false},
					})
				}

				if len(ratelimitsToInsert) > 0 {
					err = db.BulkQuery.InsertKeyRatelimits(ctx, tx, ratelimitsToInsert)
					if err != nil {
						return fault.Wrap(err,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"), fault.Public("Failed to create rate limits."),
						)
					}
				}
			}

			if len(keyData.Roles) > 0 {
				rolesToInsert := make([]db.InsertKeyRoleParams, 0, len(keyData.Roles))
				for _, role := range keyData.Roles {
					rolesToInsert = append(rolesToInsert, db.InsertKeyRoleParams{
						WorkspaceID: key.WorkspaceID,
						KeyID:       keyID,
						RoleID:      role.ID,
						CreatedAtM:  now,
					})
				}

				err = db.BulkQuery.InsertKeyRoles(ctx, tx, rolesToInsert)
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to create key role."),
					)
				}
			}

			if len(keyData.Permissions) > 0 {
				permissionsToInsert := make([]db.InsertKeyPermissionParams, 0, len(keyData.Permissions))
				for _, permission := range keyData.Permissions {
					permissionsToInsert = append(permissionsToInsert, db.InsertKeyPermissionParams{
						WorkspaceID:  key.WorkspaceID,
						KeyID:        keyID,
						PermissionID: permission.ID,
						CreatedAt:    now,
						UpdatedAt:    sql.NullInt64{Int64: 0, Valid: false},
					})
				}

				err = db.BulkQuery.InsertKeyPermissions(ctx, tx, permissionsToInsert)
				if err != nil {
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to create key permission."),
					)
				}
			}

			// Calculate the desired expiry time (rounded up to next minute)
			expiration := time.Now().Add(time.Millisecond * time.Duration(req.Expiration))
			// Round up to next minute (ceil)
			if expiration.Truncate(time.Minute) != expiration {
				expiration = expiration.Truncate(time.Minute).Add(time.Minute)
			}

			if req.Expiration == 0 {
				expiration = time.Now()
			}

			//nolint: exhaustruct
			err = db.Query.UpdateKey(ctx, tx, db.UpdateKeyParams{
				ID:               req.KeyId,
				ExpiresSpecified: 1,
				Expires: sql.NullTime{
					Time:  expiration,
					Valid: true,
				},
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"),
					fault.Public("Failed to expire old key."),
				)
			}

			var auditLogs []auditlog.AuditLog
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.KeyRerollEvent,
				ActorType:     actor.Type,
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     actor.Meta,
				Display:       fmt.Sprintf("Rerolled key (%s) to (%s)", req.KeyId, keyID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.KeyResourceType,
						ID:          keyID,
						DisplayName: key.Name.String,
						Name:        key.Name.String,
						Meta:        map[string]any{},
					},
					{
						Type:        auditlog.KeyResourceType,
						ID:          req.KeyId,
						DisplayName: key.Name.String,
						Name:        key.Name.String,
						Meta:        map[string]any{},
					},
					{
						Type:        auditlog.APIResourceType,
						ID:          key.Api.ID,
						DisplayName: key.Api.Name,
						Name:        key.Api.Name,
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
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2KeysRerollKeyResponseData{
			KeyId: keyID,
			Key:   keyResult.Key,
		},
	})
}
