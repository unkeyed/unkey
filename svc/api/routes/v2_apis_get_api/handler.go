package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/wide"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2ApisGetApiRequestBody
type Response = openapi.V2ApisGetApiResponseBody

// Handler implements zen.Route interface for the v2 APIs get API endpoint
type Handler struct {
	// Services as public fields
	Logger logging.Logger
	DB     db.Database
	Keys   keys.KeyService
	Caches caches.Caches
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/apis.getApi"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/apis.getApi")
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   "*",
			Action:       rbac.ReadAPI,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Api,
			ResourceID:   req.ApiId,
			Action:       rbac.ReadAPI,
		}),
	)))
	if err != nil {
		return err
	}

	api, hit, err := h.Caches.LiveApiByID.SWR(ctx, cache.ScopedKey{WorkspaceID: auth.AuthorizedWorkspaceID, Key: req.ApiId}, func(ctx context.Context) (db.FindLiveApiByIDRow, error) {
		return db.Query.FindLiveApiByID(ctx, h.DB.RO(), req.ApiId)
	}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("api not found",
				fault.Code(codes.Data.Api.NotFound.URN()),
				fault.Internal("api not found"), fault.Public("The requested API does not exist or has been deleted."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve API information."),
		)
	}

	if hit == cache.Null {
		return fault.New("api not found",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("api not found"),
			fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	// Check if API belongs to the authorized workspace
	if api.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("wrong workspace",
			fault.Code(codes.Data.Api.NotFound.URN()),
			fault.Internal("wrong workspace, masking as 404"), fault.Public("The requested API does not exist or has been deleted."),
		)
	}

	wide.Set(ctx, wide.FieldAPIID, api.ID)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2ApisGetApiResponseData{
			Id:   api.ID,
			Name: api.Name,
		},
	})
}
