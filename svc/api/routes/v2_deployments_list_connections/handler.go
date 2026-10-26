package handler

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/connections"
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

	rows, err := db.Query.ListDeploymentConnectionSummaries(ctx, h.DB.RO(), db.ListDeploymentConnectionSummariesParams{
		WorkspaceID:  principal.AuthorizedWorkspaceID,
		ProjectID:    deployment.ProjectID,
		DeploymentID: deployment.ID,
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve deployment connections."))
	}

	data := openapi.V2DeploymentsListConnectionsResponseData{
		Connections: make([]openapi.DeploymentConnectionSummary, 0, min(len(rows), 5)),
		Truncated:   len(rows) > 5,
	}
	for _, row := range rows[:min(len(rows), 5)] {
		switch row.TargetType {
		case db.DeploymentConnectionAppTargetsSelectionModeAutomatic:
		case db.DeploymentConnectionAppTargetsSelectionModeEnvironment:
			if err := assert.True(row.TargetEnvironmentID.Valid && row.TargetEnvironmentID.String != "", "saved connection environment is missing"); err != nil {
				return err
			}
		case db.DeploymentConnectionAppTargetsSelectionModeDeployment:
			if err := assert.True(row.TargetDeploymentID.Valid && row.TargetDeploymentID.String != "", "saved connection deployment is missing"); err != nil {
				return err
			}
		default:
			return fault.New("invalid saved connection target type", fault.Code(codes.App.Internal.UnexpectedError.URN()))
		}

		connection := openapi.DeploymentConnectionSummary{
			Id:                    row.ID,
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
		data.Connections = append(data.Connections, connection)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{RequestId: s.RequestID()},
		Data: data,
	})
}
