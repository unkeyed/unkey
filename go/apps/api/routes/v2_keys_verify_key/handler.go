package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"

	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"

	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysVerifyKeyRequestBody
type Response = openapi.V2KeysVerifyKeyResponseBody

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
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	// Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	key, err := h.Keys.Get(ctx, s, req.Key)
	if err != nil {
		return err
	}

	// Validate key belongs to authorized workspace
	if key.Key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			// nolint:exhaustruct
			Data: openapi.KeysVerifyKeyResponseData{
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
			Data: openapi.KeysVerifyKeyResponseData{
				Code:  openapi.NOTFOUND,
				Valid: false,
			},
		})
	}

	// FIXME: We are leaking a keys existance here... by telling the user that he doesn't have perms
	err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
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
		return err
	}

	opts := []keys.VerifyOption{keys.WithIPWhitelist(), keys.WithApiID(req.ApiId), keys.WithTags(ptr.SafeDeref(req.Tags))}

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
		query, queryErr := convertPermissionsToQuery(*req.Permissions)
		if queryErr != nil {
			return queryErr
		}

		opts = append(opts, keys.WithPermissions(query))
	}

	err = key.Verify(ctx, opts...)
	if err != nil {
		return err
	}

	res := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		// nolint:exhaustruct
		Data: openapi.KeysVerifyKeyResponseData{
			Code:        key.ToOpenAPIStatus(),
			Valid:       key.Status == keys.StatusValid,
			Enabled:     ptr.P(key.Key.Enabled),
			Name:        ptr.P(key.Key.Name.String),
			Permissions: ptr.P(key.Permissions),
			Roles:       ptr.P(key.Roles),
			KeyId:       ptr.P(key.Key.ID),
			Credits:     nil,
			Expires:     nil,
			Identity:    nil,
			Meta:        nil,
			Ratelimits:  nil,
		},
	}

	remaining := key.Key.RemainingRequests
	if remaining.Valid {
		res.Data.Credits = ptr.P(remaining.Int32)
	}

	if key.Key.Expires.Valid {
		res.Data.Expires = ptr.P(key.Key.Expires.Time.UnixMilli())
	}

	if key.Key.Meta.Valid {
		err = json.Unmarshal([]byte(key.Key.Meta.String), &res.Data.Meta)
		if err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Internal("unable to unmarshal key meta"),
				fault.Public("We encountered an error while trying to unmarshal the key meta data."),
			)
		}
	}

	if key.Key.IdentityID.Valid {
		res.Data.Identity = &openapi.Identity{
			ExternalId: key.Key.ExternalID.String,
			Ratelimits: nil,
			Meta:       nil,
		}

		for _, ratelimit := range key.GetRatelimitConfigs() {
			if ratelimit.IdentityID == "" {
				continue
			}

			res.Data.Identity.Ratelimits = append(res.Data.Identity.Ratelimits, openapi.RatelimitResponse{
				AutoApply: ratelimit.AutoApply == 1,
				Duration:  int64(ratelimit.Duration),
				Id:        ratelimit.ID,
				Limit:     int64(ratelimit.Limit),
				Name:      ratelimit.Name,
			})
		}

		if len(key.Key.IdentityMeta) > 0 {
			err = json.Unmarshal(key.Key.IdentityMeta, &res.Data.Identity.Meta)
			if err != nil {
				return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Internal("unable to unmarshal identity meta"),
					fault.Public("We encountered an error while trying to unmarshal the identity meta data."),
				)
			}
		}
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
				Id:        result.Name,
				Limit:     result.Limit,
				Name:      result.Name,
				Remaining: result.Response.Remaining,
				Reset:     result.Response.Reset.UnixMilli(),
			})
		}

		if len(ratelimitResponse) > 0 {
			res.Data.Ratelimits = ptr.P(ratelimitResponse)
		}
	}

	return s.JSON(http.StatusOK, res)
}

// convertPermissionsToQuery converts OpenAPI permissions to rbac.PermissionQuery
func convertPermissionsToQuery(permissions openapi.V2KeysVerifyKeyRequestBody_Permissions) (rbac.PermissionQuery, error) {
	// Try to unmarshal as string first (single permission)
	if perm, err := permissions.AsV2KeysVerifyKeyRequestBodyPermissions0(); err == nil {
		return rbac.S(perm), nil
	}

	// Try to unmarshal as object (multiple permissions with operator)
	if obj, err := permissions.AsV2KeysVerifyKeyRequestBodyPermissions1(); err == nil {
		if len(obj.Permissions) == 0 {
			return rbac.PermissionQuery{}, fmt.Errorf("permissions array cannot be empty")
		}

		queries := make([]rbac.PermissionQuery, 0, len(obj.Permissions))
		for _, perm := range obj.Permissions {
			queries = append(queries, rbac.S(perm))
		}

		switch obj.Type {
		case openapi.And:
			return rbac.And(queries...), nil
		case openapi.Or:
			return rbac.Or(queries...), nil
		default:
			return rbac.PermissionQuery{}, fmt.Errorf("unsupported operator: %s", obj.Type)
		}
	}

	return rbac.PermissionQuery{}, fmt.Errorf("invalid permissions format")
}
