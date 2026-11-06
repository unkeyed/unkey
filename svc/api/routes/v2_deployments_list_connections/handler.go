package handler

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/nullable"
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
	Request  = openapi.V2DeploymentsListConnectionsRequestBody
	Response = openapi.V2DeploymentsListConnectionsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/deployments.listConnections" }

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
	rows, err := pagination.FetchAuthorized(ctx, page, func(ctx context.Context, cursor string, limit int32) ([]db.ListDeploymentConnectionSummariesRow, error) {
		return db.Query.ListDeploymentConnectionSummaries(ctx, h.DB.RO(), db.ListDeploymentConnectionSummariesParams{
			WorkspaceID:  principal.AuthorizedWorkspaceID,
			ProjectID:    deployment.ProjectID,
			DeploymentID: deployment.ID,
			IDCursor:     cursor,
			Limit:        limit,
		})
	}, func(row db.ListDeploymentConnectionSummariesRow) bool {
		resource := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(deployment.ProjectID).App(row.TargetAppID)
		return rbac.Check(rbac.U(resource, permissions.Read), principal.Permissions) == nil
	}, func(row db.ListDeploymentConnectionSummariesRow) string { return row.ID })
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve deployment connections."))
	}

	rows, pg := pagination.Paginate(rows, page, func(row db.ListDeploymentConnectionSummariesRow) string { return row.ID })
	data := make([]openapi.DeploymentConnectionSummary, 0, len(rows))
	for _, row := range rows {
		connection := openapi.DeploymentConnectionSummary{
			Id:                    row.ID,
			TargetAppId:           row.TargetAppID,
			TargetType:            openapi.ConnectionTargetType(row.TargetType),
			TargetAppName:         nullable.NewNullNullable[string](),
			TargetEnvironmentId:   nullable.NewNullNullable[string](),
			TargetEnvironmentSlug: nullable.NewNullNullable[string](),
			TargetDeploymentId:    nullable.NewNullNullable[string](),
		}
		if row.TargetAppName.Valid {
			connection.TargetAppName.Set(row.TargetAppName.String)
		}
		if row.TargetEnvironmentID.Valid {
			connection.TargetEnvironmentId.Set(row.TargetEnvironmentID.String)
		}
		if row.TargetEnvironmentSlug.Valid {
			connection.TargetEnvironmentSlug.Set(row.TargetEnvironmentSlug.String)
		}
		if row.TargetDeploymentID.Valid {
			connection.TargetDeploymentId.Set(row.TargetDeploymentID.String)
		}
		data = append(data, connection)
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pg,
	})
}
