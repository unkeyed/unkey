package handler

import (
	"context"
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
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/connections"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ConnectionsCreateConnectionRequestBody
	Response = openapi.V2ConnectionsCreateConnectionResponseBody
)

type Handler struct {
	DB        db.Database
	Auditlogs auditlogs.AuditLogService
	Clock     clock.Clock
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/connections.createConnection" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	id := uid.New(uid.ConnectionPrefix)
	var name string
	err = db.TxRetry(ctx, h.DB.RW(), func(ctx context.Context, tx db.DBTX) error {
		environment, err := connections.RequireEnvironment(ctx, tx, principal, req.Project, req.App, req.Environment)
		if err != nil {
			return err
		}
		if err := principal.Authorize(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(environment.ProjectID).App(environment.AppID).Environment(environment.ID), permissions.Write)); err != nil {
			return err
		}

		target, err := connections.RequireTargetApp(ctx, tx, principal, environment.ProjectID, req.TargetAppId)
		if err != nil {
			return err
		}
		if target.ID == environment.AppID {
			return fault.New("self connection", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Choose another app. Connections within this app are automatic."))
		}

		if req.Name == nil {
			name = strings.ToLower(target.Slug)
			if !connectionapp.IsValidName(name) {
				return fault.New("invalid default connection name", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("The target app slug is not a valid connection name. Provide a name."))
			}
		} else {
			name = *req.Name
			if !connectionapp.IsValidName(name) {
				return fault.New("invalid connection name", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Use a DNS label of at most 63 characters. The name self and names starting with unkey are reserved."))
			}
		}

		environmentID, deploymentID, err := connections.LockTarget(ctx, tx, principal, target, db.ConnectionAppTargetsSelectionMode(req.TargetType), req.TargetEnvironmentId, req.TargetDeploymentId)
		if err != nil {
			return err
		}

		if err := db.Query.InsertAppConnection(ctx, tx, db.InsertAppConnectionParams{
			ID:            id,
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			ProjectID:     environment.ProjectID,
			AppID:         environment.AppID,
			EnvironmentID: environment.ID,
			ResourceType:  db.AppConnectionsResourceTypeApp,
			ResourceID:    target.ID,
			Name:          name,
			CreatedAt:     h.Clock.Now().UnixMilli(),
		}); err != nil {
			return err
		}
		if err := db.Query.InsertConnectionAppTarget(ctx, tx, db.InsertConnectionAppTargetParams{
			ConnectionID:        id,
			SelectionMode:       db.ConnectionAppTargetsSelectionMode(req.TargetType),
			TargetEnvironmentID: environmentID,
			TargetDeploymentID:  deploymentID,
		}); err != nil {
			return err
		}

		return h.Auditlogs.Insert(ctx, tx, []auditlog.AuditLog{{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			Event:         auditlog.AppUpdateEvent,
			Display:       "Created app connection " + name,
			ActorID:       principal.Subject.ID,
			ActorName:     principal.Subject.Name,
			ActorMeta:     map[string]any{},
			ActorType:     auditlog.AuditLogActor(principal.Subject.Type),
			RemoteIP:      s.Location(),
			UserAgent:     s.UserAgent(),
			CorrelationID: "",
			Targets: []auditlog.AuditLogTarget{{
				ID:          environment.AppID,
				Type:        auditlog.AppTargetType,
				Name:        name,
				DisplayName: name,
				Meta: map[string]any{
					"connectionId": id,
					"name":         name,
					"targetAppId":  target.ID,
					"targetType":   req.TargetType,
				},
			}},
		}})
	})
	if db.IsDuplicateKeyError(err) {
		return fault.New("duplicate connection", fault.Code(codes.Data.Connection.Conflict.URN()), fault.Public("This app is already connected, or the name is already in use. Choose another name."))
	}
	if err != nil {
		return fault.Wrap(err, fault.Internal("failed to create connection"))
	}

	return s.JSON(http.StatusCreated, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2ConnectionsCreateConnectionResponseData{Id: id, Name: name},
	})
}
