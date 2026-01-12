package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitDeleteOverrideRequestBody
	Response = openapi.V2RatelimitDeleteOverrideResponseBody
)

type Handler struct {
	Logger                  logging.Logger
	DB                      db.Database
	Keys                    keys.KeyService
	Auditlogs               auditlogs.AuditLogService
	RatelimitNamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/ratelimit.deleteOverride"
}

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

	namespace, hit, err := h.RatelimitNamespaceCache.SWR(
		ctx,
		cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: req.Namespace},
		func(ctx context.Context) (db.FindRatelimitNamespace, error) {
			result := db.FindRatelimitNamespace{} // nolint:exhaustruct
			var response db.FindRatelimitNamespaceRow
			response, err = db.WithRetryContext(ctx, func() (db.FindRatelimitNamespaceRow, error) {
				return db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
					WorkspaceID: auth.AuthorizedWorkspaceID,
					Namespace:   req.Namespace,
				})
			})
			if err != nil {
				return result, err
			}

			result = db.FindRatelimitNamespace{
				ID:                response.ID,
				WorkspaceID:       response.WorkspaceID,
				Name:              response.Name,
				CreatedAtM:        response.CreatedAtM,
				UpdatedAtM:        response.UpdatedAtM,
				DeletedAtM:        response.DeletedAtM,
				DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
				WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
			}

			overrides := make([]db.FindRatelimitNamespaceLimitOverride, 0)
			if overrideBytes, ok := response.Overrides.([]byte); ok && overrideBytes != nil {
				err = json.Unmarshal(overrideBytes, &overrides)
				if err != nil {
					return result, err
				}
			}

			for _, override := range overrides {
				result.DirectOverrides[override.Identifier] = override
				if strings.Contains(override.Identifier, "*") {
					result.WildcardOverrides = append(result.WildcardOverrides, override)
				}
			}

			return result, nil
		},
		caches.DefaultFindFirstOp,
	)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("namespace not found",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Internal("namespace not found"),
				fault.Public("This namespace does not exist."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching the namespace."),
		)
	}

	if namespace.DeletedAtM.Valid {
		return fault.New("namespace deleted",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace deleted"),
			fault.Public("This namespace does not exist."),
		)
	}

	if hit == cache.Null {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace not found"),
			fault.Public("This namespace does not exist."),
		)
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Ratelimit,
			ResourceID:   namespace.ID,
			Action:       rbac.DeleteOverride,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Ratelimit,
			ResourceID:   "*",
			Action:       rbac.DeleteOverride,
		}),
	)))
	if err != nil {
		return err
	}

	override, ok := namespace.DirectOverrides[req.Identifier]
	if !ok {
		return fault.New("override not found",
			fault.Code(codes.Data.RatelimitOverride.NotFound.URN()),
			fault.Internal("override not found"),
			fault.Public("This override does not exist."),
		)
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		err = db.Query.SoftDeleteRatelimitOverride(ctx, tx, db.SoftDeleteRatelimitOverrideParams{
			ID:  override.ID,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to soft delete ratelimit override"),
				fault.Public("The database is unavailable."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.RatelimitDeleteOverrideEvent,
				Display:     fmt.Sprintf("Deleted override %s.", override.ID),
				ActorID:     auth.Key.ID,
				ActorType:   auditlog.RootKeyActor,
				ActorName:   "root key",
				ActorMeta:   map[string]any{},
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						ID:          override.ID,
						Name:        override.Identifier,
						DisplayName: override.Identifier,
						Type:        auditlog.RatelimitOverrideResourceType,
						Meta:        nil,
					},
					{
						ID:          namespace.ID,
						Name:        namespace.Name,
						DisplayName: namespace.Name,
						Type:        auditlog.RatelimitNamespaceResourceType,
						Meta:        nil,
					},
				},
			},
		})
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	h.RatelimitNamespaceCache.Remove(
		ctx,
		cache.ScopedKey{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Key:         namespace.ID,
		},
		cache.ScopedKey{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Key:         namespace.Name,
		},
	)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2RatelimitDeleteOverrideResponseData{},
	})
}
