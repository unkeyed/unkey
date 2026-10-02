package handler

import (
	"context"
	"net/http"
	"strconv"

	"github.com/unkeyed/unkey/pkg/array"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/rbac/permissions"
	"github.com/unkeyed/unkey/pkg/urn"
	"github.com/unkeyed/unkey/pkg/zen"
	apierrors "github.com/unkeyed/unkey/svc/api/internal/errors"
	"github.com/unkeyed/unkey/svc/api/internal/pagination"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2DeploymentsGetBuildLogsRequestBody
	Response = openapi.V2DeploymentsGetBuildLogsResponseBody
)

type Handler struct {
	DB         db.Database
	ClickHouse clickhouse.ClickHouse
	Clock      clock.Clock
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/deployments.getBuildLogs"
}

func (h *Handler) Handle(ctx context.Context, s *zen.Session) error {
	// Build output can echo secrets and a poll can return about 1 MiB, so the
	// response body must not be copied into api_requests_raw_v2
	s.DisableClickHouseLogging()

	principal, err := s.GetPrincipal()
	if err != nil {
		return err
	}

	req, err := zen.BindBody[Request](s)
	if err != nil {
		return err
	}

	params := pagination.Parse(req.Limit, req.Cursor, 100)
	var afterSeq uint64
	if params.Cursor != "" {
		afterSeq, err = strconv.ParseUint(params.Cursor, 10, 64)
		if err != nil {
			return fault.Wrap(err,
				fault.Code(codes.App.Validation.InvalidInput.URN()),
				fault.Internal("invalid build logs cursor"),
				fault.Public("The cursor is not valid. Send the pagination.cursor of a previous response, or omit it to start at the first entry."),
			)
		}
	}

	dep, err := db.Query.FindDeploymentById(ctx, h.DB.RO(), req.DeploymentId)
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve deployment."),
		)
	}

	// FindDeploymentById is not workspace-scoped, so a match in another workspace
	// is masked as not found to avoid leaking a deployment's existence
	if db.IsNotFound(err) || dep.WorkspaceID != principal.AuthorizedWorkspaceID {
		return fault.New(
			"deployment not found",
			fault.Code(codes.Data.Deployment.NotFound.URN()),
			fault.Internal("deployment not found or belongs to another workspace"),
			fault.Public("The requested deployment does not exist."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.ReadDeployment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   dep.EnvironmentID,
			Action:       rbac.ReadDeployment,
		}),
		rbac.U(
			urn.New().Workspace(principal.AuthorizedWorkspaceID).Project(dep.ProjectID).App(dep.AppID).Environment(dep.EnvironmentID).Deployment(dep.ID),
			permissions.Read,
		),
	))
	if err != nil {
		return apierrors.MaskInsufficientPermissionsAsNotFound(err, codes.Data.Deployment.NotFound.URN(), "The requested deployment does not exist.")
	}

	page, err := h.ClickHouse.GetBuildLogs(ctx, clickhouse.GetBuildLogsRequest{
		WorkspaceID:  principal.AuthorizedWorkspaceID,
		ProjectID:    dep.ProjectID,
		DeploymentID: dep.ID,
		StepID:       ptr.SafeDeref(req.StepId),
		AfterSeq:     afterSeq,
		Limit:        params.Limit,
		Now:          h.Clock.Now(),
	})
	if err != nil {
		return fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("clickhouse error"),
			fault.Public("Failed to retrieve build logs."),
		)
	}

	cursor := req.Cursor
	if len(page.Entries) > 0 {
		cursor = new(strconv.FormatUint(page.Entries[len(page.Entries)-1].Seq, 10))
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: array.Map(page.Entries, func(entry clickhouse.BuildLogEntry) openapi.BuildLogEntry {
			output := openapi.BuildLogOutputStdout
			if entry.Stderr {
				output = openapi.BuildLogOutputStderr
			}
			return openapi.BuildLogEntry{
				Time:    entry.Time,
				StepId:  entry.StepID,
				Step:    entry.Step,
				Output:  output,
				Message: entry.Message,
			}
		}),
		Pagination: openapi.Pagination{
			Cursor:  cursor,
			HasMore: page.HasMore,
		},
	})
}
