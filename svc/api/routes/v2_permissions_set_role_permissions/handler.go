package handler

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/auth/principal"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Request = openapi.V2PermissionsSetRolePermissionsRequestBody
type Response = openapi.V2PermissionsSetRolePermissionsResponseBody

type rolePermission struct {
	ID          string
	Name        string
	Slug        string
	Description dbtype.NullString
}

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/permissions.setRolePermissions" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	ctx = auditlog.WithCorrelation(ctx, auditlog.NewCorrelationID())

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}
	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	requestedSlugs := make([]string, 0, len(req.Permissions))
	seen := make(map[string]struct{}, len(req.Permissions))
	for _, slug := range req.Permissions {
		normalizedSlug := strings.ToLower(slug)
		if _, ok := seen[normalizedSlug]; ok {
			continue
		}
		seen[normalizedSlug] = struct{}{}
		requestedSlugs = append(requestedSlugs, slug)
	}
	slices.Sort(requestedSlugs)

	result := make([]rolePermission, 0, len(requestedSlugs))
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		role, lockErr := db.Query.LockRoleByIDAndWorkspaceID(ctx, tx, db.LockRoleByIDAndWorkspaceIDParams{
			RoleID: req.RoleId, WorkspaceID: principal.WorkspaceID,
		})
		if lockErr != nil {
			if db.IsNotFound(lockErr) {
				return fault.New("role not found", fault.Code(codes.Data.Role.NotFound.URN()), fault.Internal("role not found"), fault.Public("The requested role does not exist."))
			}
			return fault.Wrap(lockErr, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Internal("unable to lock role"), fault.Public("Failed to retrieve role."))
		}

		authorizeErr := principal.Authorize(rbac.Or(
			rbac.U(
				urn.New().Workspace(principal.WorkspaceID).Project(role.ProjectID).RBAC().Role(role.ID),
				permissions.WriteRole{},
			),
			rbac.And(
				rbac.T(rbac.Tuple{ResourceType: rbac.Rbac, ResourceID: "*", Action: rbac.AddPermissionToRole}),
				rbac.T(rbac.Tuple{ResourceType: rbac.Rbac, ResourceID: "*", Action: rbac.RemovePermissionFromRole}),
			),
		))
		if authorizeErr != nil {
			return authorizeErr
		}

		found := make([]db.FindPermissionsBySlugsForUpdateRow, 0)
		if len(requestedSlugs) > 0 {
			found, err = db.Query.FindPermissionsBySlugsForUpdate(ctx, tx, db.FindPermissionsBySlugsForUpdateParams{
				WorkspaceID: principal.WorkspaceID,
				ProjectID:   role.ProjectID,
				Slugs:       requestedSlugs,
			})
			if err != nil {
				return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Internal("database error"), fault.Public("Failed to lookup permissions to set."))
			}
		}

		bySlug := make(map[string]rolePermission, len(found))
		for _, permission := range found {
			bySlug[strings.ToLower(permission.Slug)] = rolePermission(permission)
		}
		missing := make([]string, 0)
		for _, slug := range requestedSlugs {
			if _, ok := bySlug[strings.ToLower(slug)]; !ok {
				missing = append(missing, slug)
			}
		}

		if len(missing) > 0 {
			if authErr := principal.Authorize(rbac.Or(
				rbac.U(
					urn.New().Workspace(principal.WorkspaceID).Project(role.ProjectID).RBAC().Permission("*"),
					permissions.WritePermission{},
				),
				rbac.T(rbac.Tuple{ResourceType: rbac.Rbac, ResourceID: "*", Action: rbac.CreatePermission}),
			)); authErr != nil {
				return authErr
			}
			candidates := make(map[string]db.UpsertPermissionParams, len(missing))
			for _, slug := range missing {
				now := time.Now().UnixMilli()
				candidate := db.UpsertPermissionParams{
					PermissionID: uid.New(uid.PermissionPrefix),
					WorkspaceID:  principal.WorkspaceID,
					ProjectID:    role.ProjectID,
					Name:         slug,
					Slug:         slug,
					Description:  dbtype.NullString{String: "", Valid: false},
					CreatedAtM:   now,
				}
				candidates[strings.ToLower(slug)] = candidate
				if err = db.Query.UpsertPermission(ctx, tx, candidate); err != nil {
					return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Internal("database error"), fault.Public("Failed to create permissions."))
				}
			}

			found, err = db.Query.FindPermissionsBySlugsForUpdate(ctx, tx, db.FindPermissionsBySlugsForUpdateParams{
				WorkspaceID: principal.WorkspaceID,
				ProjectID:   role.ProjectID,
				Slugs:       requestedSlugs,
			})
			if err != nil {
				return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Internal("database error"), fault.Public("Failed to lookup created permissions."))
			}
			bySlug = make(map[string]rolePermission, len(found))
			logs := make([]auditlog.AuditLog, 0, len(candidates))
			for _, permission := range found {
				normalizedSlug := strings.ToLower(permission.Slug)
				bySlug[normalizedSlug] = rolePermission(permission)
				candidate, ok := candidates[normalizedSlug]
				if ok && candidate.PermissionID == permission.ID {
					logs = append(logs, audit(principal, s, auditlog.PermissionCreateEvent, fmt.Sprintf("Created %s (%s)", permission.Slug, permission.ID), permissionResource(permission.ID, permission.Slug, permission.Name)))
				}
			}
			if len(logs) > 0 {
				err = h.Auditlogs.Insert(ctx, tx, logs)
			}
			if err != nil {
				return err
			}
		}
		for _, slug := range requestedSlugs {
			if _, ok := bySlug[strings.ToLower(slug)]; !ok {
				return fault.New("permission not found",
					fault.Code(codes.Data.Permission.NotFound.URN()),
					fault.Internal("permission belongs to a different project"),
					fault.Public(fmt.Sprintf("Permission '%s' was not found.", slug)),
				)
			}
		}

		result = result[:0]
		requestedIDs := make(map[string]struct{}, len(requestedSlugs))
		for _, slug := range requestedSlugs {
			permission := bySlug[strings.ToLower(slug)]
			result = append(result, permission)
			requestedIDs[permission.ID] = struct{}{}
		}
		current, listErr := db.Query.ListDirectPermissionsByRoleID(ctx, tx, req.RoleId)
		if listErr != nil {
			return listErr
		}
		currentByID := make(map[string]rolePermission, len(current))
		remove := make([]string, 0)
		for _, permission := range current {
			currentByID[permission.ID] = rolePermission(permission)
			if _, ok := requestedIDs[permission.ID]; !ok {
				remove = append(remove, permission.ID)
			}
		}

		logs := make([]auditlog.AuditLog, 0)
		if len(remove) > 0 {
			if err = db.Query.DeleteManyRolePermissionsByRoleAndPermissionIDs(ctx, tx, db.DeleteManyRolePermissionsByRoleAndPermissionIDsParams{RoleID: req.RoleId, PermissionIds: remove}); err != nil {
				return err
			}
			for _, id := range remove {
				p := currentByID[id]
				logs = append(logs, audit(principal, s, auditlog.AuthDisconnectRolePermissionEvent, fmt.Sprintf("Removed permission %s from role %s", p.Name, role.Name), roleResource(role.ID, role.Name), permissionResource(p.ID, p.Slug, p.Name)))
			}
		}
		toAdd := make([]db.InsertRolePermissionParams, 0)
		for _, permission := range result {
			if _, ok := currentByID[permission.ID]; ok {
				continue
			}
			toAdd = append(toAdd, db.InsertRolePermissionParams{RoleID: req.RoleId, PermissionID: permission.ID, WorkspaceID: principal.WorkspaceID, CreatedAtM: time.Now().UnixMilli()})
			logs = append(logs, audit(principal, s, auditlog.AuthConnectRolePermissionEvent, fmt.Sprintf("Added permission %s to role %s", permission.Name, role.Name), roleResource(role.ID, role.Name), permissionResource(permission.ID, permission.Slug, permission.Name)))
		}
		if err = db.BulkQuery.InsertRolePermissions(ctx, tx, toAdd); err != nil {
			return err
		}
		if len(logs) > 0 {
			return h.Auditlogs.Insert(ctx, tx, logs)
		}
		return nil
	})
	if err != nil {
		return err
	}

	data := make(openapi.V2PermissionsSetRolePermissionsResponseData, 0, len(result))
	for _, permission := range result {
		data = append(data, openapi.Permission{Id: permission.ID, Name: permission.Name, Slug: permission.Slug, Description: permission.Description.String})
	}
	return s.JSON(http.StatusOK, Response{Meta: openapi.Meta{RequestId: s.RequestID()}, Data: data})
}

func roleResource(id, name string) auditlog.AuditLogResource {
	return auditlog.AuditLogResource{Type: auditlog.RoleResourceType, ID: id, Name: name, DisplayName: name, Meta: map[string]any{}}
}
func permissionResource(id, slug, name string) auditlog.AuditLogResource {
	return auditlog.AuditLogResource{Type: auditlog.PermissionResourceType, ID: id, Name: slug, DisplayName: name, Meta: map[string]any{}}
}

func audit(principal *principal.Principal, s *zen.Session, event auditlog.AuditLogEvent, display string, resources ...auditlog.AuditLogResource) auditlog.AuditLog {
	return auditlog.AuditLog{WorkspaceID: principal.WorkspaceID, Event: event, ActorType: auditlog.AuditLogActor(principal.Subject.Type), ActorID: principal.Subject.ID, ActorName: principal.Subject.Name, ActorMeta: map[string]any{}, Display: display, RemoteIP: s.Location(), UserAgent: s.UserAgent(), CorrelationID: "", Resources: resources}
}
