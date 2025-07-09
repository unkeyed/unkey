package handler

import (
	"context"
	"encoding/json"
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

	res := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.KeysVerifyKeyResponseData{
			Code:  openapi.NOTFOUND,
			Valid: false,
		},
	}

	key, err := h.Keys.Get(ctx, s, req.Key)
	if err != nil {
		return err
	}

	// Validate key belongs to authorized workspace
	if key.Key.WorkspaceID != auth.AuthorizedWorkspaceID {
		return s.JSON(http.StatusOK, res)
	}

	// Check if API is deleted
	if key.Key.ApiDeletedAtM.Valid {
		return s.JSON(http.StatusOK, res)
	}

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

	opts := []keys.VerifyOption{keys.WithIPWhitelist()}
	if req.Ratelimits != nil {
		opts = append(opts, keys.WithRateLimits(*req.Ratelimits))
	}

	if req.Credits != nil {
		opts = append(opts, keys.WithCredits(int32(req.Credits.Cost)))
	}

	if req.Permissions != nil {
		opts = append(opts, keys.WithPermissions(rbac.PermissionQuery{
			// Permissions: req.Permissions,
		}))
	}

	err = key.Verify(ctx, opts...)
	if err != nil {
		return err
	}

	res.Data = openapi.KeysVerifyKeyResponseData{
		Code:        key.ToOpenAPIStatus(),
		Valid:       key.Valid,
		Enabled:     ptr.P(key.Key.Enabled),
		Name:        ptr.P(key.Key.Name.String),
		Permissions: ptr.P(key.Permissions),
		Roles:       ptr.P(key.Roles),
		KeyId:       ptr.P(key.Key.ID),

		Credits:    nil,
		Expires:    nil,
		Identity:   nil,
		Meta:       nil,
		Ratelimits: nil,
	}

	if key.Key.RemainingRequests.Valid {
		res.Data.Credits = ptr.P(key.Key.RemainingRequests.Int32)
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
		// identityRatelimits := make([]openapi.RatelimitResponse, 0)

		res.Data.Identity = &openapi.Identity{
			ExternalId: key.Key.ExternalID.String,
			Id:         key.Key.IdentityID.String,
			Ratelimits: nil,
			Meta:       nil,
		}

		if len(key.Key.IdentityMeta) > 0 {
			err = json.Unmarshal([]byte(key.Key.IdentityMeta), &res.Data.Identity.Meta)
			if err != nil {
				return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()),
					fault.Internal("unable to unmarshal identity meta"),
					fault.Public("We encountered an error while trying to unmarshal the identity meta data."),
				)
			}
		}
	}

	return s.JSON(http.StatusOK, res)
}
