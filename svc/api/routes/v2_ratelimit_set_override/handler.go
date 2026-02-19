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
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitSetOverrideRequestBody
	Response = openapi.V2RatelimitSetOverrideResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit set override endpoint
type Handler struct {
	DB             db.Database
	Keys           keys.KeyService
	Auditlogs      auditlogs.AuditLogService
	NamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.setOverride"
}

type setOverrideResult struct {
	overrideID    string
	namespaceName string
	namespaceID   string
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

	// Keep the namespace lookup inside the transaction for transactional read consistency.
	result, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (setOverrideResult, error) {
		var zero setOverrideResult
		nsRow, txErr := db.Query.FindRatelimitNamespace(ctx, tx, db.FindRatelimitNamespaceParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Namespace:   req.Namespace,
		})
		if txErr != nil {
			if db.IsNotFound(txErr) {
				return zero, fault.New("namespace not found",
					fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
					fault.Public("This namespace does not exist."),
				)
			}
			return zero, txErr
		}

		if nsRow.DeletedAtM.Valid {
			return zero, fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Public("This namespace does not exist."),
			)
		}

		txErr = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   nsRow.ID,
				Action:       rbac.SetOverride,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   "*",
				Action:       rbac.SetOverride,
			}),
		)))
		if txErr != nil {
			return zero, txErr
		}

		override, txErr := db.Query.FindRatelimitOverrideByIdentifier(ctx, tx, db.FindRatelimitOverrideByIdentifierParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: nsRow.ID,
			Identifier:  req.Identifier,
		})

		if txErr != nil && !db.IsNotFound(txErr) {
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed"),
				fault.Public("The database is unavailable."),
			)
		}

		ovrID := uid.New(uid.RatelimitOverridePrefix)
		if txErr == nil {
			ovrID = override.ID
		}

		now := time.Now().UnixMilli()

		txErr = db.Query.InsertRatelimitOverride(ctx, tx, db.InsertRatelimitOverrideParams{
			ID:          ovrID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: nsRow.ID,
			Identifier:  req.Identifier,
			Limit:       int32(req.Limit),    // nolint:gosec
			Duration:    int32(req.Duration), //nolint:gosec
			CreatedAt:   now,
			UpdatedAt:   sql.NullInt64{Int64: now, Valid: true},
		})
		if txErr != nil {
			return zero, fault.Wrap(txErr,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed"),
				fault.Public("The database is unavailable."),
			)
		}

		txErr = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitSetOverrideEvent,
				ActorID:     auth.Key.ID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Display:     fmt.Sprintf("Set ratelimit override for %s and %s", nsRow.ID, req.Identifier),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.RatelimitOverrideResourceType,
						ID:          ovrID,
						Name:        req.Identifier,
						DisplayName: req.Identifier,
						Meta:        nil,
					},
				},
			},
		})
		if txErr != nil {
			return zero, txErr
		}

		return setOverrideResult{
			overrideID:    ovrID,
			namespaceName: nsRow.Name,
			namespaceID:   nsRow.ID,
		}, nil
	})
	if err != nil {
		return err
	}

	// Invalidate cache for this namespace after the transaction commits
	h.NamespaceCache.Remove(ctx,
		cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: result.namespaceID},
		cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: result.namespaceName},
	)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2RatelimitSetOverrideResponseData{
			OverrideId: result.overrideID,
		},
	})
}
