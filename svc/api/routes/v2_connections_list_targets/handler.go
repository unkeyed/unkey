package handler

import (
	"context"
	"net/http"

	"github.com/oapi-codegen/nullable"
	"github.com/unkeyed/unkey/pkg/auth/principal"
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
	Request  = openapi.V2ConnectionsListTargetsRequestBody
	Response = openapi.V2ConnectionsListTargetsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string { return http.MethodPost }
func (h *Handler) Path() string   { return "/v2/connections.listTargets" }

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
	if req.TargetAppId != nil {
		if _, err := connections.RequireTargetApp(ctx, h.DB.RO(), principal, environment.ProjectID, *req.TargetAppId); err != nil {
			return err
		}
	}
	if req.Cursor != nil && *req.Cursor != "" {
		if req.TargetAppId == nil {
			return fault.New("cursor without target app", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("A deployment cursor requires a target app."))
		}
		cursor, err := connections.RequireDeployment(ctx, h.DB.RO(), principal, *req.Cursor)
		if err != nil {
			return err
		}
		if cursor.ProjectID != environment.ProjectID || cursor.AppID != *req.TargetAppId {
			return fault.New("cursor deployment not found", fault.Code(codes.Data.Deployment.NotFound.URN()), fault.Public("The requested deployment does not exist."))
		}
	}

	apps, err := db.Query.ListAppConnectionTargetApps(ctx, h.DB.RO(), db.ListAppConnectionTargetAppsParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		ProjectID:   environment.ProjectID,
		AppID:       environment.AppID,
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	environments, err := db.Query.ListAppConnectionTargetEnvironments(ctx, h.DB.RO(), db.ListAppConnectionTargetEnvironmentsParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		ProjectID:   environment.ProjectID,
	})
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}

	data := openapi.V2ConnectionsListTargetsResponseData{
		Apps:         make([]openapi.ConnectionTargetApp, 0, len(apps)),
		Environments: make([]openapi.ConnectionTargetEnvironment, 0, len(environments)),
		Deployments:  []openapi.ConnectionTargetDeployment{},
	}
	for _, app := range apps {
		if rbac.Check(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(environment.ProjectID).App(app.ID), permissions.Read), principal.Permissions) != nil {
			continue
		}
		data.Apps = append(data.Apps, openapi.ConnectionTargetApp{Id: app.ID, Name: app.Name, Slug: app.Slug})
	}

	for _, row := range environments {
		if rbac.Check(rbac.U(urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(environment.ProjectID).App(row.AppID).Environment(row.ID), permissions.Read), principal.Permissions) != nil {
			continue
		}
		data.Environments = append(data.Environments, openapi.ConnectionTargetEnvironment{
			Id:    row.ID,
			AppId: row.AppID,
			Slug:  row.Slug,
			Kind:  openapi.ConnectionTargetEnvironmentKind(row.Kind),
		})
	}

	pinned, err := h.pinnedDeployments(ctx, principal, environment)
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()))
	}
	choices, page, err := h.deploymentChoices(ctx, principal, environment.ProjectID, req)
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve deployment choices."))
	}

	seen := make(map[string]bool, len(pinned)+len(choices))
	for _, row := range append(pinned, choices...) {
		if seen[row.ID] || !canReadDeployment(principal, environment.ProjectID, row) {
			continue
		}
		seen[row.ID] = true

		target := openapi.ConnectionTargetDeployment{
			Id:            row.ID,
			AppId:         row.AppID,
			EnvironmentId: row.EnvironmentID,
			Status:        openapi.DeploymentStatus(row.Status),
			GitBranch:     nullable.NewNullNullable[string](),
			Image:         nullable.NewNullNullable[string](),
		}
		if row.GitBranch.Valid {
			target.GitBranch.Set(row.GitBranch.String)
		}
		if row.ImageResolved.Valid {
			target.Image.Set(row.ImageResolved.String)
		}
		data.Deployments = append(data.Deployments, target)
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: page,
	})
}

func (h *Handler) pinnedDeployments(ctx context.Context, principal *principal.Principal, environment db.Environment) ([]db.ListAppConnectionTargetDeploymentsRow, error) {
	pins, err := db.Query.ListAppConnectionPinnedDeploymentIDs(ctx, h.DB.RO(), db.ListAppConnectionPinnedDeploymentIDsParams{
		WorkspaceID:   principal.AuthorizedWorkspaceID,
		ProjectID:     environment.ProjectID,
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil {
		return nil, err
	}

	ids := make([]string, 0, len(pins))
	for _, pin := range pins {
		if pin.Valid && pin.String != "" {
			ids = append(ids, pin.String)
		}
	}
	if len(ids) == 0 {
		return nil, nil
	}

	rows, err := db.Query.ListAppConnectionPinnedDeployments(ctx, h.DB.RO(), db.ListAppConnectionPinnedDeploymentsParams{
		WorkspaceID: principal.AuthorizedWorkspaceID,
		ProjectID:   environment.ProjectID,
		Ids:         ids,
	})
	if err != nil {
		return nil, err
	}

	deployments := make([]db.ListAppConnectionTargetDeploymentsRow, 0, len(rows))
	for _, row := range rows {
		deployments = append(deployments, db.ListAppConnectionTargetDeploymentsRow(row))
	}
	return deployments, nil
}

func (h *Handler) deploymentChoices(ctx context.Context, principal *principal.Principal, projectID string, req Request) ([]db.ListAppConnectionTargetDeploymentsRow, openapi.Pagination, error) {
	if req.TargetAppId == nil {
		return nil, openapi.Pagination{HasMore: false, Cursor: nil}, nil
	}

	page := pagination.Parse(nil, req.Cursor, 100)
	rows, err := pagination.FetchAuthorized(ctx, page,
		func(ctx context.Context, cursor string, limit int32) ([]db.ListAppConnectionTargetDeploymentsRow, error) {
			return db.Query.ListAppConnectionTargetDeployments(ctx, h.DB.RO(), db.ListAppConnectionTargetDeploymentsParams{
				WorkspaceID: principal.AuthorizedWorkspaceID,
				ProjectID:   projectID,
				AppID:       *req.TargetAppId,
				CursorID:    cursor,
				Limit:       limit,
			})
		}, func(row db.ListAppConnectionTargetDeploymentsRow) bool {
			return canReadDeployment(principal, projectID, row)
		}, func(row db.ListAppConnectionTargetDeploymentsRow) string { return row.ID })
	if err != nil {
		return nil, openapi.Pagination{}, err
	}

	rows, result := pagination.Paginate(rows, page, func(row db.ListAppConnectionTargetDeploymentsRow) string { return row.ID })
	return rows, result, nil
}

func canReadDeployment(principal *principal.Principal, projectID string, row db.ListAppConnectionTargetDeploymentsRow) bool {
	resource := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(projectID).App(row.AppID).Environment(row.EnvironmentID).Deployment(row.ID)
	return rbac.Check(rbac.U(resource, permissions.Read), principal.Permissions) == nil
}
