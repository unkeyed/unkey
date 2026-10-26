package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/assert"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2ConnectionsListConnectionsRequestBody
	Response = openapi.V2ConnectionsListConnectionsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/connections.listConnections" }

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	environment, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if db.IsNotFound(err) {
		return fault.New("environment not found", fault.Code(codes.Data.Environment.NotFound.URN()), fault.Public("The requested environment does not exist."))
	}
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve environment."))
	}

	err = principal.Authorize(rbac.U(
		urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(environment.ProjectID).App(environment.AppID).Environment(environment.ID),
		permissions.Read,
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Environment.NotFound.URN(), "The requested environment does not exist.")
	}

	page := pagination.Parse(req.Limit, req.Cursor, 100)
	rows, err := db.Query.ListAppConnections(ctx, h.DB.RO(), db.ListAppConnectionsParams{
		WorkspaceID:   principal.AuthorizedWorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
		IDCursor:      page.Cursor,
		Limit:         page.FetchLimit(),
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve connections."))
	}

	rows, pagination := pagination.Paginate(rows, page, func(row db.ListAppConnectionsRow) string { return row.ID })
	data := make([]openapi.AppConnection, 0, len(rows))
	for _, row := range rows {
		connection := openapi.AppConnection{
			Id:                  row.ID,
			AppId:               row.AppID,
			EnvironmentId:       row.EnvironmentID,
			Name:                row.Name,
			TargetAppId:         row.TargetAppID,
			TargetAppName:       row.TargetAppName,
			TargetAppSlug:       row.TargetAppSlug,
			TargetType:          openapi.ConnectionTargetType(row.SelectionMode.ConnectionAppTargetsSelectionMode),
			TargetEnvironmentId: nil,
			TargetDeploymentId:  nil,
		}
		switch connection.TargetType {
		case openapi.ConnectionTargetTypeAutomatic:
		case openapi.ConnectionTargetTypeEnvironment:
			if err := assert.True(row.TargetEnvironmentID.Valid && row.TargetEnvironmentID.String != "", "connection environment is missing"); err != nil {
				return err
			}
			connection.TargetEnvironmentId = new(row.TargetEnvironmentID.String)
		case openapi.ConnectionTargetTypeDeployment:
			if err := assert.True(row.TargetDeploymentID.Valid && row.TargetDeploymentID.String != "", "connection deployment is missing"); err != nil {
				return err
			}
			connection.TargetDeploymentId = new(row.TargetDeploymentID.String)
		default:
			return fault.New("invalid stored connection target", fault.Code(codes.App.Internal.UnexpectedError.URN()))
		}
		data = append(data, connection)
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pagination,
	})
}
