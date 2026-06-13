package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/ratelimit/namespace"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/match"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitGetOverrideRequestBody
	Response = openapi.V2RatelimitGetOverrideResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit get override endpoint
type Handler struct {
	DB             db.Database
	NamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
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
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	ns, found, err := h.getNamespace(ctx, principal.WorkspaceID, req.Namespace)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while fetching the namespace."))
	}

	if !found {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Public("This namespace does not exist."),
		)
	}

	if ns.DeletedAtM.Valid {
		return fault.New("namespace was deleted",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Public("This namespace does not exist."),
		)
	}

	override, overrideFound, err := matchOverride(req.Identifier, ns)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("error matching overrides"), fault.Public("Error matching ratelimit override"),
		)
	}

	// A missing override and an override the principal may not read produce
	// the same 404, so the response never distinguishes "does not exist" from
	// "exists but is forbidden". Answering them differently would let a caller
	// without read_override permission enumerate which identifiers have
	// overrides by telling 404s apart from 403s. Because of that, the
	// not-found branch needs no permission check at all, and the found branch
	// masks authorization failures as the identical 404.
	if !overrideFound {
		return fault.New("override not found",
			fault.Code(codes.Data.RatelimitOverride.NotFound.URN()),
			fault.Internal("no override matched the identifier"),
			fault.Public("This override does not exist."),
		)
	}

	// The URN leg is the canonical permission; the two tuple legs accept legacy
	// namespace-scoped and global root-key grants until those are migrated to URNs.
	err = principal.Authorize(
		rbac.Or(
			rbac.U(
				urn.Build().
					Workspace(principal.WorkspaceID).
					RatelimitNamespace(ns.ID).
					Override(override.ID),
				rbac.ReadOverride,
			),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   ns.ID,
				Action:       rbac.ReadOverride,
			}),
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Ratelimit,
				ResourceID:   "*",
				Action:       rbac.ReadOverride,
			}),
		))
	if err != nil {
		// Deliberately not fault.Wrap(err): the error middleware joins every
		// public message in the chain into the response detail, and the rbac
		// rejection's public message names the override ID, which would leak
		// the existence this 404 is masking.
		return fault.New("override exists but principal may not read it",
			fault.Code(codes.Data.RatelimitOverride.NotFound.URN()),
			fault.Internal("masking insufficient permissions as not found"),
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

func (h *Handler) getNamespace(ctx context.Context, workspaceID, nameOrID string) (db.FindRatelimitNamespace, bool, error) {
	cacheKey := cache.ScopedKey{WorkspaceID: workspaceID, Key: nameOrID}

	ns, hit, err := h.NamespaceCache.SWR(ctx, cacheKey, func(ctx context.Context) (db.FindRatelimitNamespace, error) {
		row, dbErr := db.WithRetryContext(ctx, func() (db.FindRatelimitNamespaceRow, error) {
			return db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
				WorkspaceID: workspaceID,
				Namespace:   nameOrID,
			})
		})
		if dbErr != nil {
			return db.FindRatelimitNamespace{}, dbErr //nolint:exhaustruct
		}
		return namespace.ParseNamespaceRow(row), nil
	}, caches.DefaultFindFirstOp)
	if err != nil {
		if db.IsNotFound(err) {
			return db.FindRatelimitNamespace{}, false, nil //nolint:exhaustruct
		}
		return db.FindRatelimitNamespace{}, false, err //nolint:exhaustruct
	}

	if hit == cache.Null {
		return db.FindRatelimitNamespace{}, false, nil //nolint:exhaustruct
	}

	return ns, true, nil
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

	return db.FindRatelimitNamespaceLimitOverride{
		ID:         "",
		Limit:      0,
		Identifier: "",
		Duration:   0,
	}, false, nil
}
