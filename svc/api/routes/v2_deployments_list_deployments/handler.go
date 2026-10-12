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

	// Filters nest: an app lives in a project, an environment lives in an app.
	// Requiring the parents keeps resolution unambiguous when a slug is passed.
	if req.App != nil && req.Project == nil {
		return fault.New(
			"app filter without project",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("app filter requires project"),
			fault.Public("The 'app' filter requires 'project' to be set."),
		)
	}
	if req.Environment != nil && (req.App == nil || req.Project == nil) {
		return fault.New(
			"environment filter without parents",
			fault.Code(codes.App.Validation.InvalidInput.URN()),
			fault.Internal("environment filter requires project and app"),
			fault.Public("The 'environment' filter requires both 'project' and 'app' to be set."),
		)
	}
	branches := ptr.SafeDeref(req.Branch, nil)
	if len(branches) > 0 && (req.App == nil || req.Project == nil) {
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

	// A missing project, app, or environment is reported only after the caller
	// passes authorization on the levels resolved before it, so a caller that
	// cannot read a scope gets the same 403 whether or not the resource exists
	var projectID, appID, environmentID string
	var notFound error
	if req.Project != nil {
		scope, err := db.Query.ResolveDeploymentScope(ctx, h.DB.RO(), db.ResolveDeploymentScopeParams{
			WorkspaceID: principal.AuthorizedWorkspaceID,
			Project:     *req.Project,
			App:         ptr.SafeDeref(req.App, ""),
			Environment: ptr.SafeDeref(req.Environment, ""),
		})
		switch {
		case db.IsNotFound(err):
			notFound = fault.New(
				"project not found",
				fault.Code(codes.Data.Project.NotFound.URN()),
				fault.Internal("project not found"),
				fault.Public("The requested project does not exist."),
			)
		case err != nil:
			return fault.Wrap(
				err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("database error"),
				fault.Public("Failed to retrieve deployments."),
			)
		case req.App != nil && !scope.AppID.Valid:
			projectID = scope.ProjectID
			notFound = fault.New(
				"app not found",
				fault.Code(codes.Data.App.NotFound.URN()),
				fault.Internal("app not found"),
				fault.Public("The requested app does not exist."),
			)
		case req.Environment != nil && !scope.EnvironmentID.Valid:
			projectID, appID = scope.ProjectID, scope.AppID.String
			notFound = fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment not found"),
				fault.Public("The requested environment does not exist."),
			)
		default:
			projectID, appID, environmentID = scope.ProjectID, scope.AppID.String, scope.EnvironmentID.String
		}
	}

	err = principal.Authorize(rbac.U(
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
	if notFound != nil {
		return notFound
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

	data, err := deployment.ToResponses(ctx, h.DB.RO(), principal.AuthorizedWorkspaceID, rows)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployments."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data:       data,
		Pagination: pg,
	})
}
