package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type (
	Request  = openapi.V2AppsGetAppRequestBody
	Response = openapi.V2AppsGetAppResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/apps.getApp"
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

	row, err := db.Query.FindAppByWorkspaceAndSlugs(ctx, h.DB.RO(), db.FindAppByWorkspaceAndSlugsParams{
		WorkspaceID: principal.WorkspaceID,
		ProjectSlug: req.ProjectSlug,
		AppSlug:     req.Slug,
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

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   "*",
			Action:       rbac.ReadApp,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Project,
			ResourceID:   row.App.ProjectID,
			Action:       rbac.ReadApp,
		}),
	))
	if err != nil {
		// Mirror the missing-app 404 so an unauthorized key can't probe which
		// apps exist or read identifiers from the authorization error.
		return fault.New(
			"app not found",
			fault.Code(codes.Data.App.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking app existence"),
			fault.Public("The requested app does not exist."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.App{
			Id:                  row.App.ID,
			Name:                row.App.Name,
			Slug:                row.App.Slug,
			ProjectId:           row.App.ProjectID,
			DefaultBranch:       row.App.DefaultBranch,
			CurrentDeploymentId: row.App.CurrentDeploymentID.String,
			IsRolledBack:        row.App.IsRolledBack,
			DeleteProtection:    row.App.DeleteProtection.Bool,
			CreatedAt:           row.App.CreatedAt,
			UpdatedAt:           row.App.UpdatedAt.Int64,
		},
	})
}
