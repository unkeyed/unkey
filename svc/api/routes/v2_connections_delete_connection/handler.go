package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/connections"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ConnectionsDeleteConnectionRequestBody
	Response = openapi.V2ConnectionsDeleteConnectionResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/connections.deleteConnection" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		connection, err := connections.LockConnection(ctx, tx, principal, req.ConnectionId)
		if err != nil {
			return err
		}
		if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(connection.ProjectID).App(connection.AppID).Environment(connection.EnvironmentID), permissions.Write)); err != nil {
			return err
		}

		if err := db.Query.DeleteAppConnection(ctx, tx, db.DeleteAppConnectionParams{
			ID:          connection.ID,
			WorkspaceID: principal.AuthorizedWorkspaceID,
		}); err != nil {
			return err
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			Event:         auditlog.AppUpdateEvent,
			Display:       "Deleted app connection " + connection.Name,
			ActorID:       principal.Subject.ID,
			ActorName:     principal.Subject.Name,
			ActorMeta:     nil,
			ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Targets: []auditlog.AuditLogTarget{{
				ID:          connection.AppID,
				Type:        auditlog.AppTargetType,
				Name:        "",
				DisplayName: "",
				Meta:        nil,
			}},
		}})
	})
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to delete connection"))
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2ConnectionsDeleteConnectionResponseData{Id: req.ConnectionId},
	})
}
