package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	envmapper "github.com/unkeyed/unkey/svc/api/internal/environment"
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

	runtime, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	var runtimeSettings *db.AppRuntimeSetting
	switch {
	case db.IsNotFound(err):
		// Skip. Settings are missing
	case err != nil:
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	default:
		runtimeSettings = &runtime.AppRuntimeSetting
	}

	build, err := db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	var buildSettings *db.AppBuildSetting
	switch {
	case db.IsNotFound(err):
		// Skip. Settings are missing
	case err != nil:
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	default:
		buildSettings = &build
	}

	regional, err := db.Query.FindAppRegionalSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRegionalSettingsByAppAndEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}
	regions := make([]openapi.EnvironmentRegion, 0, len(regional))
	for _, r := range regional {
		regions = append(regions, envmapper.Region(r.RegionName, r.Replicas, r.AutoscalingReplicasMin, r.AutoscalingReplicasMax))
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: envmapper.ToResponse(envmapper.Params{
			Env:     environment,
			Runtime: runtimeSettings,
			Build:   buildSettings,
			Regions: regions,
		}),
	})
}
