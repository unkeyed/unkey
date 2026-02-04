package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitSetOverrideRequestBody
	Response = openapi.V2RatelimitSetOverrideResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit set override endpoint
type Handler struct {
	// Services as public fields
	Logger                  logging.Logger
	DB                      db.Database
	Keys                    keys.KeyService
	Auditlogs               auditlogs.AuditLogService
	RatelimitNamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.setOverride"
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

	overrideID, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
		var namespace db.FindRatelimitNamespaceRow
		namespace, err = db.Query.FindRatelimitNamespace(ctx, tx, db.FindRatelimitNamespaceParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Namespace:   req.Namespace,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return "", fault.New("namespace not found",
					fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
					fault.Public("This namespace does not exist."),
				)
			}
			return "", err
		}

		if namespace.DeletedAtM.Valid {
			return "", fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Public("This namespace does not exist."),
			)
		}

		wide.Set(ctx, wide.FieldRateLimitNamespace, namespace.Name)
		wide.Set(ctx, wide.FieldRateLimitIdentifier, wide.SanitizeIdentifier(req.Identifier))

		err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   namespace.ID,
				Action:       rbac.SetOverride,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   "*",
				Action:       rbac.SetOverride,
			}),
		)))
		if err != nil {
			return "", err
		}

		var override db.RatelimitOverride
		override, err = db.Query.FindRatelimitOverrideByIdentifier(ctx, tx, db.FindRatelimitOverrideByIdentifierParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
		})

		if err != nil && !db.IsNotFound(err) {
			return "", fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed"),
				fault.Public("The database is unavailable."),
			)
		}

		overrideID := uid.New(uid.RatelimitOverridePrefix)
		if !db.IsNotFound(err) {
			overrideID = override.ID
		}

		now := time.Now().UnixMilli()

		err = db.Query.InsertRatelimitOverride(ctx, tx, db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
			Limit:       int32(req.Limit),    // nolint:gosec
			Duration:    int32(req.Duration), //nolint:gosec
			CreatedAt:   now,
			UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
		})
		if err != nil {
			return "", fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed"),
				fault.Public("The database is unavailable."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitSetOverrideEvent,
				ActorID:     auth.Key.ID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Display:     fmt.Sprintf("Set ratelimit override for %s and %s", namespace.ID, req.Identifier),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.RatelimitOverrideResourceType,
						ID:          overrideID,
						Name:        req.Identifier,
						DisplayName: req.Identifier,
						Meta:        nil,
					},
				},
			},
		})
		if err != nil {
			return "", err
		}

		h.RatelimitNamespaceCache.Remove(ctx,
			cache.ScopedKey{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Key:         namespace.ID,
			},
			cache.ScopedKey{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Key:         namespace.Name,
			},
		)

		return overrideID, nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2RatelimitSetOverrideResponseData{
			OverrideId: overrideID,
		},
	})
}
