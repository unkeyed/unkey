package handler

import (
	"context"
	"database/sql"
	"net/http"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/connections"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
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

type pin struct {
	cursor      string
	callerAppID string
	value       openapi.DeploymentConnectionPin
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

	deployment, err := connections.RequireDeployment(ctx, h.DB.RO(), principal, req.DeploymentId)
	if err != nil {
		return err
	}

	page := pagination.Parse(req.Limit, req.Cursor, 100)
	pins, err := pagination.FetchAuthorized(ctx, page, func(ctx context.Context, cursor string, limit int32) ([]pin, error) {
		return h.listPins(ctx, principal.AuthorizedWorkspaceID, deployment, cursor, limit)
	}, func(row pin) bool {
		resource := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(deployment.ProjectID).App(row.callerAppID)
		return rbac.Check(rbac.U(resource, permissions.Read), principal.Permissions) == nil
	}, func(row pin) string { return row.cursor })
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve connection pins."))
	}

	pins, pg := pagination.Paginate(pins, page, func(row pin) string { return row.cursor })
	data := make([]openapi.DeploymentConnectionPin, 0, len(pins))
	for _, row := range pins {
		data = append(data, row.value)
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pg,
	})
}

func (h *Handler) listPins(ctx context.Context, workspaceID string, deployment db.FindDeploymentConnectionScopeRow, cursor string, limit int32) ([]pin, error) {
	target := sql.NullString{String: deployment.ID, Valid: true}
	current, err := db.Query.ListCurrentDeploymentConnectionPins(ctx, h.DB.RO(), db.ListCurrentDeploymentConnectionPinsParams{
		WorkspaceID:  workspaceID,
		ProjectID:    deployment.ProjectID,
		TargetAppID:  deployment.AppID,
		DeploymentID: target,
		PinCursor:    cursor,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}

	snapshots, err := db.Query.ListSnapshotDeploymentConnectionPins(ctx, h.DB.RO(), db.ListSnapshotDeploymentConnectionPinsParams{
		WorkspaceID:  workspaceID,
		ProjectID:    deployment.ProjectID,
		TargetAppID:  deployment.AppID,
		DeploymentID: target,
		PinCursor:    cursor,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}

	pins := make([]pin, 0, len(current)+len(snapshots))
	for _, row := range current {
		var value openapi.DeploymentConnectionPin
		if err := value.FromCurrentConnectionPin(openapi.CurrentConnectionPin{
			ConnectionId:   row.ConnectionID,
			ConnectionName: row.ConnectionName,
			CallerAppId:    row.CallerAppID,
			CallerAppName:  row.CallerAppName,
			Provenance:     openapi.CurrentRule,
		}); err != nil {
			return nil, err
		}
		pins = append(pins, pin{cursor: row.ConnectionID + "/", callerAppID: row.CallerAppID, value: value})
	}
	for _, row := range snapshots {
		var value openapi.DeploymentConnectionPin
		if err := value.FromSnapshotConnectionPin(openapi.SnapshotConnectionPin{
			ConnectionId:       row.ConnectionID,
			ConnectionName:     row.ConnectionName,
			CallerAppId:        row.CallerAppID,
			CallerAppName:      row.CallerAppName,
			CallerDeploymentId: row.CallerDeploymentID,
			CallerStatus:       openapi.DeploymentStatus(row.CallerStatus),
			Provenance:         openapi.DeploymentSnapshot,
		}); err != nil {
			return nil, err
		}
		pins = append(pins, pin{cursor: row.ConnectionID + "/" + row.CallerDeploymentID, callerAppID: row.CallerAppID, value: value})
	}

	slices.SortFunc(pins, func(a, b pin) int { return strings.Compare(a.cursor, b.cursor) })
	return pins[:min(len(pins), int(limit))], nil
}
