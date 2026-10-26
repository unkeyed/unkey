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
	Request  = openapi.V2RatelimitDeleteNamespaceRequestBody
	Response = openapi.V2RatelimitDeleteNamespaceResponseBody
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
	return "/v2/ratelimit.deleteNamespace"
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

	namespace, err := db.TxWithResultRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) (db.LockRatelimitNamespaceByIdOrNameRow, error) {
		row, err := db.Query.LockRatelimitNamespaceByIdOrName(ctx, tx, db.LockRatelimitNamespaceByIdOrNameParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Search:      req.Namespace,
		})
		if db.IsNotFound(err) {
			return row, fault.New(
				"namespace not found",
				fault.Code(codes.Data.RatelimitNamespace.NotFound.URN()),
				fault.Internal("namespace not found"),
				fault.Public("The requested namespace does not exist."),
			)
		}
		if err != nil {
			return row, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve namespace."),
			)
		}

		err = principal.Authorize(rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(row.ProjectID).RatelimitNamespace(row.ID),
			permissions.Delete,
		))
		if err != nil {
			return row, apierrors.MaskInsufficientPermissionsAsNotFound(
				err,
				codes.Data.RatelimitNamespace.NotFound.URN(),
			)
		}

		overrideIDs, err := db.Query.ListRatelimitOverrideIDsByNamespaceID(ctx, tx, db.ListRatelimitOverrideIDsByNamespaceIDParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			NamespaceID: row.ID,
		})
		if err != nil {
			return row, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve namespace overrides."),
			)
		}

		now := sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()}
		err = db.Query.SoftDeleteRatelimitNamespace(ctx, tx, db.SoftDeleteRatelimitNamespaceParams{
			Now: now,
			ID:  row.ID,
		})
		if err != nil {
			return row, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to delete namespace."),
			)
		}

		err = db.Query.SoftDeleteRatelimitOverridesByNamespaceID(ctx, tx, db.SoftDeleteRatelimitOverridesByNamespaceIDParams{
			Now:         now,
			WorkspaceID: principal.AuthorizedWorkspaceID,
			NamespaceID: row.ID,
		})
		if err != nil {
			return row, fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to delete namespace overrides."),
			)
		}

		namespaceTarget := auditlog.AuditLogTarget{
			Type:        auditlog.RatelimitNamespaceTargetType,
			ID:          row.ID,
			Name:        row.Name,
			DisplayName: row.Name,
			Meta:        nil,
		}
		correlationID := auditlog.NewCorrelationID()
		auditLogs := make([]auditlog.AuditLog, 0, len(overrideIDs)+1)
		auditLogs = append(auditLogs, auditlog.AuditLog{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			Event:         auditlog.RatelimitNamespaceDeleteEvent,
			ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
			ActorID:       principal.Subject.ID,
			ActorName:     principal.Subject.Name,
			ActorMeta:     map[string]any{},
			Display:       "Deleted " + row.ID,
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: correlationID,
			Targets:       []auditlog.AuditLogTarget{namespaceTarget},
		})
		for _, overrideID := range overrideIDs {
			auditLogs = append(auditLogs, auditlog.AuditLog{
				WorkspaceID:   principal.AuthorizedWorkspaceID,
				Event:         auditlog.RatelimitDeleteOverrideEvent,
				ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
				ActorID:       principal.Subject.ID,
				ActorName:     principal.Subject.Name,
				ActorMeta:     map[string]any{},
				Display:       fmt.Sprintf("Deleted %s as part of the %s deletion", overrideID, row.ID),
				RemoteIP:      s.Location(),
				UserAgent:     s.UserAgent(),
				CorrelationID: correlationID,
				Targets: []auditlog.AuditLogTarget{
					namespaceTarget,
					{
						Type:        auditlog.RatelimitOverrideTargetType,
						ID:          overrideID,
						Name:        "",
						DisplayName: "",
						Meta:        nil,
					},
				},
			})
		}

		return row, h.Auditlogs.Insert(ctx, tx, auditLogs)
	})
	if err != nil {
		return err
	}

	h.NamespaceCache.Remove(ctx,
		cache.ScopedKey{WorkspaceID: principal.AuthorizedWorkspaceID, Key: namespace.ID},
		cache.ScopedKey{WorkspaceID: principal.AuthorizedWorkspaceID, Key: namespace.Name},
	)

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.EmptyResponse{},
	})
}
