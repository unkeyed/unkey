package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/caches"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2ApisGetApiRequestBody
type Response = openapi.V2ApisGetApiResponseBody

// Handler implements zen.Route interface for the v2 APIs get API endpoint
type Handler struct {
	DB     db.Database
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
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	api, hit, err := h.Caches.LiveApiByID.SWR(ctx, cache.ScopedKey{WorkspaceID: principal.AuthorizedWorkspaceID, Key: req.ApiId}, func(ctx context.Context) (db.FindLiveApiByIDRow, error) {
		return db.Query.FindLiveApiByID(ctx, h.DB.RO(), req.ApiId)
	}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return apierrors.NotFound(codes.Data.Api.NotFound.URN())
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve API information."),
		)
	}

	if hit == cache.Null {
		return apierrors.NotFound(codes.Data.Api.NotFound.URN())
	}

	// Check if API belongs to the authorized workspace
	if api.ApiWorkspaceID != principal.AuthorizedWorkspaceID {
		return apierrors.NotFound(codes.Data.Api.NotFound.URN())
	}

	if !api.ApiKeyAuthID.Valid {
		return fault.New("api missing keyspace",
			fault.Code(codes.App.Internal.UnexpectedError.URN()),
			fault.Internal("api has no key auth id"),
			fault.Public("Failed to retrieve API information."),
		)
	}

	permissionQuery := rbac.Or(
		rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(api.KeyAuthProjectID).Keyspace(api.ApiKeyAuthID.String),
			permissions.Read,
		),
	)
	err = principal.Authorize(permissionQuery)
	if err != nil {
		// The authorization runs after the lookup because the URN arm needs the
		// keyspace ID. Callers who cannot read the API get the same 404 as a
		// missing API. This query is itself a read, so a denial stays a 404.
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			principal.Permissions,
			permissionQuery,
			codes.Data.Api.NotFound.URN(),
			"Resource does not exist.",
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2ApisGetApiResponseData{
			Id:   api.ApiID,
			Name: api.ApiName,
		},
	})
}
