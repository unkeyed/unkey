package handler

import (
	"context"
	"net/http"
	"time"

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

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.getPermission", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.getPermission")

		// 1. Authentication
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// 2. Request validation
		req, err := zen.BindBody[Request](s)
		if err != nil {
			return err
		}

		// 3. Permission check
		err = svc.Permissions.Check(
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
		permission, err := db.Query.FindPermissionByID(ctx, svc.DB.RO(), req.PermissionId)
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
		createdAt := time.UnixMilli(permission.CreatedAtM).UTC()
		permissionResponse := openapi.Permission{
			Id:          permission.ID,
			Name:        permission.Name,
			WorkspaceId: permission.WorkspaceID,
			CreatedAt:   &createdAt,
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
	})
}
