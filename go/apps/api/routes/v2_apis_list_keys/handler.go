package handler

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/cache"
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
	Request  = openapi.V2ApisListKeysRequestBody
	Response = openapi.V2ApisListKeysResponseBody
)

// Handler implements zen.Route interface for the v2 APIs list keys endpoint
type Handler struct {
	Logger   logging.Logger
	DB       db.Database
	Keys     keys.KeyService
	Vault    *vault.Service
	ApiCache cache.Cache[cache.ScopedKey, db.FindLiveApiByIDRow]
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/apis.listKeys"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
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
		rbac.And(
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.ReadKey,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   req.ApiId,
					Action:       rbac.ReadKey,
				}),
			),
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.ReadAPI,
				}),
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Api,
					ResourceID:   req.ApiId,
					Action:       rbac.ReadAPI,
				}),
			),
		),
	)))
	if err != nil {
		return err
	}

	api, hit, err := h.ApiCache.SWR(ctx, cache.ScopedKey{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Key:         req.ApiId,
	}, func(ctx context.Context) (db.FindLiveApiByIDRow, error) {
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
			fault.Internal("api not found"), fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	// Check if API belongs to the authorized workspace
	if api.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"), fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	if ptr.SafeDeref(req.Decrypt, false) {
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
				Action:       rbac.DecryptKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   api.ID,
				Action:       rbac.DecryptKey,
			}),
		)))
		if err != nil {
			return err
		}

		if !api.KeyAuth.StoreEncryptedKeys {
			return fault.New("api not set up for key encryption",
				fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
				fault.Internal("api not set up for key encryption"), fault.Public("The requested API does not support key encryption."),
			)
		}
	}

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")

	var identityFilter string
	if req.ExternalId != nil && *req.ExternalId != "" {
		identityFilter = *req.ExternalId
	}

	// Query keys by key_auth_id instead of api_id
	keyResults, err := db.Query.ListLiveKeysByKeySpaceID(
		ctx,
		h.DB.RO(),
		db.ListLiveKeysByKeySpaceIDParams{
			KeySpaceID: api.KeyAuthID.String,
			IDCursor:   cursor,
			Identity:   identityFilter,
			Limit:      int32(limit + 1), // nolint:gosec
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve keys."),
		)
	}

	// Handle pagination
	hasMore := len(keyResults) > limit
	var nextCursor *string
	if hasMore {
		nextCursor = ptr.P(keyResults[len(keyResults)-1].ID)
		keyResults = keyResults[:limit]
	}

	if len(keyResults) == 0 {
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: []openapi.KeyResponseData{},
			Pagination: &openapi.Pagination{
				Cursor:  nextCursor,
				HasMore: hasMore,
			},
		})
	}

	// Handle decryption if requested
	plaintextMap := make(map[string]string)
	if req.Decrypt != nil && *req.Decrypt {
		for _, key := range keyResults {
			if key.EncryptedKey.Valid && key.EncryptionKeyID.Valid {
				decrypted, decryptErr := h.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
					Keyring:   key.WorkspaceID,
					Encrypted: key.EncryptedKey.String,
				})

				if decryptErr != nil {
					h.Logger.Error("failed to decrypt key",
						"keyId", key.ID,
						"error", decryptErr,
					)
					continue
				}
				plaintextMap[key.ID] = decrypted.GetPlaintext()
			}
		}
	}

	// Transform to response format
	responseData := make([]openapi.KeyResponseData, len(keyResults))
	for i, key := range keyResults {
		keyData := db.ToKeyData(key)
		response := h.buildKeyResponseData(keyData, plaintextMap[key.ID])
		responseData[i] = response
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responseData,
		Pagination: &openapi.Pagination{
			Cursor:  nextCursor,
			HasMore: hasMore,
		},
	})
}

