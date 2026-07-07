package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/deployment"
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

	var projectID, appID, environmentID string
	if req.Project != nil {
		scope, err := db.Query.ResolveDeploymentScope(ctx, h.DB.RO(), db.ResolveDeploymentScopeParams{
			WorkspaceID: principal.WorkspaceID,
			Project:     *req.Project,
			App:         ptr.SafeDeref(req.App, ""),
			Environment: ptr.SafeDeref(req.Environment, ""),
		})
		if err != nil {
			// project/app/environment not-found all return the project-not-found code
			// so a caller cannot probe which level was missing.
			if db.IsNotFound(err) {
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
				return fault.New(
					"app not found",
					fault.Code(codes.Data.Project.NotFound.URN()),
					fault.Internal("app not found"),
					fault.Public("The requested app does not exist."),
				)
			}
			appID = scope.AppID.String
		}
		if req.Environment != nil {
			if !scope.EnvironmentID.Valid {
				return fault.New(
					"environment not found",
					fault.Code(codes.Data.Project.NotFound.URN()),
					fault.Internal("environment not found"),
					fault.Public("The requested environment does not exist."),
				)
			}
			environmentID = scope.EnvironmentID.String
		}
	}

	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Environment,
		ResourceID:   "*",
		Action:       rbac.ReadDeployment,
	}))
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.Auth.Authorization.InsufficientPermissions.URN()),
			fault.Internal("insufficient permissions"),
			fault.Public("Your root key requires the 'environment.*.read_deployment' permission to perform this operation."),
		)
	}

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")

	var statuses []db.DeploymentsStatus
	if req.Status != nil {
		statuses = make([]db.DeploymentsStatus, len(*req.Status))
		for i, st := range *req.Status {
			statuses[i] = db.DeploymentsStatus(st)
		}
	}

	rows, err := db.Query.ListDeployments(ctx, h.DB.RO(), db.ListDeploymentsParams{
		WorkspaceID:     principal.WorkspaceID,
		ProjectID:       projectID,
		AppID:           appID,
		EnvironmentID:   environmentID,
		HasStatusFilter: len(statuses) > 0,
		Statuses:        statuses,
		CursorID:        cursor,
		Limit:           int32(limit + 1), // nolint:gosec
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployments."),
		)
	}

	hasMore := len(rows) > limit
	var nextCursor *string
	if hasMore {
		nextCursor = ptr.P(rows[limit-1].ID)
		rows = rows[:limit]
	}

	data := make([]openapi.Deployment, len(rows))
	for i, row := range rows {
		data[i] = deployment.ToResponse(row)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
		Pagination: &openapi.Pagination{
			Cursor:  nextCursor,
			HasMore: hasMore,
		},
	})
}
