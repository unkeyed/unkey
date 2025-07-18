package handler

import (
	"context"
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
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitSetOverrideRequestBody
type Response = openapi.V2RatelimitSetOverrideResponseBody

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
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Use the namespace field directly - it can be either name or ID
	response, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Namespace:   req.Namespace,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("namespace not found",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Internal("namespace not found"), fault.Public("This namespace does not exist."),
			)
		}
		return err
	}

	namespace := db.RatelimitNamespace{
		ID:          response.ID,
		WorkspaceID: response.WorkspaceID,
		Name:        response.Name,
		CreatedAtM:  response.CreatedAtM,
		UpdatedAtM:  response.UpdatedAtM,
		DeletedAtM:  response.DeletedAtM,
	}

	if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"), fault.Public("This namespace does not exist."),
		)
	}

	err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
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
		return fault.Wrap(err,
			fault.Internal("unable to check permissions"), fault.Public("We're unable to check the permissions of your key."),
		)
	}

	overrideID, err := db.TxWithResult(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (string, error) {
		overrideID := uid.New(uid.RatelimitOverridePrefix)
		err = db.Query.InsertRatelimitOverride(ctx, tx, db.InsertRatelimitOverrideParams{
			ID:          overrideID,
			WorkspaceID: auth.AuthorizedWorkspaceID,
			NamespaceID: namespace.ID,
			Identifier:  req.Identifier,
			Limit:       int32(req.Limit),    // nolint:gosec
			Duration:    int32(req.Duration), //nolint:gosec
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			return "", fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed"), fault.Public("The database is unavailable."),
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
		Data: openapi.RatelimitSetOverrideResponseData{
			OverrideId: overrideID,
		},
	})
}
