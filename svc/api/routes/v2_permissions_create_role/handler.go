package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"slices"
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
	rbacpermissions "github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/projects"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2PermissionsCreateRoleRequestBody
type Response = openapi.V2PermissionsCreateRoleResponseBody

type rolePermission struct {
	ID          string
	Name        string
	Slug        string
	Description dbtype.NullString
}

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

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
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
	slices.Sort(permissionSlugs)

	roleID := uid.New(uid.RolePrefix)
	description := ptr.SafeDeref(req.Description)

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		projectID, resolveErr := projects.EnsureDefaultProject(ctx, tx, principal.WorkspaceID)
		if resolveErr != nil {
			return resolveErr
		}

		legacyAuthorization := rbac.T(rbac.Tuple{
			ResourceType: rbac.Rbac,
			ResourceID:   "*",
			Action:       rbac.CreateRole,
		})
		if len(permissionSlugs) > 0 {
			legacyAuthorization = rbac.And(
				legacyAuthorization,
				rbac.T(rbac.Tuple{
					ResourceType: rbac.Rbac,
					ResourceID:   "*",
					Action:       rbac.AddPermissionToRole,
				}),
			)
		}
		if authorizeErr := principal.Authorize(rbac.Or(
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Project(projectID).RBAC().Role("*"),
				rbacpermissions.Write{},
			),
			legacyAuthorization,
		)); authorizeErr != nil {
			return authorizeErr
		}

		permissions := make([]rolePermission, 0, len(permissionSlugs))
		createdPermissions := make([]rolePermission, 0)
		var missingSlugs []string

		if len(permissionSlugs) > 0 {
			foundPermissions, findErr := db.Query.FindPermissionsBySlugsForUpdate(ctx, tx, db.FindPermissionsBySlugsForUpdateParams{
				WorkspaceID: principal.WorkspaceID,
				ProjectID:   projectID,
				Slugs:       permissionSlugs,
			})
			if findErr != nil {
				return fault.Wrap(findErr,
					fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
					fault.Internal("database error"), fault.Public("Failed to lookup permissions."),
				)
			}

			permissionsBySlug := make(map[string]rolePermission, len(foundPermissions))
			for _, permission := range foundPermissions {
				permissionsBySlug[strings.ToLower(permission.Slug)] = rolePermission(permission)
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
				if authorizeErr := principal.Authorize(rbac.Or(
					rbac.U(
						urn.New().Workspace(principal.WorkspaceID).Project(projectID).RBAC().Permission("*"),
						rbacpermissions.Write{},
					),
					rbac.T(rbac.Tuple{
						ResourceType: rbac.Rbac,
						ResourceID:   "*",
						Action:       rbac.CreatePermission,
					}),
				)); authorizeErr != nil {
					return authorizeErr
				}

				candidates := make(map[string]db.UpsertPermissionParams, len(missingSlugs))
				for _, slug := range missingSlugs {
					candidate := db.UpsertPermissionParams{
						PermissionID: uid.New(uid.PermissionPrefix),
						WorkspaceID:  principal.WorkspaceID,
						ProjectID:    projectID,
						Name:         slug,
						Slug:         slug,
						Description:  dbtype.NullString{String: "", Valid: false},
						CreatedAtM:   time.Now().UnixMilli(),
					}
					candidates[strings.ToLower(slug)] = candidate
					if upsertErr := db.Query.UpsertPermission(ctx, tx, candidate); upsertErr != nil {
						return fault.Wrap(upsertErr,
							fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
							fault.Internal("database error"), fault.Public("Failed to create permissions."),
						)
					}
				}

				foundPermissions, findErr = db.Query.FindPermissionsBySlugsForUpdate(ctx, tx, db.FindPermissionsBySlugsForUpdateParams{
					WorkspaceID: principal.WorkspaceID,
					ProjectID:   projectID,
					Slugs:       permissionSlugs,
				})
				if findErr != nil {
					return fault.Wrap(findErr,
						fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
						fault.Internal("database error"), fault.Public("Failed to lookup created permissions."),
					)
				}

				permissionsBySlug = make(map[string]rolePermission, len(foundPermissions))
				for _, permission := range foundPermissions {
					normalizedSlug := strings.ToLower(permission.Slug)
					canonicalPermission := rolePermission(permission)
					permissionsBySlug[normalizedSlug] = canonicalPermission
					candidate, ok := candidates[normalizedSlug]
					if ok && candidate.PermissionID == permission.ID {
						createdPermissions = append(createdPermissions, canonicalPermission)
					}
				}
			}

			permissions = permissions[:0]
			for _, slug := range permissionSlugs {
				permission, ok := permissionsBySlug[strings.ToLower(slug)]
				if !ok {
					return fault.New("permission not found",
						fault.Code(codes.Data.Permission.NotFound.URN()),
						fault.Internal("permission belongs to a different project"),
						fault.Public(fmt.Sprintf("Permission '%s' was not found.", slug)),
					)
				}
				permissions = append(permissions, permission)
			}
		}
		now := time.Now().UnixMilli()

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

		for _, permission := range createdPermissions {
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID:   principal.WorkspaceID,
				Event:         auditlog.PermissionCreateEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Created %s (%s)", permission.Slug, permission.ID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.PermissionResourceType,
						ID:          permission.ID,
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
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.V2PermissionsCreateRoleResponseData{
			RoleId: roleID,
		},
	})
}
