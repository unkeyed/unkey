package handler

import (
	"context"
	"database/sql"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/connections"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsListConnectionPinsRequestBody
	Response = openapi.V2DeploymentsListConnectionPinsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/deployments.listConnectionPins" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	deployment, err := connections.RequireDeployment(ctx, h.DB.RO(), principal, req.DeploymentId)
	if err != nil {
		return err
	}

	current, err := db.Query.ListCurrentDeploymentConnectionPins(ctx, h.DB.RO(), db.ListCurrentDeploymentConnectionPinsParams{
		WorkspaceID:  principal.AuthorizedWorkspaceID,
		ProjectID:    deployment.ProjectID,
		TargetAppID:  deployment.AppID,
		DeploymentID: sql.NullString{String: deployment.ID, Valid: true},
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	snapshots, err := db.Query.ListSnapshotDeploymentConnectionPins(ctx, h.DB.RO(), db.ListSnapshotDeploymentConnectionPinsParams{
		WorkspaceID:  principal.AuthorizedWorkspaceID,
		ProjectID:    deployment.ProjectID,
		TargetAppID:  deployment.AppID,
		DeploymentID: sql.NullString{String: deployment.ID, Valid: true},
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	pins := make([]openapi.DeploymentConnectionPin, 0, len(current)+len(snapshots))
	for _, row := range current {
		var pin openapi.DeploymentConnectionPin
		if err := pin.FromCurrentConnectionPin(openapi.CurrentConnectionPin{
			ConnectionId:   row.ConnectionID,
			ConnectionName: row.ConnectionName,
			CallerAppId:    row.CallerAppID,
			CallerAppName:  row.CallerAppName,
			Provenance:     openapi.CurrentRule,
		}); err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()))
		}
		pins = append(pins, pin)
	}

	for _, row := range snapshots {
		var pin openapi.DeploymentConnectionPin
		if err := pin.FromSnapshotConnectionPin(openapi.SnapshotConnectionPin{
			ConnectionId:       row.ConnectionID,
			ConnectionName:     row.ConnectionName,
			CallerAppId:        row.CallerAppID,
			CallerAppName:      row.CallerAppName,
			CallerDeploymentId: row.CallerDeploymentID,
			CallerStatus:       openapi.DeploymentStatus(row.CallerStatus),
			Provenance:         openapi.DeploymentSnapshot,
		}); err != nil {
			return fault.Wrap(err, fault.Code(codes.App.Internal.UnexpectedError.URN()))
		}
		pins = append(pins, pin)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: openapi.V2DeploymentsListConnectionPinsResponseData{
			Pins:      pins[:min(len(pins), 10)],
			Truncated: len(pins) > 10,
		},
	})
}
