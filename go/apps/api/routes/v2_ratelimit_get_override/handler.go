package handler

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/match"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitGetOverrideRequestBody
type Response = openapi.V2RatelimitGetOverrideResponseBody

// Handler implements zen.Route interface for the v2 ratelimit get override endpoint
type Handler struct {
	// Services as public fields
	Logger                        logging.Logger
	DB                            db.Database
	Keys                          keys.KeyService
	RatelimitNamespaceByNameCache cache.Cache[string, db.FindRatelimitNamespace]
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
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	// nolint:exhaustruct
	req := Request{}
	err = s.BindBody(&req)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
		)
	}

	response, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Name:        sql.NullString{String: ptr.SafeDeref(req.NamespaceName), Valid: req.NamespaceName != nil},
		ID:          sql.NullString{String: ptr.SafeDeref(req.NamespaceId), Valid: req.NamespaceId != nil},
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("namespace not found",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Internal("namespace not found"), fault.Public("The namespace was not found."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to find the namespace"), fault.Public("Error finding the ratelimit namespace."),
		)
	}

	namespace := db.FindRatelimitNamespace{
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
	err = json.Unmarshal(response.Overrides.([]byte), &overrides)
	if err != nil {
		return fault.Wrap(err,
			fault.Internal("unable to unmarshal ratelimit overrides"),
			fault.Public("We're unable to parse the ratelimits overrides."),
		)
	}

	for _, override := range overrides {
		namespace.DirectOverrides[override.Identifier] = override
		if strings.Contains(override.Identifier, "*") {
			namespace.WildcardOverrides = append(namespace.WildcardOverrides, override)
		}
	}

	err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
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
			OverrideId:  override.ID,
			NamespaceId: namespace.ID,
			Limit:       override.Limit,
			Duration:    override.Duration,
			Identifier:  override.Identifier,
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
