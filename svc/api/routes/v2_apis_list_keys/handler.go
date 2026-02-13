package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/oapi-codegen/nullable"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ApisListKeysRequestBody
	Response = openapi.V2ApisListKeysResponseBody
)

// Handler implements zen.Route interface for the v2 APIs list keys endpoint
type Handler struct {
	DB       db.Database
	Keys     keys.KeyService
	Vault    vault.Client
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

	// Resolve identity ID if external_id filter is provided
	var identityID sql.NullString
	if req.ExternalId != nil && *req.ExternalId != "" {
		identity, identityErr := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			ExternalID:  *req.ExternalId,
			Deleted:     false,
		})
		if identityErr != nil {
			if db.IsNotFound(identityErr) {
				// Identity doesn't exist, return empty result set
				return s.JSON(http.StatusOK, Response{
					Meta: openapi.Meta{
						RequestId: s.RequestID(),
					},
					Data: []openapi.KeyResponseData{},
					Pagination: &openapi.Pagination{
						Cursor:  nil,
						HasMore: false,
					},
				})
			}
			return fault.Wrap(identityErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve identity."),
			)
		}
		identityID = sql.NullString{String: identity.ID, Valid: true}
	}

	// Query keys by key_auth_id instead of api_id
	keyResults, err := db.Query.ListLiveKeysByKeySpaceID(
		ctx,
		h.DB.RO(),
		db.ListLiveKeysByKeySpaceIDParams{
			KeySpaceID: api.KeyAuthID.String,
			IDCursor:   cursor,
			IdentityID: identityID,
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
					logger.Error("failed to decrypt key",
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
		Name:        keyData.Key.Name.String,
		UpdatedAt:   keyData.Key.UpdatedAtM.Int64,
		Credits:     nil,
		Expires:     0,
		Identity:    nil,
		Permissions: nil,
		Roles:       nil,
		Plaintext:   plaintext,
		CreatedAt:   keyData.Key.CreatedAtM,
		Enabled:     keyData.Key.Enabled,
		KeyId:       keyData.Key.ID,
		Start:       keyData.Key.Start,
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

		identityMeta, err := db.UnmarshalNullableJSONTo[map[string]any](keyData.Identity.Meta)
		response.Identity.Meta = identityMeta
		if err != nil {
			logger.Error("failed to unmarshal identity meta", "error", err)
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
		response.Permissions = slugs
	}

	// Set roles
	if len(keyData.Roles) > 0 {
		roleNames := make([]string, len(keyData.Roles))
		for i, role := range keyData.Roles {
			roleNames[i] = role.Name
		}
		response.Roles = roleNames
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
		logger.Error("failed to unmarshal key meta",
			"keyId", keyData.Key.ID,
			"error", err,
		)
	}
	response.Meta = meta

	return response
}
