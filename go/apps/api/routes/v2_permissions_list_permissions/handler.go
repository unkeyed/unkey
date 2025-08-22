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
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsListPermissionsRequestBody
type Response = openapi.V2PermissionsListPermissionsResponseBody

// Handler implements zen.Route interface for the v2 permissions list permissions endpoint
type Handler struct {
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
	return "/v2/permissions.listPermissions"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.listPermissions")

	// 1. Authentication
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	cursor := ptr.SafeDeref(req.Cursor, "")
	limit := ptr.SafeDeref(req.Limit, 100)

	err = auth.VerifyRootKey(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.ReadPermission,
		}),
	)))
	if err != nil {
		return err
	}

	permissions, err := db.Query.ListPermissions(
		ctx,
		h.DB.RO(),
		db.ListPermissionsParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			IDCursor:    cursor,
			Limit:       int32(limit) + 1,
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve permissions."),
		)
	}

	hasMore := len(permissions) > limit
	var nextCursor *string

	if hasMore {
		nextCursor = ptr.P(permissions[limit].ID)
		permissions = permissions[:limit]
	}

	responsePermissions := make([]openapi.Permission, 0, len(permissions))
	for _, perm := range permissions {
		permission := openapi.Permission{
			Id:          perm.ID,
			Name:        perm.Name,
			Slug:        perm.Slug,
			Description: nil,
		}

		// Add description only if it's valid
		if perm.Description.Valid {
			permission.Description = &perm.Description.String
		}

		responsePermissions = append(responsePermissions, permission)
	}

	// 7. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: responsePermissions,
		Pagination: &openapi.Pagination{
			Cursor:  nextCursor,
			HasMore: hasMore,
		},
	})
}
