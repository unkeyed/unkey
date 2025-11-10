package handler

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2KeysGetKeyRequestBody
	Response = openapi.V2KeysGetKeyResponseBody
)

// Handler implements zen.Route interface for the v2 keys.getKey endpoint
type Handler struct {
	Logger    logging.Logger
	DB        db.Database
	Keys      keys.KeyService
	Auditlogs auditlogs.AuditLogService
	Vault     *vault.Service
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/keys.getKey"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.getKey")

	// Authentication
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := db.Query.FindLiveKeyByID(ctx, h.DB.RO(), req.KeyId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.Wrap(
				err,
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key does not exist"),
				fault.Public("We could not find the requested key."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve Key information."),
		)
	}

	keyData := db.ToKeyData(key)

	// Validate key belongs to authorized workspace
	if keyData.Key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("key not found",
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("key belongs to different workspace"),
			fault.Public("The specified key was not found."),
		)
	}

	// Permission check
	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   keyData.Api.ID,
			Action:       rbac.ReadKey,
		}),
	)))
	if err != nil {
		return err
	}

	// Handle decryption if requested
	var plaintext string
	decrypt := ptr.SafeDeref(req.Decrypt, false)
	if decrypt {
		rawKey, err := h.decryptKey(ctx, auth, keyData)
		if err != nil {
			return err
		}
		plaintext = *rawKey
	}

	response := openapi.KeyResponseData{
		Meta:        nil,
		Ratelimits:  nil,
		Name:        keyData.Key.Name.String,
		UpdatedAt:   keyData.Key.UpdatedAtM.Int64,
		Credits:     nil,
		Expires:     0,
		Identity:    nil,
		Permissions: nil,
		Roles:       nil,
		CreatedAt:   keyData.Key.CreatedAtM,
		Enabled:     keyData.Key.Enabled,
		KeyId:       keyData.Key.ID,
		Start:       keyData.Key.Start,
		Plaintext:   plaintext,
	}

	if keyData.Key.Expires.Valid {
		response.Expires = keyData.Key.Expires.Time.UnixMilli()
	}

	// Set credits
	if keyData.Key.RemainingRequests.Valid {
		response.Credits = &openapi.KeyCreditsData{
			Refill:    nil,
			Remaining: nullable.NewNullableWithValue(int64(keyData.Key.RemainingRequests.Int32)),
		}

		if keyData.Key.RefillAmount.Valid {
			var refillDay int
			interval := openapi.KeyCreditsRefillIntervalDaily
			if keyData.Key.RefillDay.Valid {
				interval = openapi.KeyCreditsRefillIntervalMonthly
				refillDay = int(keyData.Key.RefillDay.Int16)
			}

			response.Credits.Refill = &openapi.KeyCreditsRefill{
				Amount:    int64(keyData.Key.RefillAmount.Int32),
				Interval:  interval,
				RefillDay: refillDay,
			}
		}
	}

	// Set identity
	if keyData.Identity != nil {
		response.Identity = &openapi.Identity{
			Meta:       nil,
			Ratelimits: nil,
			Id:         keyData.Identity.ID,
			ExternalId: keyData.Identity.ExternalID,
		}

		if len(keyData.Identity.Meta) > 0 {
			identityMeta, err := db.UnmarshalNullableJSONTo[map[string]any](keyData.Identity.Meta)
			response.Identity.Meta = identityMeta
			if err != nil {
				h.Logger.Error("failed to unmarshal identity meta", "error", err)
			}
		}
	}

	// Set permissions, combine direct + role permissions
	permissionSlugs := make(map[string]struct{})
	for _, p := range keyData.Permissions {
		permissionSlugs[p.Slug] = struct{}{}
	}
	for _, p := range keyData.RolePermissions {
		permissionSlugs[p.Slug] = struct{}{}
	}

	for slug := range permissionSlugs {
		response.Permissions = append(response.Permissions, slug)
	}

	for _, role := range keyData.Roles {
		response.Roles = append(response.Roles, role.Name)
	}

	// Set ratelimits
	if len(keyData.Ratelimits) > 0 {
		var keyRatelimits []openapi.RatelimitResponse
		var identityRatelimits []openapi.RatelimitResponse

		for _, rl := range keyData.Ratelimits {
			ratelimitResp := openapi.RatelimitResponse{
				Id:        rl.ID,
				Duration:  rl.Duration,
				Limit:     int64(rl.Limit),
				Name:      rl.Name,
				AutoApply: rl.AutoApply,
			}

			// Add to key ratelimits if it belongs to this key
			if rl.KeyID.Valid {
				keyRatelimits = append(keyRatelimits, ratelimitResp)
			}

			// Add to identity ratelimits if it has an identity_id that matches
			if rl.IdentityID.Valid {
				identityRatelimits = append(identityRatelimits, ratelimitResp)
			}
		}

		response.Ratelimits = keyRatelimits

		if response.Identity != nil {
			response.Identity.Ratelimits = identityRatelimits
		}
	}

	// Set meta
	meta, err := db.UnmarshalNullableJSONTo[map[string]any](keyData.Key.Meta.String)
	if err != nil {
		h.Logger.Error("failed to unmarshal key meta",
			"keyId", keyData.Key.ID,
			"error", err,
		)
	}

	response.Meta = meta

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: response,
	})
}

func (h *Handler) decryptKey(ctx context.Context, auth *keys.KeyVerifier, keyData *db.KeyData) (*string, error) {
	if h.Vault == nil {
		return nil, fault.New("vault missing",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Public("Vault hasn't been set up."),
		)
	}

	// Permission check for decryption
	err := auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.DecryptKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   keyData.Api.ID,
			Action:       rbac.DecryptKey,
		}),
	)))
	if err != nil {
		return nil, err
	}

	if !keyData.KeyAuth.StoreEncryptedKeys {
		return nil, fault.New("api not set up for key encryption",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("api not set up for key encryption"),
			fault.Public("The API for this key does not support key encryption."),
		)
	}

	// Only decrypt if the key is actually encrypted
	if !keyData.EncryptedKey.Valid || !keyData.EncryptionKeyID.Valid {
		return nil, nil //nolint:nilnil
	}

	decrypted, err := h.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
		Keyring:   keyData.Key.WorkspaceID,
		Encrypted: keyData.EncryptedKey.String,
	})
	if err != nil {
		h.Logger.Error("failed to decrypt key",
			"keyId", keyData.Key.ID,
			"error", err,
		)
		// Return nil instead of failing the entire request
		return nil, nil //nolint:nilnil
	}

	return ptr.P(decrypted.GetPlaintext()), nil
}
