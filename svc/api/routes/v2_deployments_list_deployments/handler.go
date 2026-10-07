package handler

import (
	"cmp"
	"context"
	"database/sql"
	"net/http"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsListDeploymentsRequestBody
	Response = openapi.V2DeploymentsListDeploymentsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.listDeployments"
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

	page := pagination.Parse(req.Limit, req.Cursor, 100)

	// Every rejection authorizes against the scope resolved so far, and a denial
	// names no resolved ids, so a caller that cannot read a scope gets the same
	// 403 whether or not the resource exists
	authorize := func(projectID, appID, environmentID string) error {
		err := principal.Authorize(rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).
				Project(cmp.Or(projectID, "*")).
				App(cmp.Or(appID, "*")).
				Environment(cmp.Or(environmentID, "*")).
				Deployment("*"),
			permissions.Read,
		))
		if err != nil {
			return fault.New("insufficient permissions to list deployments",
				fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
				fault.Internal(fault.InternalMessage(err)),
				fault.Public("You cannot read deployments in the requested scope. Grant read on projects/<project>/apps/<app>/environments/<environment>/deployments/* with * for every level you do not filter by."),
			)
		}
		return nil
	}

	// Filters nest: an app lives in a project, an environment lives in an app.
	// Requiring the parents keeps resolution unambiguous when a slug is passed.
	if req.App != nil && req.Project == nil {
		if err = authorize("", "", ""); err != nil {
			return err
		}
		return fault.New(
			"app filter without project",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("app filter requires project"),
			fault.Public("The 'app' filter requires 'project' to be set."),
		)
	}
	if req.Environment != nil && (req.App == nil || req.Project == nil) {
		if err = authorize("", "", ""); err != nil {
			return err
		}
		return fault.New(
			"environment filter without parents",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("environment filter requires project and app"),
			fault.Public("The 'environment' filter requires both 'project' and 'app' to be set."),
		)
	}

	var projectID, appID, environmentID string
	if req.Project != nil {
		scope, err := db.Query.ResolveDeploymentScope(ctx, h.DB.RO(), db.ResolveDeploymentScopeParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Project:     *req.Project,
			App:         ptr.SafeDeref(req.App, ""),
			Environment: ptr.SafeDeref(req.Environment, ""),
		})
		if err != nil {
			if db.IsNotFound(err) {
				if err = authorize("", "", ""); err != nil {
					return err
				}
				return fault.New(
					"project not found",
					fault.Code(codes.Data.Project.NotFound.URN()),
					fault.Internal("project not found"),
					fault.Public("The requested project does not exist."),
				)
			}
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve deployments."),
			)
		}
		projectID = scope.ProjectID

		if req.App != nil {
			if !scope.AppID.Valid {
				if err = authorize(scope.ProjectID, "", ""); err != nil {
					return err
				}
				return fault.New(
					"app not found",
					fault.Code(codes.Data.App.NotFound.URN()),
					fault.Internal("app not found"),
					fault.Public("The requested app does not exist."),
				)
			}
			appID = scope.AppID.String
		}
		if req.Environment != nil {
			if !scope.EnvironmentID.Valid {
				if err = authorize(scope.ProjectID, appID, ""); err != nil {
					return err
				}
				return fault.New(
					"environment not found",
					fault.Code(codes.Data.Environment.NotFound.URN()),
					fault.Internal("environment not found"),
					fault.Public("The requested environment does not exist."),
				)
			}
			environmentID = scope.EnvironmentID.String
		}
	}

	if err = authorize(projectID, appID, environmentID); err != nil {
		return err
	}

	branches := ptr.SafeDeref(req.Branch, nil)
	if len(branches) > 0 && appID == "" {
		return fault.New(
			"branch filter without parents",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("branch filter requires project and app"),
			fault.Public("The 'branch' filter requires both 'project' and 'app' to be set."),
		)
	}
	if req.StartTime != nil && req.EndTime != nil && *req.StartTime >= *req.EndTime {
		return fault.New(
			"empty time range",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("startTime is not before endTime"),
			fault.Public("'startTime' must be earlier than 'endTime'."),
		)
	}

	var statuses []mysqltype.DeploymentsStatus
	if req.Status != nil {
		statuses = make([]mysqltype.DeploymentsStatus, len(*req.Status))
		for i, st := range *req.Status {
			statuses[i] = mysqltype.DeploymentsStatus(st)
		}
	}

	branchFilter := make([]sql.NullString, len(branches))
	for i, branch := range branches {
		branchFilter[i] = sql.NullString{String: branch, Valid: true}
	}

	rows, err := db.Query.ListDeployments(ctx, h.DB.RO(), db.ListDeploymentsParams{
		WorkspaceID:     principal.AuthorizedWorkspaceID,
		ProjectID:       projectID,
		AppID:           appID,
		EnvironmentID:   environmentID,
		HasStatusFilter: len(statuses) > 0,
		Statuses:        statuses,
		HasBranchFilter: len(branchFilter) > 0,
		Branches:        branchFilter,
		StartTime:       sql.NullInt64{Int64: ptr.SafeDeref(req.StartTime, 0), Valid: req.StartTime != nil},
		EndTime:         sql.NullInt64{Int64: ptr.SafeDeref(req.EndTime, 0), Valid: req.EndTime != nil},
		CursorID:        page.Cursor,
		Limit:           page.FetchLimit(),
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployments."),
		)
	}

	rows, pg := pagination.Paginate(rows, page, func(r db.ListDeploymentsRow) string { return r.ID })

	data := make([]openapi.Deployment, len(rows))
	if len(rows) > 0 {
		ids := make([]string, len(rows))
		for i, row := range rows {
			ids[i] = row.ID
		}
		state, err := db.Query.ListDeploymentEnvAndAppState(ctx, h.DB.RO(), db.ListDeploymentEnvAndAppStateParams{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			DeploymentIds: ids,
		})
		if err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve deployments."),
			)
		}
		byID := make(map[string]db.ListDeploymentEnvAndAppStateRow, len(state))
		for _, r := range state {
			byID[r.DeploymentID] = r
		}

		regionRows, err := db.Query.ListDeploymentRegionsByIds(ctx, h.DB.RO(), db.ListDeploymentRegionsByIdsParams{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			DeploymentIds: ids,
		})
		if err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve deployments."),
			)
		}
		regionsByID := make(map[string][]string, len(rows))
		for _, rr := range regionRows {
			regionsByID[rr.DeploymentID] = append(regionsByID[rr.DeploymentID], rr.Region)
		}

		stepRows, err := db.Query.ListDeploymentStepsByIds(ctx, h.DB.RO(), db.ListDeploymentStepsByIdsParams{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			DeploymentIds: ids,
		})
		if err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve deployments."),
			)
		}
		stepsByID := make(map[string][]db.DeploymentStep, len(rows))
		for _, sr := range stepRows {
			stepsByID[sr.DeploymentID] = append(stepsByID[sr.DeploymentID], sr)
		}

		domainRows, err := db.Query.ListDeploymentDomainsByIds(ctx, h.DB.RO(), db.ListDeploymentDomainsByIdsParams{
			WorkspaceID:   principal.AuthorizedWorkspaceID,
			DeploymentIds: ids,
		})
		if err != nil {
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve deployments."),
			)
		}
		domainsByID := make(map[string][]string, len(rows))
		for _, dr := range domainRows {
			domainsByID[dr.DeploymentID] = append(domainsByID[dr.DeploymentID], dr.Domain)
		}

		for i, row := range rows {
			data[i] = deployment.ToResponse(deployment.Input{
				Deployment: row,
				State:      byID[row.ID],
				Regions:    regionsByID[row.ID],
				Steps:      stepsByID[row.ID],
				Domains:    domainsByID[row.ID],
			})
		}
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       data,
		Pagination: pg,
	})
}
