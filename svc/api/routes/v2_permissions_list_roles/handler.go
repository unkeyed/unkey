package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PermissionsListRolesRequestBody
	Response = openapi.V2PermissionsListRolesResponseBody
)

// Handler implements zen.Route interface for the v2 permissions list roles endpoint
type Handler struct {
	DB   db.Database
	Keys keys.KeyService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/permissions.listRoles"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.listRoles")

	// 1. Authentication
	auth, emit, err := h.Keys.GetRootKey(ctx, s)
	defer emit()
	if err != nil {
		return err
	}

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
			Action:       rbac.ReadRole,
		}),
	)))
	if err != nil {
		return err
	}

	roles, err := db.Query.ListRoles(
		ctx,
		h.DB.RO(),
		db.ListRolesParams{
			WorkspaceID: auth.AuthorizedWorkspaceID,
			IDCursor:    cursor,
			//nolint:gosec
			Limit: int32(limit) + 1,
		},
	)
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve roles."),
		)
	}

	var nextCursor *string
	hasMore := len(roles) > limit
	if hasMore {
		nextCursor = ptr.P(roles[limit].ID)
		roles = roles[:limit]
	}

	roleResponses := make([]openapi.Role, 0, len(roles))
	for _, role := range roles {
		roleResponse := openapi.Role{
			Id:          role.ID,
			Name:        role.Name,
			Description: role.Description.String,
			Permissions: nil,
		}

		perms, err := db.UnmarshalNullableJSONTo[[]db.Permission](role.Permissions)
		if err != nil {
			logger.Error("Failed to unmarshal permissions", "error", err)
		}

		for _, perm := range perms {
			permission := openapi.Permission{
				Id:          perm.ID,
				Name:        perm.Name,
				Slug:        perm.Slug,
				Description: perm.Description.String,
			}

			roleResponse.Permissions = append(roleResponse.Permissions, permission)
		}

		roleResponses = append(roleResponses, roleResponse)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: roleResponses,
		Pagination: &openapi.Pagination{
			Cursor:  nextCursor,
			HasMore: hasMore,
		},
	})
}
