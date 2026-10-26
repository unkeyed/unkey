package handler

import (
	"context"
	"database/sql"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/internal/services/auditlogs"
	"github.com/unkeyed/unkey/pkg/auditlog"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	connectionapp "github.com/unkeyed/unkey/pkg/deploy/connections/app"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/connections"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ConnectionsUpdateConnectionRequestBody
	Response = openapi.V2ConnectionsUpdateConnectionResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/connections.updateConnection" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	var name string
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		connection, err := connections.LockConnection(ctx, tx, principal, req.ConnectionId)
		if err != nil {
			return err
		}
		if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(connection.ProjectID).App(connection.AppID).Environment(connection.EnvironmentID), permissions.Write)); err != nil {
			return err
		}
		if _, err := connections.RequireTargetApp(ctx, tx, principal, connection.ProjectID, connection.ResourceID); err != nil {
			return err
		}

		name = connection.Name
		if req.Name != nil {
			name = strings.TrimSpace(*req.Name)
		}
		if !connectionapp.IsValidName(name) {
			return fault.New("invalid connection name", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Use a DNS label of at most 63 characters. The name self and names starting with unkey are reserved."))
		}

		selection, err := connections.ValidateTarget(ctx, tx, principal, connection.ProjectID, connection.ResourceID, connections.Target{
			Type:          db.ConnectionAppTargetsSelectionMode(req.TargetType),
			EnvironmentID: req.TargetEnvironmentId,
			DeploymentID:  req.TargetDeploymentId,
		})
		if err != nil {
			return err
		}

		if err := db.Query.UpdateAppConnection(ctx, tx, db.UpdateAppConnectionParams{
			ID:          connection.ID,
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Name:        name,
			UpdatedAt:   sql.NullInt64{Int64: h.Clock.Now().UnixMilli(), Valid: true},
		}); err != nil {
			return err
		}
		if err := db.Query.UpdateConnectionAppTarget(ctx, tx, db.UpdateConnectionAppTargetParams{
			ConnectionID:        connection.ID,
			SelectionMode:       selection.SelectionMode,
			TargetEnvironmentID: selection.TargetEnvironmentID,
			TargetDeploymentID:  selection.TargetDeploymentID,
		}); err != nil {
			return err
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			Event:         auditlog.AppUpdateEvent,
			Display:       "Updated app connection " + name,
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
	if db.IsDuplicateKeyError(err) {
		return fault.New("duplicate connection", fault.Code(codes.Data.Connection.Conflict.URN()), fault.Public("This app is already connected, or the name is already in use."))
	}
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to update connection"))
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2ConnectionsUpdateConnectionResponseData{Id: req.ConnectionId, Name: name},
	})
}
