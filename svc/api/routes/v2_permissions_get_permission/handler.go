package handler

import (
	"context"
	"net/http"

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

type Request = openapi.V2PermissionsGetPermissionRequestBody
type Response = openapi.V2PermissionsGetPermissionResponseBody

// Handler implements zen.Route interface for the v2 permissions get permission endpoint
type Handler struct {
	DB db.Database
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

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	permission, err := db.Query.FindPermissionByIdOrSlug(ctx, h.DB.RO(), db.FindPermissionByIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Search:      req.Permission,
	})
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

	readPermission := rbac.U(
		urn.New().Workspace(principal.WorkspaceID).Project(permission.ProjectID).RBAC().Permission(permission.ID),
		permissions.Read,
	)
	err = principal.Authorize(rbac.Or(
		readPermission,
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.ReadPermission,
		}),
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(
			err,
			codes.Data.Permission.NotFound.URN(),
			"The requested permission does not exist.",
		)
	}

	permissionResponse := openapi.Permission{
		Id:          permission.ID,
		Name:        permission.Name,
		Slug:        permission.Slug,
		Description: permission.Description.String,
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: permissionResponse,
	})
}
