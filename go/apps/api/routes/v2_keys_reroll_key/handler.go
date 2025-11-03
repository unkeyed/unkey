package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"

	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2KeysRerollKeyRequestBody
	Response = openapi.V2KeysRerollKeyResponseBody
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
	return "/v2/keys.rerollKey"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.rerollKey")

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

	keyData := db.ToKeyData(key, h.Logger)

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

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(checks))
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

	keyID := uid.New(uid.KeyPrefix)
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

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.InsertKey(ctx, tx, db.InsertKeyParams{
			ID:                keyID,
			KeySpaceID:        key.KeyAuthID,
			Hash:              keyResult.Hash,
			Start:             keyResult.Start,
			WorkspaceID:       key.WorkspaceID,
			ForWorkspaceID:    key.ForWorkspaceID,
			CreatedAtM:        now,
			Enabled:           key.Enabled,
			RemainingRequests: key.RemainingRequests,
			RefillDay:         key.RefillDay,
			RefillAmount:      key.RefillAmount,
			Name:              key.Name,
			IdentityID:        key.IdentityID,
			Meta:              key.Meta,
			Expires:           key.Expires,
		})
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
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Event:       auditlog.KeyRerollEvent,
			ActorType:   auditlog.RootKeyActor,
			ActorID:     auth.Key.ID,
			ActorName:   "root key",
			ActorMeta:   map[string]any{},
			Display:     fmt.Sprintf("Rerolled key (%s) to (%s)", req.KeyId, keyID),
			RemoteIP:    s.Location(),
			UserAgent:   s.UserAgent(),
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
