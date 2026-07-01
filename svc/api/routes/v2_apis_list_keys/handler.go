package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/oapi-codegen/nullable"
	vaultv1 "github.com/unkeyed/unkey/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/gen/rpc/vault"
	"github.com/unkeyed/unkey/internal/services/caches"
	authprincipal "github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ApisListKeysRequestBody
	Response = openapi.V2ApisListKeysResponseBody
)

// Handler implements zen.Route interface for the v2 APIs list keys endpoint
type Handler struct {
	DB       db.Database
	Vault    vault.VaultServiceClient
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
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}
	api, hit, err := h.ApiCache.SWR(ctx, cache.ScopedKey{
		WorkspaceID: principal.WorkspaceID,
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
	if api.WorkspaceID != principal.WorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"), fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	if !api.KeyAuthID.Valid {
		return fault.New("api missing keyspace",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("api has no key auth id"),
			fault.Public("Failed to retrieve API information."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.And(
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Keyspace(api.KeyAuthID.String).Key("*"),
				permissions.ReadKey{},
			),
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Keyspace(api.KeyAuthID.String),
				permissions.ReadKeyspace{},
			),
		),
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
	))
	if err != nil {
		// Mask a read-authorization failure as 404 so that callers who lack read
		// access cannot distinguish an existing API from a non-existent one and
		// enumerate API IDs in the workspace. The authorization runs after the
		// lookup because the URN check needs the keyspace ID.
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Api.NotFound.URN(),
			"The requested API does not exist or has been deleted.",
		)
	}

	if ptr.SafeDeref(req.Decrypt, false) {
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
				Action:       rbac.DecryptKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   api.ID,
				Action:       rbac.DecryptKey,
			}),
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Keyspace(api.KeyAuthID.String).Key("*"),
				permissions.DecryptKey{},
			),
		))
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

	// Portal sessions are scoped to a single external identity. Override any
	// user-supplied externalId filter so that the session can only list its own keys.
	// Fail closed: if the externalId is empty, reject the request rather than
	// returning unscoped keys.
	//
	// Identity scoping is intentionally separate from the RBAC permission system.
	// Permissions gate what operations a principal can perform; identity scoping
	// gates what data is visible. Portal sessions carry a fixed externalId that
	// restricts visibility regardless of what the request body says.
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

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")

	// Resolve identity ID if external_id filter is provided
	var identityID sql.NullString
	if req.ExternalId != nil && *req.ExternalId != "" {
		identity, identityErr := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: principal.WorkspaceID,
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

	plaintextMap := h.decryptKeys(ctx, req, keyResults, principal.WorkspaceID)

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

func (h *Handler) decryptKeys(ctx context.Context, req Request, keys []db.ListLiveKeysByKeySpaceIDRow, workspaceID string) map[string]string {
	if req.Decrypt == nil || !*req.Decrypt {
		return nil
	}

	bulkItems := make(map[string]string, len(keys))
	for _, key := range keys {
		if key.EncryptedKey.Valid && key.EncryptionKeyID.Valid {
			bulkItems[key.ID] = key.EncryptedKey.String
		}
	}
	if len(bulkItems) == 0 {
		return nil
	}

	bulkRes, err := h.Vault.DecryptBulk(ctx, &vaultv1.DecryptBulkRequest{
		Keyring: workspaceID,
		Items:   bulkItems,
	})
	if err != nil {
		logger.Error("failed to bulk decrypt keys", "error", err)
		return nil
	}

	return bulkRes.GetItems()
}

// buildKeyResponseData transforms internal key data into API response format.
func (h *Handler) buildKeyResponseData(keyData *db.KeyData, plaintext string) openapi.KeyResponseData {
	response := openapi.KeyResponseData{
		Meta:        nil,
		Ratelimits:  nil,
		Name:        keyData.Key.Name.String,
		UpdatedAt:   keyData.Key.UpdatedAtM.Int64,
		LastUsedAt:  int64(keyData.Key.LastUsedAt),
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
			Remaining: nullable.NewNullableWithValue(keyData.Key.RemainingRequests.Int64),
		}

		if keyData.Key.RefillAmount.Valid {
			var refillDay int
			interval := openapi.KeyCreditsRefillIntervalDaily
			if keyData.Key.RefillDay.Valid {
				interval = openapi.KeyCreditsRefillIntervalMonthly
				refillDay = int(keyData.Key.RefillDay.Int16)
			}

			response.Credits.Refill = &openapi.KeyCreditsRefill{
				Amount:    keyData.Key.RefillAmount.Int64,
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
				Duration:  int64(rl.Duration),
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
