package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"
	"time"

	"github.com/unkeyed/unkey/go/apps/api/openapi"
	"github.com/unkeyed/unkey/go/internal/services/auditlogs"
	"github.com/unkeyed/unkey/go/internal/services/keys"
	"github.com/unkeyed/unkey/go/internal/services/permissions"
	"github.com/unkeyed/unkey/go/pkg/auditlog"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/rbac"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

type Request = openapi.V2KeysSetRolesRequestBody
type Response = openapi.V2KeysSetRolesResponse

type Services struct {
	Logger      logging.Logger
	DB          db.Database
	Keys        keys.KeyService
	Permissions permissions.PermissionService
	Auditlogs   auditlogs.AuditLogService
}

func New(svc Services) zen.Route {
	return zen.NewRoute("POST", "/v2/keys.setRoles", func(ctx context.Context, s *zen.Session) error {
		svc.Logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/keys.setRoles")

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
					ResourceType: rbac.Api,
					ResourceID:   "*",
					Action:       rbac.UpdateKey,
				}),
			),
		)
		if err != nil {
			return err
		}

		// 4. Validate key exists and belongs to workspace
		key, err := db.Query.FindKeyByID(ctx, svc.DB.RO(), req.KeyId)
		if err != nil {
			if db.IsNotFound(err) {
				return fault.New("key not found",
					fault.Code(codes.Data.Key.NotFound.URN()),
					fault.Internal("key not found"), fault.Public("The specified key was not found."),
				)
			}
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve key."),
			)
		}

		// Validate key belongs to authorized workspace
		if key.WorkspaceID != auth.AuthorizedWorkspaceID {
			return fault.New("key not found",
				fault.Code(codes.Data.Key.NotFound.URN()),
				fault.Internal("key belongs to different workspace"), fault.Public("The specified key was not found."),
			)
		}

		// 5. Get current roles for the key
		currentRoles, err := db.Query.ListRolesByKeyID(ctx, svc.DB.RO(), req.KeyId)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve current roles."),
			)
		}

		// 6. Resolve and validate requested roles
		requestedRoles := make([]db.Role, 0, len(req.Roles))
		for _, roleRef := range req.Roles {
			var role db.Role

			if roleRef.Id != nil {
				// Find by ID
				role, err = db.Query.FindRoleByID(ctx, svc.DB.RO(), *roleRef.Id)
				if err != nil {
					if db.IsNotFound(err) {
						return fault.New("role not found",
							fault.Code(codes.Data.Role.NotFound.URN()),
							fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role with ID '%s' was not found.", *roleRef.Id)),
						)
					}
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to retrieve role."),
					)
				}
			} else if roleRef.Name != nil {
				// Find by name
				role, err = db.Query.FindRoleByNameAndWorkspaceID(ctx, svc.DB.RO(), db.FindRoleByNameAndWorkspaceIDParams{
					Name:        *roleRef.Name,
					WorkspaceID: auth.AuthorizedWorkspaceID,
				})
				if err != nil {
					if db.IsNotFound(err) {
						return fault.New("role not found",
							fault.Code(codes.Data.Role.NotFound.URN()),
							fault.Internal("role not found"), fault.Public(fmt.Sprintf("Role with name '%s' was not found.", *roleRef.Name)),
						)
					}
					return fault.Wrap(err,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to retrieve role."),
					)
				}
			} else {
				return fault.New("invalid role reference",
					fault.Code(codes.App.Validation.InvalidInput.URN()),
					fault.Internal("role missing id and name"), fault.Public("Each role must specify either 'id' or 'name'."),
				)
			}

			// Validate role belongs to the same workspace
			if role.WorkspaceID != auth.AuthorizedWorkspaceID {
				return fault.New("role not found",
					fault.Code(codes.Data.Role.NotFound.URN()),
					fault.Internal("role belongs to different workspace"), fault.Public(fmt.Sprintf("Role '%s' was not found.", role.Name)),
				)
			}

			requestedRoles = append(requestedRoles, role)
		}

		// 7. Calculate differential update
		// Create maps for efficient lookup
		currentRoleIDs := make(map[string]bool)
		for _, role := range currentRoles {
			currentRoleIDs[role.ID] = true
		}

		requestedRoleIDs := make(map[string]bool)
		requestedRoleMap := make(map[string]db.Role)
		for _, role := range requestedRoles {
			requestedRoleIDs[role.ID] = true
			requestedRoleMap[role.ID] = role
		}

		// Determine roles to remove and add
		rolesToRemove := make([]string, 0)
		for _, role := range currentRoles {
			if !requestedRoleIDs[role.ID] {
				rolesToRemove = append(rolesToRemove, role.ID)
			}
		}

		rolesToAdd := make([]db.Role, 0)
		for _, role := range requestedRoles {
			if !currentRoleIDs[role.ID] {
				rolesToAdd = append(rolesToAdd, role)
			}
		}

		// 8. Apply changes in transaction
		tx, err := svc.DB.RW().Begin(ctx)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to create transaction"), fault.Public("Unable to start database transaction."),
			)
		}

		defer func() {
			if err := tx.Rollback(); err != nil && err != sql.ErrTxDone {
				svc.Logger.Error("failed to rollback transaction", "error", err)
			}
		}()

		var auditLogs []auditlog.AuditLog

		// Remove roles that are no longer needed
		for _, roleID := range rolesToRemove {
			err := db.Query.DeleteManyKeyRolesByKeyID(ctx, tx, db.DeleteManyKeyRolesByKeyIDParams{
				KeyID:  req.KeyId,
				RoleID: roleID,
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to remove role assignment."),
				)
			}

			// Find the role details for the audit log
			var removedRole db.Role
			for _, role := range currentRoles {
				if role.ID == roleID {
					removedRole = role
					break
				}
			}

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.AuthDisconnectRoleKeyEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				Display:     fmt.Sprintf("Removed role %s from key %s", removedRole.Name, req.KeyId),
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        "key",
						ID:          req.KeyId,
						Name:        key.Name.String,
						DisplayName: key.Name.String,
					},
					{
						Type:        "role",
						ID:          removedRole.ID,
						Name:        removedRole.Name,
						DisplayName: removedRole.Name,
					},
				},
			})
		}

		// Add new roles
		for _, role := range rolesToAdd {
			err := db.Query.InsertKeyRole(ctx, tx, db.InsertKeyRoleParams{
				KeyID:       req.KeyId,
				RoleID:      role.ID,
				WorkspaceID: auth.AuthorizedWorkspaceID,
				CreatedAtM:  time.Now().UnixMilli(),
			})
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to add role assignment."),
				)
			}

			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID: auth.AuthorizedWorkspaceID,
				Event:       auditlog.AuthConnectRoleKeyEvent,
				ActorType:   auditlog.RootKeyActor,
				ActorID:     auth.KeyID,
				ActorName:   "root key",
				Display:     fmt.Sprintf("Added role %s to key %s", role.Name, req.KeyId),
				RemoteIP:    s.Location(),
				UserAgent:   s.UserAgent(),
				Resources: []auditlog.AuditLogResource{
					{
						Type:        "key",
						ID:          req.KeyId,
						Name:        key.Name.String,
						DisplayName: key.Name.String,
					},
					{
						Type:        "role",
						ID:          role.ID,
						Name:        role.Name,
						DisplayName: role.Name,
					},
				},
			})
		}

		// Insert audit logs if there are changes
		if len(auditLogs) > 0 {
			err = svc.Auditlogs.Insert(ctx, tx, auditLogs)
			if err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("audit log error"), fault.Public("Failed to create audit log for role changes."),
				)
			}
		}

		// Commit the transaction
		err = tx.Commit()
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database failed to commit transaction"), fault.Public("Unable to commit database transaction."),
			)
		}

		// 10. Get final state of roles and build response
		finalRoles, err := db.Query.ListRolesByKeyID(ctx, svc.DB.RO(), req.KeyId)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"), fault.Public("Failed to retrieve final role state."),
			)
		}

		// Sort roles alphabetically by name for consistent response
		slices.SortFunc(finalRoles, func(a, b db.Role) int {
			if a.Name < b.Name {
				return -1
			} else if a.Name > b.Name {
				return 1
			}
			return 0
		})

		// Build response data
		responseData := make(openapi.V2KeysSetRolesResponseData, len(finalRoles))
		for i, role := range finalRoles {
			responseData[i] = struct {
				Id   string `json:"id"`
				Name string `json:"name"`
			}{
				Id:   role.ID,
				Name: role.Name,
			}
		}

		// 11. Return success response
		return s.JSON(http.StatusOK, Response{
			Meta: openapi.Meta{
				RequestId: s.RequestID(),
			},
			Data: responseData,
		})
	})
}
