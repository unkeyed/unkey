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
	Request  = openapi.V2EnvironmentsGetEnvironmentRequestBody
	Response = openapi.V2EnvironmentsGetEnvironmentResponseBody
)

type Handler struct {
	DB db.Database
}

func (h *Handler) Method() string {
	return "POST"
}

func (h *Handler) Path() string {
	return "/v2/environments.getEnvironment"
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

	environment, err := db.Query.FindEnvironmentByAppAndIdOrSlug(ctx, h.DB.RO(), db.FindEnvironmentByAppAndIdOrSlugParams{
		WorkspaceID: principal.WorkspaceID,
		Project:     req.Project,
		App:         req.App,
		Environment: req.Environment,
	})
	if err != nil {
		if db.IsNotFound(err) {
			return fault.New(
				"environment not found",
				fault.Code(codes.Data.Environment.NotFound.URN()),
				fault.Internal("environment not found"),
				fault.Public("The requested environment does not exist."),
			)
		}
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}

	err = principal.Authorize(rbac.Or(
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   "*",
			Action:       rbac.ReadEnvironment,
		}),
		rbac.T(rbac.Tuple{
			ResourceType: rbac.Environment,
			ResourceID:   environment.ID,
			Action:       rbac.ReadEnvironment,
		}),
	))
	if err != nil {
		return fault.New(
			"environment not found",
			fault.Code(codes.Data.Environment.NotFound.URN()),
			fault.Internal("authorization failed; returning not found to avoid leaking environment existence"),
			fault.Public("The requested environment does not exist."),
		)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: openapi.Environment{
			Id:               environment.ID,
			ProjectId:        environment.ProjectID,
			AppId:            environment.AppID,
			Slug:             environment.Slug,
			Description:      environment.Description,
			DeleteProtection: environment.DeleteProtection.Bool,
			CreatedAt:        environment.CreatedAt,
			UpdatedAt:        environment.UpdatedAt.Int64,
		},
	})
}
