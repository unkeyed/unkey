package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/keys"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PermissionsGetRoleRequestBody
	Response = openapi.V2PermissionsGetRoleResponseBody
)

// Handler implements zen.Route interface for the v2 permissions get role endpoint
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
	return "/v2/permissions.getRole"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.getRole")

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

	role, err := db.Query.FindRoleByIdOrNameWithPerms(ctx, h.DB.RO(), db.FindRoleByIdOrNameWithPermsParams{
		WorkspaceID: auth.AuthorizedWorkspaceID,
		Search:      req.Role,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role not found"), fault.Public("The requested role does not exist."),
			)
		}

		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve role information."),
		)
	}

	roleResponse := openapi.Role{
		Id:          role.ID,
		Name:        role.Name,
		Permissions: nil,
		Description: role.Description.String,
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

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: roleResponse,
	})
}
