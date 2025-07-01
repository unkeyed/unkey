package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	vaultv1 "github.com/unkeyed/unkey/go/gen/proto/vault/v1"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2ApisListKeysRequestBody
type Response = openapi.V2ApisListKeysResponseBody

// Handler implements zen.Route interface for the v2 APIs list keys endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Vault       *vault.Service
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
// The current implementation queries the database directly without caching, which may impact performance.
// Consider implementing cache with optional bypass via revalidateKeysCache parameter.
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	var req Request
	err = s.BindBody(&req)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
		)
	}

	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(
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
		),
	)
	if err != nil {
		return err
	}

	// 4. Get API from database
	api, err := db.Query.FindApiByID(ctx, h.DB.RO(), req.ApiId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("api not found",
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("api not found"), fault.Public("The requested API does not exist or has been deleted."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve API information."),
		)
	}
	// Check if API belongs to the authorized workspace
	if api.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"), fault.Public("The requested API does not exist or has been deleted."),
		)
	}
	// Check if API is deleted
	if api.DeletedAtM.Valid {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api not found"), fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	// Check if API is set up to handle keys
	if !api.KeyAuthID.Valid || api.KeyAuthID.String == "" {
		return fault.New("api not set up for keys",
			fault.Code(codes.App.Precondition.PreconditionFailed.URN()),
			fault.Internal("api not set up for keys"), fault.Public("The requested API is not set up to handle keys."),
		)
	}

	// 5. Query the keys
	var identityId string
	if req.ExternalId != nil && *req.ExternalId != "" {
		identity, findErr := db.Query.FindIdentityByExternalID(ctx, h.DB.RO(), db.FindIdentityByExternalIDParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			ExternalID:  *req.ExternalId,
			Deleted:     false,
		})
		if findErr != nil {
			if !db.IsNotFound(findErr) {
				return fault.Wrap(findErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to retrieve identity information."),
				)
			}
			// If identity not found, return empty result
			return s.JSON(http.StatusOK, Response{
				Meta: openapi.Meta{
					RequestId: s.RequestID(),
				},
				Data: []openapi.KeyResponse{},
				Pagination: &openapi.Pagination{
					Cursor:  nil,
					HasMore: false,
				},
			})
		}
		identityId = identity.ID
	}

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")
	// List keys
	keys, err := db.Query.ListKeysByKeyAuthID(
		ctx,
		h.DB.RO(),
		db.ListKeysByKeyAuthIDParams{
			KeyAuthID:  api.KeyAuthID.String,
			Limit:      int32(limit + 1), // nolint:gosec
			IDCursor:   cursor,
			IdentityID: sql.NullString{Valid: identityId != "", String: identityId},
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve keys."),
		)
	}

	// Query ratelimits for all returned keys
	ratelimitsMap := make(map[string][]db.ListRatelimitsByKeyIDsRow)
	if len(keys) > 0 {
		// Extract key IDs and convert to sql.NullString slice
		keyIDs := make([]sql.NullString, len(keys))
		for i, key := range keys {
			keyIDs[i] = sql.NullString{Valid: true, String: key.Key.ID}
		}

		// Query ratelimits for these keys
		ratelimits, listErr := db.Query.ListRatelimitsByKeyIDs(ctx, h.DB.RO(), keyIDs)
		if listErr != nil {
			return fault.Wrap(listErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve ratelimits."),
			)
		}

		// Group ratelimits by key_id
		for _, rl := range ratelimits {
			if rl.KeyID.Valid {
				ratelimitsMap[rl.KeyID.String] = append(ratelimitsMap[rl.KeyID.String], rl)
			}
		}
	}

	// If user requested decryption, check permissions and decrypt
	plaintextMap := map[string]string{} // nolint:staticcheck
	if req.Decrypt != nil && *req.Decrypt {
		err = h.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
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
			),
		)
		if err != nil {
			return err
		}

		// If we have permission, proceed with decryption
		for _, key := range keys {
			if key.EncryptedKey.Valid && key.EncryptionKeyID.Valid {
				decrypted, decryptErr := h.Vault.Decrypt(ctx, &vaultv1.DecryptRequest{
					Keyring:   key.Key.WorkspaceID,
					Encrypted: key.EncryptedKey.String,
				})
				if decryptErr != nil {
					h.Logger.Error("failed to decrypt key",
						"keyId", key.Key.ID,
						"error", decryptErr,
					)
					continue
				}
				plaintextMap[key.Key.ID] = decrypted.GetPlaintext()
			}
		}
	}

	// Filter out the cursor key if cursor was provided (to avoid duplicates)
	filteredKeys := keys
	if cursor != "" {
		var filtered []db.ListKeysByKeyAuthIDRow
		for _, key := range keys {
			if key.Key.ID != cursor {
				filtered = append(filtered, key)
			}
		}
		filteredKeys = filtered
	}

	// Determine the actual number of keys to return (respect the limit)
	numKeysToReturn := len(filteredKeys)
	hasMore := false
	if len(filteredKeys) > limit {
		numKeysToReturn = limit
		hasMore = true
	}

	// Transform keys into the response format
	responseData := make([]openapi.KeyResponse, numKeysToReturn)
	for i := 0; i < numKeysToReturn; i++ {
		key := filteredKeys[i]
		k := openapi.KeyResponse{
			KeyId:       key.Key.ID,
			Start:       key.Key.Start,
			CreatedAt:   key.Key.CreatedAtM,
			Enabled:     key.Key.Enabled,
			Credits:     nil,
			Expires:     nil,
			Identity:    nil,
			Meta:        nil,
			Name:        nil,
			Permissions: nil,
			Plaintext:   nil,
			Ratelimits:  nil,
			Roles:       nil,
			UpdatedAt:   nil,
		}

		if key.Key.Name.Valid {
			k.Name = ptr.P(key.Key.Name.String)
		}

		if key.Key.Meta.Valid {
			err = json.Unmarshal([]byte(key.Key.Meta.String), &k.Meta)
			if err != nil {
				h.Logger.Error("unable to unmarshal key metadata",
					"keyId", key.Key.ID,
					"error", err,
				)
			}
		}

		if key.Key.UpdatedAtM.Valid {
			k.UpdatedAt = ptr.P(key.Key.UpdatedAtM.Int64)
		}

		if key.Key.Expires.Valid {
			k.Expires = ptr.P(key.Key.Expires.Time.UnixMilli())
		}

		if key.Key.RemainingRequests.Valid {
			k.Credits = &openapi.KeyCredits{
				Remaining: int64(key.Key.RemainingRequests.Int32),
				Refill:    nil,
			}
			if key.Key.RefillAmount.Valid {
				k.Credits.Refill = &openapi.KeyCreditsRefill{
					Amount:       int64(key.Key.RefillAmount.Int32),
					Interval:     "",
					RefillDay:    ptr.P(int(key.Key.RefillDay.Int16)),
					LastRefillAt: nil,
				}
				if key.Key.LastRefillAt.Valid {
					k.Credits.Refill.LastRefillAt = ptr.P(key.Key.LastRefillAt.Time.UnixMilli())
				}
			}
		}

		// Add plaintext if available
		if plaintext, ok := plaintextMap[key.Key.ID]; ok {
			k.Plaintext = ptr.P(plaintext)
		}

		// Add identity information if available
		if key.IdentityID.Valid {
			k.Identity = &openapi.Identity{
				ExternalId: key.ExternalID.String,
				Id:         key.IdentityID.String,
				Meta:       nil,
				Ratelimits: nil,
			}
			if len(key.IdentityMeta) > 0 {
				err = json.Unmarshal(key.IdentityMeta, &k.Identity.Meta)
				if err != nil {
					return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
						fault.Internal("unable to unmarshal identity meta"), fault.Public("We encountered an error while trying to unmarshal the identity meta data."))
				}
			}
		}

		// Get permissions for the key
		permissionSlugs, err := db.Query.ListPermissionsByKeyID(ctx, h.DB.RO(), db.ListPermissionsByKeyIDParams{
			KeyID: k.KeyId,
		})
		if err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("unable to find permissions for key"), fault.Public("Could not load permissions for key."))
		}
		k.Permissions = ptr.P(permissionSlugs)

		// Get roles for the key
		roles, err := db.Query.ListRolesByKeyID(ctx, h.DB.RO(), k.KeyId)
		if err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("unable to find roles for key"), fault.Public("Could not load roles for key."))
		}

		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}

		k.Roles = ptr.P(roleNames)

		// Add ratelimits for the key
		if keyRatelimits, exists := ratelimitsMap[k.KeyId]; exists {
			ratelimitsResponse := make([]openapi.RatelimitResponse, len(keyRatelimits))
			for j, rl := range keyRatelimits {
				ratelimitsResponse[j] = openapi.RatelimitResponse{
					Id:        rl.ID,
					Name:      rl.Name,
					Limit:     int64(rl.Limit),
					Duration:  rl.Duration,
					AutoApply: rl.AutoApply,
				}
			}
			k.Ratelimits = ptr.P(ratelimitsResponse)
		}

		responseData[i] = k
	}

	// Determine the cursor for the next page
	var nextCursor *string
	if hasMore && numKeysToReturn > 0 {
		cursor := responseData[numKeysToReturn-1].KeyId
		nextCursor = &cursor
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
