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

type Request = openapi.V2PermissionsGetPermissionRequestBody
type Response = openapi.V2PermissionsGetPermissionResponseBody

// Handler implements zen.Route interface for the v2 permissions get permission endpoint
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
	return "/v2/permissions.getPermission"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	h.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.getPermission")

	// 1. Authentication
	auth, err := h.Keys.VerifyRootKey(ctx, s)
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// 3. Permission check
	err = h.Permissions.Check(
		ctx,
		auth.KeyID,
		rbac.Or(
			rbac.T(rbac.Tuple{
				ResourceType: rbac.Rbac,
				ResourceID:   "*",
				Action:       rbac.ReadPermission,
			}),
		),
	)
	if err != nil {
		return err
	}

	// 4. Get permission by ID
	permission, err := db.Query.FindPermissionByID(ctx, h.DB.RO(), req.PermissionId)
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Internal("permission not found"), fault.Public("The requested permission does not exist."),
			)
		}
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"), fault.Public("Failed to retrieve permission information."),
		)
	}

	// 5. Check if permission belongs to authorized workspace
	if permission.WorkspaceID != auth.AuthorizedWorkspaceID {
		return fault.New("permission does not belong to authorized workspace",
			fault.Code(codes.Data.Permission.NotFound.URN()),
			fault.Public("The requested permission does not exist."),
		)
	}

	// 6. Return success response
	permissionResponse := openapi.Permission{
		Id:          permission.ID,
		Name:        permission.Name,
		Description: nil,
		CreatedAt:   permission.CreatedAtM,
	}

	// Add description only if it's valid
	if permission.Description.Valid {
		permissionResponse.Description = &permission.Description.String
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.PermissionsGetPermissionResponseData{
			Permission: permissionResponse,
		},
	})
}
