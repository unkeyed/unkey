package handler

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2RatelimitUpdateNamespaceRequestBody
	Response = openapi.V2RatelimitUpdateNamespaceResponseBody
)

type Handler struct {
	DB             db.Database
	Auditlogs      auditlogs.AuditLogService
	NamespaceCache cache.Cache[cache.ScopedKey, db.FindRatelimitNamespace]
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/ratelimit.updateNamespace"
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

	var previousName string
	namespace, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (openapi.RatelimitNamespace, error) {
		row, err := db.Query.LockRatelimitNamespaceByIdOrName(ctx, tx, db.LockRatelimitNamespaceByIdOrNameParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Search:      req.Namespace,
		})
		if db.IsNotFound(err) {
			return openapi.RatelimitNamespace{}, apierrors.NotFound(codes.Data.RatelimitNamespace.NotFound.URN())
		}
		if err != nil {
			return openapi.RatelimitNamespace{}, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve namespace."),
			)
		}

		permissionQuery := rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(row.ProjectID).RatelimitNamespace(row.ID),
			permissions.Write,
		)
		err = principal.Authorize(permissionQuery)
		if err != nil {
			return openapi.RatelimitNamespace{}, apierrors.MaskInsufficientPermissionsAsNotFound(
				err,
				principal.Permissions,
				permissionQuery,
				codes.Data.RatelimitNamespace.NotFound.URN(),
			)
		}

		previousName = row.Name
		result := openapi.RatelimitNamespace{
			Id:        row.ID,
			Name:      req.Name,
			CreatedAt: row.CreatedAtM,
			UpdatedAt: row.UpdatedAtM.Int64,
		}
		if req.Name == row.Name {
			return result, nil
		}

		result.UpdatedAt = time.Now().UnixMilli()
		err = db.Query.UpdateRatelimitNamespaceName(ctx, tx, db.UpdateRatelimitNamespaceNameParams{
			Name:        req.Name,
			UpdatedAtM:  sql.NullInt64{Valid: true, Int64: result.UpdatedAt},
			WorkspaceID: principal.AuthorizedWorkspaceID,
			ID:          row.ID,
		})
		if db.IsDuplicateKeyError(err) {
			return openapi.RatelimitNamespace{}, fault.Wrap(
				err,
				fault.Code(codes.Data.RatelimitNamespace.Duplicate.URN()),
				fault.Internal("namespace name already exists"),
				fault.Public(fmt.Sprintf("A namespace named '%s' already exists or was deleted in this workspace. Choose a different name.", req.Name)),
			)
		}
		if err != nil {
			return openapi.RatelimitNamespace{}, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to update namespace."),
			)
		}

		return result, h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{
			{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.RatelimitNamespaceUpdateEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Changed %s name from %s to %s", row.ID, row.Name, req.Name),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: "",
				Targets: []auditlog.AuditLogTarget{
					{
						Type:        auditlog.RatelimitNamespaceTargetType,
						ID:          row.ID,
						Name:        req.Name,
						DisplayName: req.Name,
						Meta: map[string]any{
							"previousName": row.Name,
						},
					},
				},
			},
		})
	})
	if err != nil {
		return err
	}

	// if nothing changed, don't invalidate
	if previousName != namespace.Name {
		h.NamespaceCache.Remove(
			ctx,
			cache.ScopedKey{WorkspaceID: principal.AuthorizedWorkspaceID, Key: namespace.Id},
			cache.ScopedKey{WorkspaceID: principal.AuthorizedWorkspaceID, Key: previousName},
			cache.ScopedKey{WorkspaceID: principal.AuthorizedWorkspaceID, Key: namespace.Name},
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: namespace,
	})
}
