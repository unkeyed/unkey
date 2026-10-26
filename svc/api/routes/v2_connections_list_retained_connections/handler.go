package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/connections"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ConnectionsListRetainedConnectionsRequestBody
	Response = openapi.V2ConnectionsListRetainedConnectionsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/connections.listRetainedConnections" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	environment, err := connections.RequireEnvironment(ctx, h.DB.RO(), principal, req.Project, req.App, req.Environment)
	if err != nil {
		return err
	}

	rows, err := db.Query.ListRetainedAppConnections(ctx, h.DB.RO(), db.ListRetainedAppConnectionsParams{
		WorkspaceID:   principal.AuthorizedWorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve retained connections."))
	}

	data := make([]openapi.RetainedAppConnection, 0, len(rows))
	for _, row := range rows {
		data = append(data, openapi.RetainedAppConnection{
			Id:            row.ID,
			TargetAppName: row.TargetAppName,
			Deployments:   row.Deployments,
		})
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
