package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitListOverridesRequestBody
type Response = openapi.V2RatelimitListOverridesResponseBody

// Handler implements zen.Route interface for the v2 ratelimit list overrides endpoint
type Handler struct {
	// Services as public fields
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
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
	auth, err := h.Keys.VerifyRootKey(ctx, s)
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

	namespace, err := getNamespace(ctx, h, auth.AuthorizedWorkspaceID, req)

	if db.IsNotFound(err) {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace not found"), fault.Public("This namespace does not exist."),
		)
	}
	if err != nil {
		return err
	}

	if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace was deleted"), fault.Public("This namespace does not exist."),
		)
	}

	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(
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
		),
	)
	if err != nil {
		return err
	}

	overrides, err := db.Query.ListRatelimitOverridesByNamespaceID(ctx, h.DB.RO(), db.ListRatelimitOverridesByNamespaceIDParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		NamespaceID: namespace.ID,
	})

	if err != nil {
		return err
	}
	response := Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: make([]openapi.RatelimitOverride, len(overrides)),
		Pagination: &openapi.Pagination{
			Cursor:  nil,
			HasMore: false,
		},
	}

	for i, override := range overrides {
		response.Data[i] = openapi.RatelimitOverride{
			OverrideId:  override.ID,
			Duration:    int64(override.Duration),
			Identifier:  override.Identifier,
			NamespaceId: override.NamespaceID,
			Limit:       int64(override.Limit),
		}
	}

	return s.JSON(http.StatusOK, response)
}

func getNamespace(ctx context.Context, h *Handler, workspaceID string, req Request) (db.RatelimitNamespace, error) {

	switch {
	case req.NamespaceId != nil:
		{
			return db.Query.FindRatelimitNamespaceByID(ctx, h.DB.RO(), *req.NamespaceId)

		}
	case req.NamespaceName != nil:
		{
			return db.Query.FindRatelimitNamespaceByName(ctx, h.DB.RO(), db.FindRatelimitNamespaceByNameParams{
				WorkspaceID: workspaceID,
				Name:        *req.NamespaceName,
			})
		}
	}

	return db.RatelimitNamespace{}, fault.New("missing namespace id or name",
		fault.Code(codes.App.Validation.InvalidInput.URN()),
		fault.Internal("missing namespace id or name"), fault.Public("You must provide either a namespace ID or name."),
	)

}
