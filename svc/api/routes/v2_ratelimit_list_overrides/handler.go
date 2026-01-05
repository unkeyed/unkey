package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitListOverridesRequestBody
	Response = openapi.V2RatelimitListOverridesResponseBody
)

// Handler implements zen.Route interface for the v2 ratelimit list overrides endpoint
type Handler struct {
	// Services as public fields
	Logger logging.Logger
	DB     db.Database
	Keys   keys.KeyService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/ratelimit.listOverrides"
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

	// Use the namespace field directly - it can be either name or ID
	namespace, err := db.Query.FindRatelimitNamespace(ctx, h.DB.RO(), db.FindRatelimitNamespaceParams{
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
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Public("An unexpected error occurred while loading your namespace."),
		)
	}

	if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace was deleted"), fault.Public("This namespace does not exist."),
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

	limit := ptr.SafeDeref(req.Limit, 50)

	overrides, err := db.Query.ListRatelimitOverridesByNamespaceID(ctx, h.DB.RO(), db.ListRatelimitOverridesByNamespaceIDParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		NamespaceID: namespace.ID,
		//nolint:gosec
		Limit:    int32(limit) + 1,
		CursorID: ptr.SafeDeref(req.Cursor, ""),
	})
	if err != nil {
		return err
	}

	hasMore := len(overrides) > limit
	var cursor *string
	if hasMore {
		cursor = ptr.P(overrides[limit].ID)
		overrides = overrides[:limit]
	}

	responseBody := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: make([]openapi.RatelimitOverride, len(overrides)),
		Pagination: &openapi.Pagination{
			Cursor:  cursor,
			HasMore: hasMore,
		},
	}

	for i, override := range overrides {
		responseBody.Data[i] = openapi.RatelimitOverride{
			OverrideId: override.ID,
			Duration:   int64(override.Duration),
			Identifier: override.Identifier,
			Limit:      int64(override.Limit),
		}
	}

	return s.JSON(http.StatusOK, responseBody)
}