// buildKeyResponseData transforms internal key data into API response format.
func (h *Handler) buildKeyResponseData(keyData *db.KeyData, plaintext string) openapi.KeyResponseData {
	response := openapi.KeyResponseData{
		Meta:        nil,
		Ratelimits:  nil,
		Name:        nil,
		UpdatedAt:   nil,
		Credits:     nil,
		Expires:     nil,
		Identity:    nil,
		Permissions: nil,
		Roles:       nil,
		Plaintext:   nil,
		CreatedAt:   keyData.Key.CreatedAtM,
		Enabled:     keyData.Key.Enabled,
		KeyId:       keyData.Key.ID,
		Start:       keyData.Key.Start,
	}

	if plaintext != "" {
		response.Plaintext = ptr.P(plaintext)
	}

	// Set optional fields
	if keyData.Key.Name.Valid {
		response.Name = ptr.P(keyData.Key.Name.String)
	}

	if keyData.Key.UpdatedAtM.Valid {
		response.UpdatedAt = ptr.P(keyData.Key.UpdatedAtM.Int64)
	}

	if keyData.Key.Expires.Valid {
		response.Expires = ptr.P(keyData.Key.Expires.Time.UnixMilli())
	}

	// Set credits
	if keyData.KeyCredits != nil {
		response.Credits = &openapi.Credits{
			Refill:    nil,
			Remaining: nullable.NewNullableWithValue(int64(keyData.KeyCredits.Remaining)),
		}

		if keyData.KeyCredits.RefillAmount.Valid {
			var refillDay *int
			interval := openapi.Daily
			if keyData.KeyCredits.RefillDay.Valid {
				interval = openapi.Monthly
				refillDay = ptr.P(int(keyData.KeyCredits.RefillDay.Int16))
			}

			response.Credits.Refill = &openapi.CreditsRefill{
				Amount:    int64(keyData.KeyCredits.RefillAmount.Int32),
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
			var identityMeta map[string]any
			_ = json.Unmarshal(keyData.Identity.Meta, &identityMeta) // Ignore error, default to nil
			if identityMeta != nil {
				response.Identity.Meta = &identityMeta
			}
		}

		// Add identity credits if they exist
		if keyData.IdentityCredits != nil {
			response.Identity.Credits = &openapi.Credits{
				Remaining: nullable.NewNullableWithValue(int64(keyData.IdentityCredits.Remaining)),
			}

			if keyData.IdentityCredits.RefillAmount.Valid {
				var refillDay *int
				interval := openapi.Daily
				if keyData.IdentityCredits.RefillDay.Valid {
					interval = openapi.Monthly
					refillDay = ptr.P(int(keyData.IdentityCredits.RefillDay.Int16))
				}

				response.Identity.Credits.Refill = &openapi.CreditsRefill{
					Amount:    int64(keyData.IdentityCredits.RefillAmount.Int32),
					Interval:  interval,
					RefillDay: refillDay,
				}
			}
		}
	}

	// Set permissions (combine direct + role permissions)
	permissionSlugs := make(map[string]struct{})
	for _, p := range keyData.Permissions {
		permissionSlugs[p.Slug] = struct{}{}
	}
	for _, p := range keyData.RolePermissions {
		permissionSlugs[p.Slug] = struct{}{}
	}
	if len(permissionSlugs) > 0 {
		slugs := make([]string, 0, len(permissionSlugs))
		for slug := range permissionSlugs {
			slugs = append(slugs, slug)
		}
		response.Permissions = &slugs
	}

	// Set roles
	if len(keyData.Roles) > 0 {
		roleNames := make([]string, len(keyData.Roles))
		for i, role := range keyData.Roles {
			roleNames[i] = role.Name
		}
		response.Roles = &roleNames
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

		if len(keyRatelimits) > 0 {
			response.Ratelimits = &keyRatelimits
		}

		if len(identityRatelimits) > 0 {
			response.Identity.Ratelimits = &identityRatelimits
		}
	}

	// Set meta
	if keyData.Key.Meta.Valid {
		var meta map[string]any
		_ = json.Unmarshal([]byte(keyData.Key.Meta.String), &meta) // Ignore error, default to nil
		if meta != nil {
			response.Meta = &meta
		}
	}

	return response
}
