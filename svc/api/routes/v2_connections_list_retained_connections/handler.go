package handler

import (
	"context"
	"net/http"

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

	page := pagination.Parse(req.Limit, req.Cursor, 100)
	rows, err := pagination.FetchAuthorized(ctx, page, func(ctx context.Context, cursor string, limit int32) ([]db.ListRetainedAppConnectionsRow, error) {
		return db.Query.ListRetainedAppConnections(ctx, h.DB.RO(), db.ListRetainedAppConnectionsParams{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			ProjectID:     environment.ProjectID,
			AppID:         environment.AppID,
			EnvironmentID: environment.ID,
			IDCursor:      cursor,
			Limit:         limit,
		})
	}, func(row db.ListRetainedAppConnectionsRow) bool {
		resource := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(environment.ProjectID).App(row.TargetAppID)
		return rbac.Check(rbac.U(resource, permissions.Read), principal.Permissions) == nil
	}, func(row db.ListRetainedAppConnectionsRow) string { return row.ID })
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve retained connections."))
	}

	rows, pg := pagination.Paginate(rows, page, func(row db.ListRetainedAppConnectionsRow) string { return row.ID })
	data := make([]openapi.RetainedAppConnection, 0, len(rows))
	for _, row := range rows {
		data = append(data, openapi.RetainedAppConnection{
			Id:            row.ID,
			TargetAppId:   row.TargetAppID,
			TargetAppName: row.TargetAppName,
			Deployments:   row.Deployments,
		})
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pg,
	})
}
