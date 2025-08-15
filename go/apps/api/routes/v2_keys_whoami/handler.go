package handler

import (
	"context"
	"encoding/json"
	"net/http"

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

	// Validate key belongs to authorized workspace
	if key.WorkspaceID != auth.AuthorizedWorkspaceID {
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
			ResourceID:   key.Api.ID,
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

	k := openapi.KeyResponseData{
		CreatedAt:   key.CreatedAtM,
		Enabled:     key.Enabled,
		KeyId:       key.ID,
		Start:       key.Start,
		Plaintext:   nil,
		Name:        nil,
		Meta:        nil,
		Identity:    nil,
		Credits:     nil,
		Expires:     nil,
		Permissions: nil,
		Ratelimits:  nil,
		Roles:       nil,
		UpdatedAt:   nil,
	}

	if key.Name.Valid {
		k.Name = ptr.P(key.Name.String)
	}

	if key.UpdatedAtM.Valid {
		k.UpdatedAt = ptr.P(key.UpdatedAtM.Int64)
	}

	if key.Expires.Valid {
		k.Expires = ptr.P(key.Expires.Time.UnixMilli())
	}

	h.setCredits(&k, key)
	h.setIdentity(&k, key)
	h.setPermissions(&k, key)
	h.setRoles(&k, key)
	h.setRatelimits(&k, key)
	h.setMeta(&k, key)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: k,
	})
}

func (h *Handler) setCredits(response *openapi.KeyResponseData, key db.FindLiveKeyByHashRow) {
	if !key.RemainingRequests.Valid {
		return
	}

	response.Credits = &openapi.KeyCreditsData{
		Remaining: nullable.NewNullableWithValue(int64(key.RemainingRequests.Int32)),
		Refill:    nil,
	}

	if key.RefillAmount.Valid {
		var refillDay *int
		interval := openapi.Daily
		if key.RefillDay.Valid {
			interval = openapi.Monthly
			refillDay = ptr.P(int(key.RefillDay.Int16))
		}

		response.Credits.Refill = &openapi.KeyCreditsRefill{
			Amount:    int64(key.RefillAmount.Int32),
			Interval:  interval,
			RefillDay: refillDay,
		}
	}
}

func (h *Handler) setIdentity(response *openapi.KeyResponseData, key db.FindLiveKeyByHashRow) {
	if !key.IdentityID.Valid {
		return
	}

	response.Identity = &openapi.Identity{
		Id:         key.IdentityTableID.String,
		ExternalId: key.IdentityExternalID.String,
		Meta:       nil,
		Ratelimits: nil,
	}

	if len(key.IdentityMeta) > 0 {
		if err := json.Unmarshal(key.IdentityMeta, &response.Identity.Meta); err != nil {
			h.Logger.Error("failed to unmarshal identity meta", "error", err)
		}
	}
}

func (h *Handler) setPermissions(response *openapi.KeyResponseData, key db.FindLiveKeyByHashRow) {
	permissionSlugs := make(map[string]struct{})

	// Direct permissions
	if key.Permissions != nil {
		var directPermissions []db.PermissionInfo
		if err := json.Unmarshal(key.Permissions.([]byte), &directPermissions); err != nil {
			h.Logger.Error("failed to unmarshal permissions", "error", err)
		} else {
			for _, p := range directPermissions {
				permissionSlugs[p.Slug] = struct{}{}
			}
		}
	}

	// Role permissions
	if key.RolePermissions != nil {
		var rolePermissions []db.PermissionInfo
		if err := json.Unmarshal(key.RolePermissions.([]byte), &rolePermissions); err != nil {
			h.Logger.Error("failed to unmarshal role permissions", "error", err)
		} else {
			for _, p := range rolePermissions {
				permissionSlugs[p.Slug] = struct{}{}
			}
		}
	}

	if len(permissionSlugs) > 0 {
		slugs := make([]string, 0, len(permissionSlugs))
		for slug := range permissionSlugs {
			slugs = append(slugs, slug)
		}
		response.Permissions = ptr.P(slugs)
	}
}

func (h *Handler) setRoles(response *openapi.KeyResponseData, key db.FindLiveKeyByHashRow) {
	if key.Roles == nil {
		return
	}

	var roles []db.RoleInfo
	if err := json.Unmarshal(key.Roles.([]byte), &roles); err != nil {
		h.Logger.Error("failed to unmarshal roles", "error", err)
		return
	}

	if len(roles) > 0 {
		roleNames := make([]string, len(roles))
		for i, role := range roles {
			roleNames[i] = role.Name
		}
		response.Roles = ptr.P(roleNames)
	}
}

func (h *Handler) setRatelimits(response *openapi.KeyResponseData, key db.FindLiveKeyByHashRow) {
	if key.Ratelimits == nil {
		return
	}

	var ratelimits []db.RatelimitInfo
	if err := json.Unmarshal(key.Ratelimits.([]byte), &ratelimits); err != nil {
		h.Logger.Error("failed to unmarshal ratelimits", "error", err)
		return
	}

	var keyRatelimits []openapi.RatelimitResponse
	var identityRatelimits []openapi.RatelimitResponse

	for _, rl := range ratelimits {
		ratelimitResp := openapi.RatelimitResponse{
			Id:        rl.ID,
			Duration:  rl.Duration,
			Limit:     int64(rl.Limit),
			Name:      rl.Name,
			AutoApply: rl.AutoApply,
		}

		// Add to key ratelimits if it belongs to this key
		if rl.KeyID.Valid && rl.KeyID.String == key.ID {
			keyRatelimits = append(keyRatelimits, ratelimitResp)
		}

		// Also add to identity ratelimits if it has an identity_id that matches
		if rl.IdentityID.Valid && key.IdentityID.Valid && rl.IdentityID.String == key.IdentityID.String {
			identityRatelimits = append(identityRatelimits, ratelimitResp)
		}
	}

	if len(keyRatelimits) > 0 {
		response.Ratelimits = ptr.P(keyRatelimits)
	}

	if len(identityRatelimits) > 0 && response.Identity != nil {
		response.Identity.Ratelimits = ptr.P(identityRatelimits)
	}
}

func (h *Handler) setMeta(response *openapi.KeyResponseData, key db.FindLiveKeyByHashRow) {
	if key.Meta.Valid {
		if err := json.Unmarshal([]byte(key.Meta.String), &response.Meta); err != nil {
			h.Logger.Error("failed to unmarshal key meta", "error", err)
		}
	}
}
