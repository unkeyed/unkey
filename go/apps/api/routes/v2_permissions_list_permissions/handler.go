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
	return "/v2/permissions.listPermissions"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.listPermissions")

	// 1. Authentication
	auth, err := h.Keys.GetRootKey(ctx, s)
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// Handle null cursor - use empty string to start from beginning
	cursor := ptr.SafeDeref(req.Cursor, "")

	// 3. Permission check
	err = auth.Verify(ctx, keys.WithPermissions(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.ReadPermission,
		}),
	)))
	if err != nil {
		return err
	}

	// 4. Query permissions with pagination
	permissions, err := db.Query.ListPermissions(
		ctx,
		h.DB.RO(),
		db.ListPermissionsParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			IDCursor:    cursor,
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve permissions."),
		)
	}

	// Check if we have more results by seeing if we got 101 permissions
	hasMore := len(permissions) > 100
	var nextCursor *string

	// If we have more than 100, truncate to 100
	if hasMore {
		nextCursor = ptr.P(permissions[100].ID)
		permissions = permissions[:100]
	}

	// 5. Transform permissions into response format
	responsePermissions := make([]openapi.Permission, 0, len(permissions))
	for _, perm := range permissions {
		permission := openapi.Permission{
			Id:          perm.ID,
			Name:        perm.Name,
			Description: nil,
			CreatedAt:   perm.CreatedAtM,
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
