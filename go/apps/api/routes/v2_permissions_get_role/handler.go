package handler

import (
	"context"
	"encoding/json"
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

type Request = openapi.V2PermissionsGetRoleRequestBody
type Response = openapi.V2PermissionsGetRoleResponseBody

// Handler implements zen.Route interface for the v2 permissions get role endpoint
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
	return "/v2/permissions.getRole"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.getRole")

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
		Description: nil,
	}

	if role.Description.Valid {
		roleResponse.Description = &role.Description.String
	}

	rolePermissions := make([]db.Permission, 0)
	if permBytes, ok := role.Permissions.([]byte); ok && permBytes != nil {
		_ = json.Unmarshal(permBytes, &rolePermissions) // Ignore error, default to empty array
	}

	perms := make([]openapi.Permission, 0)
	for _, perm := range rolePermissions {
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

		perms = append(perms, permission)
	}

	if len(perms) > 0 {
		roleResponse.Permissions = ptr.P(perms)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: roleResponse,
	})
}
