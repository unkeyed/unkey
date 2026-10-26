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
	target, err := connections.RequireTargetApp(ctx, h.DB.RO(), principal, environment.ProjectID, req.TargetAppId)
	if err != nil {
		return err
	}
	if target.ID == environment.AppID {
		return fault.New("self connection", fault.Code(codes.App.Validation.InvalidInput.URN()), fault.Public("Choose another app. Connections within this app are automatic."))
	}

	canRead := func(environmentID, deploymentID string) bool {
		resource := urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(target.ProjectID).App(target.ID).Environment(environmentID).Deployment(deploymentID)
		return rbac.Check(rbac.U(resource, permissions.Read), principal.Permissions) == nil
	}

	page := pagination.Parse(req.Limit, req.Cursor, 100)
	rows, err := pagination.FetchAuthorized(ctx, page, func(ctx context.Context, cursor string, limit int32) ([]db.ListAppConnectionTargetDeploymentsRow, error) {
		return db.Query.ListAppConnectionTargetDeployments(ctx, h.DB.RO(), db.ListAppConnectionTargetDeploymentsParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			AppID:       target.ID,
			CursorID:    cursor,
			Limit:       limit,
		})
	}, func(row db.ListAppConnectionTargetDeploymentsRow) bool {
		return canRead(row.EnvironmentID, row.ID)
	}, func(row db.ListAppConnectionTargetDeploymentsRow) string { return row.ID })
	if err != nil {
		return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve target deployments."))
	}

	rows, pg := pagination.Paginate(rows, page, func(row db.ListAppConnectionTargetDeploymentsRow) string { return row.ID })
	data := openapi.V2ConnectionsListTargetsResponseData{
		Deployments: make([]openapi.ConnectionTargetDeployment, 0, len(rows)),
		Pinned:      nil,
	}
	for _, row := range rows {
		data.Deployments = append(data.Deployments, targetDeployment(row))
	}

	if page.Cursor == "" {
		pinned, err := db.Query.FindAppConnectionPinnedDeployment(ctx, h.DB.RO(), db.FindAppConnectionPinnedDeploymentParams{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			AppID:         environment.AppID,
			EnvironmentID: environment.ID,
			TargetAppID:   target.ID,
		})
		if err != nil && !db.IsNotFound(err) {
			return fault.Wrap(err, fault.Code(codes.App.Internal.ServiceUnavailable.URN()), fault.Public("Failed to retrieve the pinned deployment."))
		}
		if err == nil && canRead(pinned.EnvironmentID, pinned.ID) {
			data.Pinned = new(targetDeployment(db.ListAppConnectionTargetDeploymentsRow(pinned)))
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta:       openapi.Meta{RequestId: s.RequestID()},
		Data:       data,
		Pagination: pg,
	})
}

func targetDeployment(row db.ListAppConnectionTargetDeploymentsRow) openapi.ConnectionTargetDeployment {
	deployment := openapi.ConnectionTargetDeployment{
		Id:            row.ID,
		EnvironmentId: row.EnvironmentID,
		Status:        openapi.DeploymentStatus(row.Status),
		GitBranch:     nullable.NewNullNullable[string](),
		Image:         nullable.NewNullNullable[string](),
	}
	if row.GitBranch.Valid {
		deployment.GitBranch.Set(row.GitBranch.String)
	}
	if row.ImageResolved.Valid {
		deployment.Image.Set(row.ImageResolved.String)
	}
	return deployment
}
