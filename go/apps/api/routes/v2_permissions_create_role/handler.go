package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsCreateRoleRequestBody
type Response = openapi.V2PermissionsCreateRoleResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.createRole", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.createRole")

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

		// 3. Permission check
		permissionCheck, err := svc.Permissions.Check(
			ctx,
			auth.KeyID,
			rbac.Or(
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Rbac,
					ResourceID:   "*",
					Action:       rbac.CreateRole,
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

		// 4. Prepare role creation
		roleID := uid.New(uid.RolePrefix)
		var description string
		if req.Description != nil {
			description = *req.Description
		}

		// Check for existing role with the same name in this workspace
		existingRole, err := db.Query.FindRoleByNameAndWorkspace(ctx, svc.DB.RO(), req.Name, auth.AuthorizedWorkspaceID)
		if err == nil && existingRole != nil {
			// Role with this name already exists
			return fault.New("role already exists",
				fault.WithCode(codes.Data.Role.AlreadyExists.URN()),
				fault.WithDesc("role already exists",
					"A role with name \""+req.Name+"\" already exists in this workspace"),
			)
		}

		// 5. Validate permission IDs if provided
		var permissionIDs []string
		if req.PermissionIds != nil && len(*req.PermissionIds) > 0 {
			permissionIDs = *req.PermissionIds

			// Verify all permissions exist and belong to the workspace
			for _, permID := range permissionIDs {
				permission, err := db.Query.FindPermissionById(ctx, svc.DB.RO(), permID)
				if err != nil {
					if db.IsNotFound(err) {
						return fault.New("permission not found",
							fault.WithCode(codes.Data.Permission.NotFound.URN()),
							fault.WithDesc("permission not found", "Permission with ID \""+permID+"\" does not exist."),
						)
					}
					return fault.Wrap(err,
						fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
						fault.WithDesc("database error", "Failed to retrieve permission information."),
					)
				}

				// Check if permission belongs to authorized workspace
				if permission.WorkspaceID != auth.AuthorizedWorkspaceID {
					return fault.New("permission not found",
						fault.WithCode(codes.Data.Permission.NotFound.URN()),
						fault.WithDesc("permission not found", "Permission with ID \""+permID+"\" does not exist."),
					)
				}
			}
		}

		// 6. Create role in a transaction with permission assignments and audit log
		err = svc.DB.WithTransaction(ctx, func(ctx context.Context, tx db.Tx) error {
			// Insert the role
			_, err := db.Query.InsertRole(ctx, tx, db.InsertRoleParams{
				ID:          roleID,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Name:        req.Name,
				Description: db.NewNullString(description),
			})
			if err != nil {
				if db.IsDuplicate(err) {
					return fault.New("role already exists",
						fault.WithCode(codes.Data.Role.AlreadyExists.URN()),
						fault.WithDesc("role already exists",
							"A role with name \""+req.Name+"\" already exists in this workspace"),
					)
				}
				return fault.Wrap(err,
					fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
					fault.WithDesc("database error", "Failed to create role."),
				)
			}

			// Create role-permission relationships if permissions were provided
			for _, permID := range permissionIDs {
				_, err := db.Query.InsertRolePermission(ctx, tx, db.InsertRolePermissionParams{
					RoleID:       roleID,
					PermissionID: permID,
				})
				if err != nil {
					return fault.Wrap(err,
						fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
						fault.WithDesc("database error", "Failed to assign permissions to role."),
					)
				}
			}

			// Create audit log
			metaData := map[string]interface{}{
				"name":        req.Name,
				"description": description,
			}

			if len(permissionIDs) > 0 {
				metaData["permissions"] = permissionIDs
			}

			err = svc.Auditlogs.CreateAuditLog(ctx, tx, auditlogs.CreateAuditLogParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       "role.create",
				ActorType:   "key",
				ActorID:     auth.KeyID,
				Description: "Created " + roleID,
				Resources: []auditlogs.Resource{
					{
						Type: "role",
						ID:   roleID,
						Meta: metaData,
					},
				},
				Context: auditlogs.Context{
					Location:  s.Location(),
					UserAgent: s.UserAgent(),
				},
			})
			if err != nil {
				return fault.Wrap(err,
					fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
					fault.WithDesc("audit log error", "Failed to create audit log for role creation."),
				)
			}

			return nil
		})
		if err != nil {
			return err
		}

		// 7. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.PermissionsCreateRoleResponseData{
				RoleId: roleID,
			},
		})
	})
}
