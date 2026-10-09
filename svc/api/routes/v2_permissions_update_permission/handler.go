package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	dbtype "github.com/unkeyed/unkey/pkg/db/types"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2PermissionsUpdatePermissionRequestBody
	Response = openapi.V2PermissionsUpdatePermissionResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/permissions.updatePermission"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	data, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.Permission, error) {
		permission, err := db.Query.LockPermissionByIdOrSlug(ctx, tx, db.LockPermissionByIdOrSlugParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Search:      req.Permission,
		})
		if db.IsNotFound(err) {
			return openapi.Permission{}, fault.New("permission not found",
				fault.Code(codes.Data.Permission.NotFound.URN()),
				fault.Internal("permission not found"),
				fault.Public("The requested permission does not exist."),
			)
		}
		if err != nil {
			return openapi.Permission{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve permission information."),
			)
		}

		err = principal.Authorize(rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(permission.ProjectID).RBAC().Permission(permission.ID),
			permissions.Write,
		))
		if err != nil {
			return openapi.Permission{}, apierrors.MaskInsufficientPermissionsAsNotFound(
				err,
				codes.Data.Permission.NotFound.URN(),
				"The requested permission does not exist.",
			)
		}

		description := permission.Description.String
		switch {
		case req.Description.IsNull():
			description = ""
		case req.Description.IsSpecified():
			description = req.Description.MustGet()
		}

		result := openapi.Permission{
			Id:          permission.ID,
			Name:        ptr.SafeDeref(req.Name, permission.Name),
			Slug:        ptr.SafeDeref(req.Slug, permission.Slug),
			Description: description,
		}

		if req.Name == nil && req.Slug == nil && !req.Description.IsSpecified() {
			return result, nil
		}

		err = db.Query.UpdatePermission(ctx, tx, db.UpdatePermissionParams{
			Name:        result.Name,
			Slug:        result.Slug,
			Description: dbtype.NullString{Valid: description != "", String: description},
			UpdatedAtM:  sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			WorkspaceID: principal.AuthorizedWorkspaceID,
			ID:          permission.ID,
		})
		if db.IsDuplicateKeyError(err) {
			return openapi.Permission{}, fault.Wrap(err,
				fault.Code(codes.Data.Permission.Duplicate.URN()),
				fault.Internal("permission slug already exists"),
				fault.Public(fmt.Sprintf("A permission with slug '%s' already exists in this workspace.", result.Slug)),
			)
		}
		if err != nil {
			return openapi.Permission{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to update permission."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.PermissionUpdateEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       "Updated " + permission.ID,
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.PermissionResourceType,
						ID:          permission.ID,
						Name:        result.Slug,
						DisplayName: result.Name,
						Meta: map[string]any{
							"name":        result.Name,
							"slug":        result.Slug,
							"description": result.Description,
						},
					},
				},
			},
		})
		if err != nil {
			return openapi.Permission{}, err
		}

		return result, nil
	})
	if err != nil {
		return err
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
