package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitDeleteOverrideRequestBody
type Response = openapi.V2RatelimitDeleteOverrideResponseBody

// Handler implements zen.Route interface for the v2 ratelimit delete override endpoint
type Handler struct {
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
	return "/v2/ratelimit.deleteOverride"
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

	err = db.Tx(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		namespace, err := db.Query.FindRatelimitNamespace(ctx, tx, db.FindRatelimitNamespaceParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			Namespace:   req.Namespace,
		})
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("namespace not found",
					fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
					fault.Internal("namespace not found"),
					fault.Public("This namespace does not exist."),
				)
			}
			return err
		}

		if namespace.DeletedAtM.Valid {
			return fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
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
		// Check if the override exists before deleting
		override, overrideErr := db.Query.FindRatelimitOverrideByIdentifier(ctx, tx, db.FindRatelimitOverrideByIdentifierParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
		})

		if db.IsNotFound(overrideErr) {
			return fault.New("override not found",
				fault.Code(codes.Data.RatelimitOverride.NotFound.URN()),
				fault.Internal("override not found"),
				fault.Public("This override does not exist."),
			)
		}
		if overrideErr != nil {
			return overrideErr
		}

		// Perform soft delete by updating the DeletedAt field
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

		return nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2RatelimitDeleteOverrideResponseData{},
	})
}
