package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2RatelimitGetOverrideRequestBody
type Response = openapi.V2RatelimitGetOverrideResponseBody

// Handler implements zen.Route interface for the v2 ratelimit get override endpoint
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

	namespace, err := getNamespace(ctx, h, auth.AuthorizedWorkspaceID, req)

	if err != nil {
		// already handled correctly in getNamespace
		return err
	}

	if namespace.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("namespace not found",
			fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
			fault.Internal("namespace was deleted"), fault.Public("This namespace does not exist."),
		)
	}

	auth, err = auth.WithPermissions(ctx, rbac.Or(
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
	)).Result()
	if err != nil {
		return err
	}

	override, err := db.Query.FindRatelimitOverrideByIdentifier(ctx, h.DB.RO(), db.FindRatelimitOverrideByIdentifierParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		NamespaceID: namespace.ID,
		Identifier:  req.Identifier,
	})

	if db.IsNotFound(err) {
		return fault.New("override not found",
			fault.Code(codes.Data.RatelimitOverride.NotFound.URN()),
			fault.Internal("override not found"), fault.Public("This override does not exist."),
		)
	}
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to find the override"), fault.Public("Error finding the ratelimit override."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.RatelimitOverride{

			OverrideId:  override.ID,
			Duration:    int64(override.Duration),
			Identifier:  override.Identifier,
			NamespaceId: override.NamespaceID,
			Limit:       int64(override.Limit),
		},
	})
}

func getNamespace(ctx context.Context, h *Handler, workspaceID string, req Request) (namespace db.RatelimitNamespace, err error) {

	switch {
	case req.NamespaceId != nil:
		{
			namespace, err = db.Query.FindRatelimitNamespaceByID(ctx, h.DB.RO(), *req.NamespaceId)
			break
		}
	case req.NamespaceName != nil:
		{
			namespace, err = db.Query.FindRatelimitNamespaceByName(ctx, h.DB.RO(), db.FindRatelimitNamespaceByNameParams{
				WorkspaceID: workspaceID,
				Name:        *req.NamespaceName,
			})
			break
		}
	default:
		return db.RatelimitNamespace{}, fault.New("namespace id or name required",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("namespace id or name required"), fault.Public("You must provide either a namespace ID or name."),
		)
	}

	if err != nil {

		if db.IsNotFound(err) {
			return db.RatelimitNamespace{}, fault.New("namespace not found",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Internal("namespace not found"), fault.Public("The namespace was not found."),
			)
		}

		return db.RatelimitNamespace{}, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database failed to find the namespace"), fault.Public("Error finding the ratelimit namespace."),
		)
	}
	return namespace, nil
}
