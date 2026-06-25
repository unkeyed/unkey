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
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2EnvironmentsListEnvironmentsRequestBody
	Response = openapi.V2EnvironmentsListEnvironmentsResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/environments.listEnvironments"
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

	app, err := db.Query.FindAppByProjectAndIdOrSlug(ctx, h.DB.RO(), db.FindAppByProjectAndIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"app not found",
				fault.Code(codes.Data.App.NotFound.URN()),
				fault.Internal("app not found"),
				fault.Public("The requested app does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve app."),
		)
	}

	err = principal.Authorize(rbac.T(rbac.Tuple{
		ResourceType: rbac.Environment,
		ResourceID:   "*",
		Action:       rbac.ReadEnvironment,
	}))
	if err != nil {
		return fault.New(
			"app not found",
			fault.Code(codes.Data.App.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking app existence"),
			fault.Public("The requested app does not exist."),
		)
	}

	limit := ptr.SafeDeref(req.Limit, 100)
	cursor := ptr.SafeDeref(req.Cursor, "")

	rows, err := db.Query.ListEnvironmentsByApp(ctx, h.DB.RO(), db.ListEnvironmentsByAppParams{
		AppID:    app.App.ID,
		IDCursor: cursor,
		Limit:    int32(limit + 1), // nolint:gosec
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environments."),
		)
	}

	hasMore := len(rows) > limit
	var nextCursor *string
	if hasMore {
		nextCursor = ptr.P(rows[limit].Environment.ID)
		rows = rows[:limit]
	}

	data := make([]openapi.Environment, len(rows))
	for i, row := range rows {
		data[i] = openapi.Environment{
			Id:               row.Environment.ID,
			ProjectId:        row.Environment.ProjectID,
			AppId:            row.Environment.AppID,
			Slug:             row.Environment.Slug,
			Description:      row.Environment.Description,
			DeleteProtection: row.Environment.DeleteProtection.Bool,
			CreatedAt:        row.Environment.CreatedAt,
			UpdatedAt:        row.Environment.UpdatedAt.Int64,
		}
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
