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
	"github.com/unkeyed/unkey/go/pkg/ptr"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsListPermissionsRequestBody
type Response = openapi.V2PermissionsListPermissionsResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.listPermissions", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.listPermissions")

		// 1. Authentication
		auth, err := svc.Keys.VerifyRootKey(ctx, s)
		if err != nil {
			return err
		}

		// 2. Request validation
		var req Request
		err = s.BindBody(&req)
		if err != nil {
			return fault.Wrap(err,
				fault.WithDesc("invalid request body", "The request body is invalid."),
			)
		}

		limit := ptr.SafeDeref(req.Limit, 100)

		// Handle null cursor
		cursor := ptr.SafeDeref(req.Cursor, "")

		// 3. Permission check
		permissionCheck, err := svc.Permissions.Check(
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
			return fault.Wrap(err,
				fault.WithDesc("unable to check permissions", "We're unable to check the permissions of your key."),
			)
		}

		if !permissionCheck.Valid {
			return fault.New("insufficient permissions",
				fault.WithCode(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.WithDesc(permissionCheck.Message, permissionCheck.Message),
			)
		}

		// 4. Query permissions with pagination
		permissions, total, err := db.Query.ListPermissions(
			ctx,
			svc.DB.RO(),
			db.ListPermissionsParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Cursor:      cursor,
				Limit:       limit,
			},
		)
		if err != nil {
			return fault.Wrap(err,
				fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
				fault.WithDesc("database error", "Failed to retrieve permissions."),
			)
		}

		// 5. Transform permissions into response format
		responsePermissions := make([]openapi.Permission, 0, len(permissions))
		for _, perm := range permissions {
			responsePermissions = append(responsePermissions, openapi.Permission{
				Id:          perm.ID,
				Name:        perm.Name,
				WorkspaceId: perm.WorkspaceID,
				Description: perm.Description.String,
				CreatedAt:   perm.CreatedAtM.Time.Format(http.TimeFormat),
			})
		}

		// 6. Determine next cursor
		var nextCursor *string
		if len(permissions) > 0 && len(permissions) == limit {
			cursor := permissions[len(permissions)-1].ID
			nextCursor = &cursor
		}

		// 7. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.PermissionsListPermissionsResponseData{
				Permissions: responsePermissions,
				Total:       int(total),
				Cursor:      nextCursor,
			},
		})
	})
}
