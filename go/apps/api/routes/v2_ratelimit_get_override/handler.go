package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/match"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitGetOverrideRequestBody
type Response = openapi.V2RatelimitGetOverrideResponseBody

// Handler implements zen.Route interface for the v2 ratelimit get override endpoint
type Handler struct {
	// Services as public fields
	Logger                  logging.Logger
	DB                      db.Database
	Keys                    keys.KeyService
	RatelimitNamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
}

// decodeOverrides safely decodes JSON bytes into override slice with proper error handling
func decodeOverrides(data interface{}) ([]db.FindRatelimitNamespaceLimitOverride, error) {
	overrides := make([]db.FindRatelimitNamespaceLimitOverride, 0)
	if overrideBytes, ok := data.([]byte); ok && overrideBytes != nil {
		if err := json.Unmarshal(overrideBytes, &overrides); err != nil {
			return nil, fault.Wrap(err,
				fault.Code(codes.App.Internal.UnexpectedError.URN()),
				fault.Public("An unexpected error occurred while processing override data."))
		}
	}
	return overrides, nil
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.getOverride"
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

	namespace, hit, err := h.RatelimitNamespaceCache.SWR(ctx,
		cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: req.Namespace},
		func(ctx context.Context) (db.FindRatelimitNamespace, error) {
			response, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Namespace:   req.Namespace,
			})
			if err != nil {
				return db.FindRatelimitNamespace{}, err
			}

			result := db.FindRatelimitNamespace{
				ID:                response.ID,
				WorkspaceID:       response.WorkspaceID,
				Name:              response.Name,
				CreatedAtM:        response.CreatedAtM,
				UpdatedAtM:        response.UpdatedAtM,
				DeletedAtM:        response.DeletedAtM,
				DirectOverrides:   make(map[string]db.FindRatelimitNamespaceLimitOverride),
				WildcardOverrides: make([]db.FindRatelimitNamespaceLimitOverride, 0),
			}

			overrides, err := decodeOverrides(response.Overrides)
			if err != nil {
				return result, err
			}

			for _, override := range overrides {
				result.DirectOverrides[override.Identifier] = override
				if strings.Contains(override.Identifier, "*") {
					result.WildcardOverrides = append(result.WildcardOverrides, override)
				}
			}

			return result, nil
		}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("namespace was deleted",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Public("This namespace does not exist."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching the namespace."))
	}

	if hit == cache.Null {
		return fault.New("namespace cache null",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Public("This namespace does not exist."),
		)
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
			Action:       rbac.ReadOverride,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Ratelimit,
			ResourceID:   "*",
			Action:       rbac.ReadOverride,
		}),
	)))
	if err != nil {
		return err
	}

	override, found, err := matchOverride(req.Identifier, namespace)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("error matching overrides"), fault.Public("Error matching ratelimit override"),
		)
	}

	if !found {
		return fault.New("override not found",
			fault.Code(codes.Data.RatelimitOverride.NotFound.URN()),
			fault.Public("This override does not exist."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.RatelimitOverride{
			OverrideId: override.ID,
			Limit:      override.Limit,
			Duration:   override.Duration,
			Identifier: override.Identifier,
		},
	})
}

func matchOverride(identifier string, namespace db.FindRatelimitNamespace) (db.FindRatelimitNamespaceLimitOverride, bool, error) {
	if override, ok := namespace.DirectOverrides[identifier]; ok {
		return override, true, nil
	}

	for _, override := range namespace.WildcardOverrides {
		ok, err := match.Wildcard(identifier, override.Identifier)
		if err != nil {
			return db.FindRatelimitNamespaceLimitOverride{}, false, err
		}

		if !ok {
			continue
		}

		return override, true, nil
	}

	return db.FindRatelimitNamespaceLimitOverride{}, false, nil
}
