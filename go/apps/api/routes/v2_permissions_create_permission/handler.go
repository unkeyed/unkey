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
	"github.com/unkeyed/unkey/go/pkg/id"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2PermissionsCreatePermissionRequestBody
type Response = openapi.V2PermissionsCreatePermissionResponseBody

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/permissions.createPermission", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.createPermission")

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
					Action:       rbac.CreatePermission,
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

		// 4. Create permission
		permissionID := id.NewPermission()

		// Check for existing permission with the same name in this workspace
		existingPerm, err := db.Query.FindPermissionByNameAndWorkspace(ctx, svc.DB.RO(), req.Name, auth.AuthorizedWorkspaceID)
		if err == nil && existingPerm != nil {
			// Permission with this name already exists
			return fault.New("permission already exists",
				fault.WithCode(codes.Data.Permission.AlreadyExists.URN()),
				fault.WithDesc("permission already exists",
					"A permission with name \""+req.Name+"\" already exists in this workspace"),
			)
		}

		var description string
		if req.Description != nil {
			description = *req.Description
		}

		// Create permission in a transaction with audit log
		err = svc.DB.WithTransaction(ctx, func(ctx context.Context, tx db.Tx) error {
			// Insert the permission
			_, err := db.Query.InsertPermission(ctx, tx, db.InsertPermissionParams{
				ID:          permissionID,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Name:        req.Name,
				Description: db.NewNullString(description),
			})
			if err != nil {
				if db.IsDuplicate(err) {
					return fault.New("permission already exists",
						fault.WithCode(codes.Data.Permission.AlreadyExists.URN()),
						fault.WithDesc("permission already exists",
							"A permission with name \""+req.Name+"\" already exists in this workspace"),
					)
				}
				return fault.Wrap(err,
					fault.WithCode(codes.App.Internal.ServiceUnavailable.URN()),
					fault.WithDesc("database error", "Failed to create permission."),
				)
			}

			// Create audit log
			err = svc.Auditlogs.CreateAuditLog(ctx, tx, auditlogs.CreateAuditLogParams{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       "permission.create",
				ActorType:   "key",
				ActorID:     auth.KeyID,
				Description: "Created " + permissionID,
				Resources: []auditlogs.Resource{
					{
						Type: "permission",
						ID:   permissionID,
						Meta: map[string]interface{}{
							"name":        req.Name,
							"description": description,
						},
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
					fault.WithDesc("audit log error", "Failed to create audit log for permission creation."),
				)
			}

			return nil
		})
		if err != nil {
			return err
		}

		// 5. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: openapi.PermissionsCreatePermissionResponseData{
				PermissionId: permissionID,
			},
		})
	})
}
