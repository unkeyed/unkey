package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hash"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysVerifyKeyRequestBody
type Response = openapi.V2KeysVerifyKeyResponseBody

// Handler implements zen.Route interface for the v2 keys.verify endpoint
type Handler struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
	ClickHouse  clickhouse.ClickHouse
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
	auth, err := h.Keys.VerifyRootKey(ctx, s)
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

	key, err := h.Keys.Get(ctx, hash.Sha256(req.Key))
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

	// Permission check
	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(
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
		),
	)
	if err != nil {
		// The Root Key is in the same workspace as the API so we can show that it does not have permission to verify keys.
		// (I think so?)
		return err
	}

	// result, err := key.
	// 	WithCredits(ctx, 1).
	// 	WithIPWhitelist(ctx, s.Location()).
	// 	WithPermissions(ctx, rbac.PermissionQuery{}).
	// 	WithRateLimits(ctx, req.Ratelimits).
	// 	Result()

	h.ClickHouse.BufferKeyVerification(schema.KeyVerificationRequestV1{
		RequestID:   s.RequestID(),
		Time:        time.Now().UnixMilli(),
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Region:      "",
		Outcome:     "",
		KeySpaceID:  "",
		KeyID:       "",
		IdentityID:  "",
		Tags:        []string{},
	})

	return s.JSON(http.StatusOK, res)
}
