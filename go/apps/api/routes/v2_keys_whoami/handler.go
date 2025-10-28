package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"sort"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type (
	Request  = openapi.V2KeysWhoamiRequestBody
	Response = openapi.V2KeysWhoamiResponseBody
)

// Handler implements zen.Route interface for the v2 keys.whoami endpoint
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
	return "/v2/keys.whoami"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
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

	key, err := db.Query.FindLiveKeyByHash(ctx, h.DB.RO(), hash.Sha256(req.Key))
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

	keyData := db.ToKeyData(key, h.Logger)

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
		return fault.Wrap(err,
			fault.Code(codes.Data.Key.NotFound.URN()),
			fault.Internal("user doesn't have permissions and we don't want to leak the existence of the key"),
			fault.Public("The specified key was not found."),
		)
	}

	response := openapi.KeyResponseData{
		CreatedAt: keyData.Key.CreatedAtM,
		Enabled:   keyData.Key.Enabled,
		KeyId:     keyData.Key.ID,
		Start:     keyData.Key.Start,
		Plaintext: nil,
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
	if keyData.Key.RemainingRequests.Valid {
		response.Credits = &openapi.KeyCreditsData{
			Remaining: nullable.NewNullableWithValue(int64(keyData.Key.RemainingRequests.Int32)),
		}

		if keyData.Key.RefillAmount.Valid {
			var refillDay *int
			interval := openapi.KeyCreditsRefillIntervalDaily
			if keyData.Key.RefillDay.Valid {
				interval = openapi.KeyCreditsRefillIntervalMonthly
				refillDay = ptr.P(int(keyData.Key.RefillDay.Int16))
			}

			response.Credits.Refill = &openapi.KeyCreditsRefill{
				Amount:    int64(keyData.Key.RefillAmount.Int32),
				Interval:  interval,
				RefillDay: refillDay,
			}
		}
	}

	if keyData.Identity != nil {
		response.Identity = &openapi.Identity{
			Id:         keyData.Identity.ID,
			ExternalId: keyData.Identity.ExternalID,
		}

		if len(keyData.Identity.Meta) > 0 {
			meta := db.UnmarshalNullableJSONTo[map[string]any](keyData.Identity.Meta, h.Logger)
			response.Identity.Meta = &meta
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
	if len(permissionSlugs) > 0 {
		slugs := make([]string, 0, len(permissionSlugs))
		for slug := range permissionSlugs {
			slugs = append(slugs, slug)
		}
		sort.Strings(slugs)
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
		if err := json.Unmarshal([]byte(keyData.Key.Meta.String), &meta); err != nil {
			h.Logger.Error("failed to unmarshal key meta", "error", err)
		} else {
			response.Meta = &meta
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: response,
	})
}
