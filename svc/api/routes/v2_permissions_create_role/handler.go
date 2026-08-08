package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2PermissionsCreateRoleRequestBody
type Response = openapi.V2PermissionsCreateRoleResponseBody

// Handler implements zen.Route interface for the v2 permissions create role endpoint
type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

// Method returns the HTTP method this route responds to
func (h *Handler) Method() string {
	return "POST"
}

// Path returns the URL path pattern this route matches
func (h *Handler) Path() string {
	return "/v2/permissions.createRole"
}

// Handle processes the HTTP request
func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	logger.Debug("handling request", "requestId", s.RequestID(), "path", "/v2/permissions.createRole")

	// 1. Authentication
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	// 2. Request validation
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	// 3. Permission check
	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.CreateRole,
		}),
	))
	if err != nil {
		return err
	}

	permissionSlugs := make([]string, 0)
	if req.Permissions != nil {
		seen := make(map[string]struct{}, len(*req.Permissions))
		for _, slug := range *req.Permissions {
			normalizedSlug := strings.ToLower(slug)
			if _, ok := seen[normalizedSlug]; ok {
				continue
			}
			seen[normalizedSlug] = struct{}{}
			permissionSlugs = append(permissionSlugs, slug)
		}
	}

	if len(permissionSlugs) > 0 {
		err = principal.Authorize(rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.AddPermissionToRole,
		}))
		if err != nil {
			return err
		}
	}

	// 4. Prepare role creation
	roleID := uid.New(uid.RolePrefix)
	description := ptr.SafeDeref(req.Description)

	// 5. Create role and permissions in a transaction with audit logs
	for attempt := 0; attempt < db.DefaultAttempts; attempt++ {
		permissionInsertConflict := false
		err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
			permissions := make([]db.Permission, 0, len(permissionSlugs))
			permissionsToInsert := make([]db.InsertPermissionParams, 0)
			missingSlugs := make([]string, 0)

			if len(permissionSlugs) > 0 {
				foundPermissions, findErr := db.Query.FindPermissionsBySlugs(ctx, tx, db.FindPermissionsBySlugsParams{
					WorkspaceID: principal.WorkspaceID,
					Slugs:       permissionSlugs,
				})
				if findErr != nil {
					return fault.Wrap(findErr,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to lookup permissions."),
					)
				}

				permissionsBySlug := make(map[string]db.Permission, len(foundPermissions))
				for _, permission := range foundPermissions {
					permissionsBySlug[strings.ToLower(permission.Slug)] = permission
				}

				missingSlugs = make([]string, 0, len(permissionSlugs)-len(foundPermissions))
				for _, slug := range permissionSlugs {
					if permission, ok := permissionsBySlug[strings.ToLower(slug)]; ok {
						permissions = append(permissions, permission)
						continue
					}
					missingSlugs = append(missingSlugs, slug)
				}

				if len(missingSlugs) > 0 {
					if authorizeErr := principal.Authorize(rbac.T(rbac.Tuple{
						ResourceType: rbac.Rbac,
						ResourceID:   "*",
						Action:       rbac.CreatePermission,
					})); authorizeErr != nil {
						return authorizeErr
					}
				}
			}

			projectID, resolveErr := projects.EnsureDefaultProject(ctx, tx, principal.WorkspaceID)
			if resolveErr != nil {
				return resolveErr
			}

			now := time.Now().UnixMilli()
			for _, slug := range missingSlugs {
				permissionID := uid.New(uid.PermissionPrefix)
				permissionsToInsert = append(permissionsToInsert, db.InsertPermissionParams{
					PermissionID: permissionID,
					WorkspaceID:  principal.WorkspaceID,
					ProjectID:    projectID,
					Name:         slug,
					Slug:         slug,
					Description:  dbtype.NullString{String: "", Valid: false},
					CreatedAtM:   now,
				})
				permissions = append(permissions, db.Permission{
					Pk:          0,
					ID:          permissionID,
					WorkspaceID: principal.WorkspaceID,
					ProjectID:   projectID,
					Name:        slug,
					Slug:        slug,
					Description: dbtype.NullString{String: "", Valid: false},
					CreatedAtM:  now,
					UpdatedAtM:  sql.NullInt64{Int64: now, Valid: true},
				})
			}

			// Insert the role
			err = db.Query.InsertRole(ctx, tx, db.InsertRoleParams{
				RoleID:      roleID,
				WorkspaceID: principal.WorkspaceID,
				ProjectID:   projectID,
				Name:        req.Name,
				Description: sql.NullString{Valid: description != "", String: description},
				CreatedAt:   time.Now().UnixMilli(),
			})
			if err != nil {
				if db.IsDuplicateKeyError(err) {
					return fault.New("role already exists",
						fault.Code(codes.Data.Role.Duplicate.URN()),
						fault.Internal("role already exists"), fault.Public(fmt.Sprintf("A role with name '%s' already exists in this workspace", req.Name)),
					)
				}
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to create role."),
				)
			}

			if err = db.BulkQuery.InsertPermissions(ctx, tx, permissionsToInsert); err != nil {
				permissionInsertConflict = db.IsDuplicateKeyError(err)
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to create permissions."),
				)
			}

			rolePermissions := make([]db.InsertRolePermissionParams, len(permissions))
			for i, permission := range permissions {
				rolePermissions[i] = db.InsertRolePermissionParams{
					RoleID:       roleID,
					PermissionID: permission.ID,
					WorkspaceID:  principal.WorkspaceID,
					CreatedAtM:   now,
				}
			}
			if err = db.BulkQuery.InsertRolePermissions(ctx, tx, rolePermissions); err != nil {
				return fault.Wrap(err,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to attach permissions to role."),
				)
			}

			// Create audit logs
			metaData := map[string]interface{}{
				"name":        req.Name,
				"description": description,
			}

			auditLogs := []auditlog.AuditLog{
				{
					WorkspaceID:   principal.WorkspaceID,
					Event:         auditlog.RoleCreateEvent,
					ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
					ActorID:       principal.Subject.ID,
					ActorName:     principal.Subject.Name,
					ActorMeta:     map[string]any{},
					Display:       "Created " + roleID,
					RemoteIP:      s.Location(),
					UserAgent:     s.UserAgent(),
					CorrelationID: "",
					Resources: []auditlog.AuditLogResource{
						{
							Type:        auditlog.RoleResourceType,
							ID:          roleID,
							Name:        req.Name,
							DisplayName: req.Name,
							Meta:        metaData,
						},
					},
				},
			}

			for _, permission := range permissionsToInsert {
				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID:   principal.WorkspaceID,
					Event:         auditlog.PermissionCreateEvent,
					ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
					ActorID:       principal.Subject.ID,
					ActorName:     principal.Subject.Name,
					ActorMeta:     map[string]any{},
					Display:       fmt.Sprintf("Created %s (%s)", permission.Slug, permission.PermissionID),
					RemoteIP:      s.Location(),
					UserAgent:     s.UserAgent(),
					CorrelationID: "",
					Resources: []auditlog.AuditLogResource{
						{
							Type:        auditlog.PermissionResourceType,
							ID:          permission.PermissionID,
							Name:        permission.Slug,
							DisplayName: permission.Name,
							Meta: map[string]any{
								"name": permission.Name,
								"slug": permission.Slug,
							},
						},
					},
				})
			}

			for _, permission := range permissions {
				auditLogs = append(auditLogs, auditlog.AuditLog{
					WorkspaceID:   principal.WorkspaceID,
					Event:         auditlog.AuthConnectRolePermissionEvent,
					ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
					ActorID:       principal.Subject.ID,
					ActorName:     principal.Subject.Name,
					ActorMeta:     map[string]any{},
					Display:       fmt.Sprintf("Added permission %s to role %s", permission.Name, req.Name),
					RemoteIP:      s.Location(),
					UserAgent:     s.UserAgent(),
					CorrelationID: "",
					Resources: []auditlog.AuditLogResource{
						{
							Type:        auditlog.RoleResourceType,
							ID:          roleID,
							Name:        req.Name,
							DisplayName: req.Name,
							Meta:        metaData,
						},
						{
							Type:        auditlog.PermissionResourceType,
							ID:          permission.ID,
							Name:        permission.Slug,
							DisplayName: permission.Name,
							Meta:        map[string]any{},
						},
					},
				})
			}

			err = h.Auditlogs.Insert(ctx, tx, auditLogs)
			if err != nil {
				return err
			}

			return nil
		})
		if !permissionInsertConflict || !db.IsDuplicateKeyError(err) {
			break
		}
	}
	if err != nil {
		return err
	}

	// 7. Return success response
	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2PermissionsCreateRoleResponseData{
			RoleId: roleID,
		},
	})
}
