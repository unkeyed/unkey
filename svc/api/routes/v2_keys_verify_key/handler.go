package handler

import (
	"context"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/svc/api/openapi"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/hash"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/zen"
)

type (
	Request  = openapi.V2KeysVerifyKeyRequestBody
	Response = openapi.V2KeysVerifyKeyResponseBody
)

const DefaultCost = 1

// Handler implements zen.Route interface for the v2 keys.verify endpoint
type Handler struct {
	Logger     logging.Logger
	DB         db.Database
	Keys       keys.KeyService
	Auditlogs  auditlogs.AuditLogService
	ClickHouse clickhouse.ClickHouse
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/keys.verifyKey"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.verifyKey")

	// Authentication
	auth, rootEmit, err := h.Keys.GetRootKey(ctx, s)
	defer rootEmit()
	if err != nil {
		return err
	}

	// Check if the root key has ANY verify permissions at all.
	// If not, return a proper permissions error immediately without looking up the key.
	// This prevents returning NOT_FOUND for every request when the root key simply lacks verify permissions entirely.
	if !auth.HasAnyPermission(rbac.Api, rbac.VerifyKey) {
		return auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "*",
				Action:       rbac.VerifyKey,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Api,
				ResourceID:   "<API_ID>",
				Action:       rbac.VerifyKey,
			}),
		)))
	}

	// Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, emit, err := h.Keys.Get(ctx, s, hash.Sha256(req.Key))
	if err != nil {
		return err
	}

	if key.Status == keys.StatusNotFound && req.MigrationId != nil {
		key, emit, err = h.Keys.GetMigrated(ctx, s, req.Key, ptr.SafeDeref(req.MigrationId))
		if err != nil {
			return err
		}
	}

	if key.Status != keys.StatusNotFound {
		wide.Set(ctx, wide.FieldKeyID, key.Key.ID)
		wide.Set(ctx, wide.FieldAPIID, key.Key.ApiID)
	}

	// Validate key belongs to authorized workspace
	if key.Key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			// nolint:exhaustruct
			Data: openapi.V2KeysVerifyKeyResponseData{
				Code:  openapi.NOTFOUND,
				Valid: false,
			},
		})
	}

	// Check if API is deleted
	if key.Key.ApiDeletedAtM.Valid {
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			// nolint:exhaustruct
			Data: openapi.V2KeysVerifyKeyResponseData{
				Code:  openapi.NOTFOUND,
				Valid: false,
			},
		})
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.VerifyKey,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   key.Key.ApiID,
			Action:       rbac.VerifyKey,
		}),
	)))
	if err != nil {
		// We are just respond with a 200 OK with a not found since the user doesn't have permission to verify the key
		// this would otherwise leak the keys existence otherwise
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			// nolint:exhaustruct
			Data: openapi.V2KeysVerifyKeyResponseData{
				Code:  openapi.NOTFOUND,
				Valid: false,
			},
		})
	}

	opts := []keys.VerifyOption{
		keys.WithTags(ptr.SafeDeref(req.Tags)),
		keys.WithIPWhitelist(),
	}

	// If a custom cost was specified, use it, otherwise use a DefaultCost of 1
	if req.Credits != nil {
		opts = append(opts, keys.WithCredits(req.Credits.Cost))
	} else if key.Key.RemainingRequests.Valid {
		opts = append(opts, keys.WithCredits(DefaultCost))
	}

	if req.Ratelimits != nil {
		opts = append(opts, keys.WithRateLimits(*req.Ratelimits))
	} else {
		// check auto applied ratelimits
		opts = append(opts, keys.WithRateLimits(nil))
	}

	if req.Permissions != nil {
		// Parse the permissions query string using the RBAC parser
		query, parseErr := rbac.ParseQuery(*req.Permissions)
		if parseErr != nil {
			return fault.Wrap(parseErr,
				fault.Code(codes.User.BadRequest.PermissionsQuerySyntaxError.URN()),
				fault.Internal(fmt.Sprintf("failed to parse permissions query: %s", *req.Permissions)),
			)
		}

		opts = append(opts, keys.WithPermissions(query))
	}

	err = key.Verify(ctx, opts...)
	if err != nil {
		return err
	}

	keyData := openapi.V2KeysVerifyKeyResponseData{
		Code:        key.ToOpenAPIStatus(),
		Valid:       key.Status == keys.StatusValid,
		Enabled:     ptr.P(key.Key.Enabled),
		Name:        key.Key.Name.String,
		KeyId:       key.Key.ID,
		Permissions: key.Permissions,
		Roles:       key.Roles,
		Credits:     nil,
		Expires:     0,
		Identity:    nil,
		Meta:        nil,
		Ratelimits:  nil,
	}

	if key.Key.Expires.Valid {
		keyData.Expires = key.Key.Expires.Time.UnixMilli()
	}

	remaining := key.Key.RemainingRequests
	if remaining.Valid {
		keyData.Credits = ptr.P(remaining.Int32)
	}

	if key.Key.Meta.Valid {
		meta, err := db.UnmarshalNullableJSONTo[map[string]any](key.Key.Meta.String)
		if err != nil {
			h.Logger.Error("failed to unmarshal key meta", "keyId", key.Key.ID, "error", err)
		}
		keyData.Meta = meta
	}

	if key.Key.IdentityID.Valid {
		keyData.Identity = &openapi.Identity{
			Id:         key.Key.IdentityID.String,
			ExternalId: key.Key.ExternalID.String,
			Ratelimits: nil,
			Meta:       nil,
		}

		identityRatelimits := make([]openapi.RatelimitResponse, 0)
		for _, ratelimit := range key.GetRatelimitConfigs() {
			if ratelimit.IdentityID == "" {
				continue
			}

			identityRatelimits = append(identityRatelimits, openapi.RatelimitResponse{
				AutoApply: ratelimit.AutoApply == 1,
				Duration:  int64(ratelimit.Duration),
				Id:        ratelimit.ID,
				Limit:     int64(ratelimit.Limit),
				Name:      ratelimit.Name,
			})
		}

		if len(identityRatelimits) > 0 {
			keyData.Identity.Ratelimits = identityRatelimits
		}

		meta, err := db.UnmarshalNullableJSONTo[map[string]any](key.Key.IdentityMeta)
		if err != nil {
			h.Logger.Error(
				"failed to unmarshal identity meta",
				"identityId", key.Key.IdentityID.String,
				"error", err,
			)
		}
		keyData.Identity.Meta = meta
	}

	if len(key.RatelimitResults) > 0 {
		ratelimitResponse := make([]openapi.VerifyKeyRatelimitData, 0)
		for _, result := range key.RatelimitResults {
			if result.Response == nil {
				continue
			}

			ratelimitResponse = append(ratelimitResponse, openapi.VerifyKeyRatelimitData{
				AutoApply: result.AutoApply,
				Duration:  result.Duration.Milliseconds(),
				Exceeded:  !result.Response.Success,
				Id:        result.ID,
				Limit:     result.Limit,
				Name:      result.Name,
				Remaining: result.Response.Remaining,
				Reset:     result.Response.Reset.UnixMilli(),
			})
		}

		if len(ratelimitResponse) > 0 {
			keyData.Ratelimits = ratelimitResponse
		}
	}

	emit()

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: keyData,
	})
}
