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
	Request  = openapi.V2PermissionsUpdateRoleRequestBody
	Response = openapi.V2PermissionsUpdateRoleResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/permissions.updateRole"
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

	data, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.V2PermissionsUpdateRoleResponseData, error) {
		role, err := db.Query.LockRoleByIdOrName(ctx, tx, db.LockRoleByIdOrNameParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Search:      req.Role,
		})
		if db.IsNotFound(err) {
			return openapi.V2PermissionsUpdateRoleResponseData{}, fault.New("role not found",
				fault.Code(codes.Data.Role.NotFound.URN()),
				fault.Internal("role not found"),
				fault.Public("The requested role does not exist."),
			)
		}
		if err != nil {
			return openapi.V2PermissionsUpdateRoleResponseData{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve role information."),
			)
		}

		err = principal.Authorize(rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(role.ProjectID).RBAC().Role(role.ID),
			permissions.Write,
		))
		if err != nil {
			return openapi.V2PermissionsUpdateRoleResponseData{}, apierrors.MaskInsufficientPermissionsAsNotFound(
				err,
				codes.Data.Role.NotFound.URN(),
				"The requested role does not exist.",
			)
		}

		description := role.Description.String
		switch {
		case req.Description.IsNull():
			description = ""
		case req.Description.IsSpecified():
			description = req.Description.MustGet()
		}

		result := openapi.V2PermissionsUpdateRoleResponseData{
			Id:          role.ID,
			Name:        ptr.SafeDeref(req.Name, role.Name),
			Description: description,
		}

		if req.Name == nil && !req.Description.IsSpecified() {
			return result, nil
		}

		err = db.Query.UpdateRole(ctx, tx, db.UpdateRoleParams{
			Name:        result.Name,
			Description: sql.NullString{Valid: description != "", String: description},
			UpdatedAtM:  sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			WorkspaceID: principal.AuthorizedWorkspaceID,
			ID:          role.ID,
		})
		if db.IsDuplicateKeyError(err) {
			return openapi.V2PermissionsUpdateRoleResponseData{}, fault.Wrap(err,
				fault.Code(codes.Data.Role.Duplicate.URN()),
				fault.Internal("role name already exists"),
				fault.Public(fmt.Sprintf("A role with name '%s' already exists in this workspace.", result.Name)),
			)
		}
		if err != nil {
			return openapi.V2PermissionsUpdateRoleResponseData{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to update role."),
			)
		}

		err = h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.RoleUpdateEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       "Updated " + role.ID,
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Resources: []auditlog.AuditLogResource{
					{
						Type:        auditlog.RoleResourceType,
						ID:          role.ID,
						Name:        result.Name,
						DisplayName: result.Name,
						Meta: map[string]any{
							"name":        result.Name,
							"description": result.Description,
						},
					},
				},
			},
		})
		if err != nil {
			return openapi.V2PermissionsUpdateRoleResponseData{}, err
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
