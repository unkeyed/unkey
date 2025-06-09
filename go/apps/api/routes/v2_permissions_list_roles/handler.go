package handler

import (
	"context"
	"net/http"
	"strconv"

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

type Request = openapi.V2PermissionsListRolesRequestBody
type Response = openapi.V2PermissionsListRolesResponseBody

const (
	defaultLimit = 10
	maxLimit     = 100
)

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.listRoles", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.listRoles")

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
				fault.Internal("invalid request body"), fault.Public("The request body is invalid."),
			)
		}

		// 3. Permission check
		err = svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Role,
					ResourceID:   "*",
					Action:       rbac.ListRoles,
				}),
			),
		)
		if err != nil {
			return err
		}

		// 4. Determine pagination parameters
		var limit int32 = defaultLimit
		if req.Limit != nil {
			limit = *req.Limit
			if limit <= 0 {
				limit = defaultLimit
			}
			if limit > maxLimit {
				limit = maxLimit
			}
		}

		var cursor string
		if req.Cursor != nil {
			cursor = *req.Cursor
		}

		// 5. Query roles with pagination
		roles, nextCursor, total, err := listRolesWithPermissions(ctx, svc.DB, auth.AuthorizedWorkspaceID, cursor, limit)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve roles."),
			)
		}

		// 6. Transform roles to response format
		roleResponses := make([]openapi.RoleWithPermissions, 0, len(roles))
		for _, role := range roles {
			roleResponses = append(roleResponses, role)
		}

		// 7. Return paginated list of roles
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.PermissionsListRolesResponseData{
				Roles:  roleResponses,
				Total:  int(total),
				Cursor: nextCursor,
			},
		})
	})
}

// listRolesWithPermissions queries roles with their associated permissions, applying pagination
func listRolesWithPermissions(ctx context.Context, database db.Database, workspaceID, cursor string, limit int32) ([]openapi.RoleWithPermissions, *string, int64, error) {
	// 1. Get total count of roles in the workspace
	total, err := db.Query.CountRolesByWorkspaceId(ctx, database.RO(), workspaceID)
	if err != nil {
		return nil, nil, 0, err
	}

	// 2. Get roles with pagination
	dbRoles, err := db.Query.FindRolesByWorkspaceIdWithPagination(ctx, database.RO(), db.FindRolesByWorkspaceIdWithPaginationParams{
		WorkspaceID: workspaceID,
		Cursor:      cursor,
		Limit:       int32(limit + 1), // request one more to determine if there are more results
	})
	if err != nil {
		return nil, nil, 0, err
	}

	// 3. Determine if there are more results and create next cursor
	var nextCursor *string
	hasMore := len(dbRoles) > int(limit)
	if hasMore {
		dbRoles = dbRoles[:limit] // Remove the extra item
		lastRole := dbRoles[len(dbRoles)-1]
		nextCursorVal := lastRole.ID
		nextCursor = &nextCursorVal
	}

	// 4. Build map of role IDs to get permissions
	roleIDs := make([]string, len(dbRoles))
	for i, role := range dbRoles {
		roleIDs = append(roleIDs, role.ID)
	}

	// 5. Get all permissions for the roles
	var allRolePermissions []db.FindPermissionsByRoleIdsRow
	if len(roleIDs) > 0 {
		allRolePermissions, err = db.Query.FindPermissionsByRoleIds(ctx, database.RO(), roleIDs)
		if err != nil {
			return nil, nil, 0, err
		}
	}

	// 6. Create a map of role ID to permissions
	rolePermissionsMap := make(map[string][]openapi.Permission)
	for _, rp := range allRolePermissions {
		permission := openapi.Permission{
			Id:          rp.PermissionID,
			Name:        rp.PermissionName,
			WorkspaceId: rp.PermissionWorkspaceID,
			Description: &rp.PermissionDescription.String,
		}

		if rp.PermissionCreatedAt.Valid {
			createdAt := rp.PermissionCreatedAt.Time.Format(http.TimeFormat)
			permission.CreatedAt = &createdAt
		}

		rolePermissionsMap[rp.RoleID] = append(rolePermissionsMap[rp.RoleID], permission)
	}

	// 7. Build response with roles and their permissions
	roles := make([]openapi.RoleWithPermissions, 0, len(dbRoles))
	for _, dbRole := range dbRoles {
		role := openapi.RoleWithPermissions{
			Id:          dbRole.ID,
			Name:        dbRole.Name,
			WorkspaceId: dbRole.WorkspaceID,
			Description: &dbRole.Description.String,
			Permissions: rolePermissionsMap[dbRole.ID],
		}

		if dbRole.CreatedAtM.Valid {
			createdAt := dbRole.CreatedAtM.Time.Format(http.TimeFormat)
			role.CreatedAt = &createdAt
		}

		roles = append(roles, role)
	}

	return roles, nextCursor, total, nil
}
