package handler

import (
	"context"
	"net/http"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/rbac"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/api/internal/environment"
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

	rows, err := db.Query.ListEnvironmentsByApp(ctx, h.DB.RO(), app.ID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environments."),
		)
	}

	runtimeRows, err := db.Query.ListAppRuntimeSettingsByApp(ctx, h.DB.RO(), app.ID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environments."),
		)
	}
	runtimeByEnv := make(map[string]db.AppRuntimeSetting, len(runtimeRows))
	for _, r := range runtimeRows {
		runtimeByEnv[r.AppRuntimeSetting.EnvironmentID] = r.AppRuntimeSetting
	}

	buildRows, err := db.Query.ListAppBuildSettingsByApp(ctx, h.DB.RO(), app.ID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environments."),
		)
	}
	buildByEnv := make(map[string]db.AppBuildSetting, len(buildRows))
	for _, b := range buildRows {
		buildByEnv[b.EnvironmentID] = b
	}

	regionalRows, err := db.Query.ListAppRegionalSettingsByApp(ctx, h.DB.RO(), app.ID)
	if err != nil {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environments."),
		)
	}
	regionsByEnv := make(map[string][]openapi.EnvironmentRegion, len(regionalRows))
	for _, r := range regionalRows {
		regionsByEnv[r.EnvironmentID] = append(regionsByEnv[r.EnvironmentID], environment.Region(r.RegionName, r.Replicas, r.AutoscalingReplicasMin, r.AutoscalingReplicasMax))
	}

	data := make([]openapi.Environment, len(rows))
	for i, row := range rows {
		var runtimeSettings *db.AppRuntimeSetting
		if rs, ok := runtimeByEnv[row.ID]; ok {
			runtimeSettings = &rs
		}

		var buildSettings *db.AppBuildSetting
		if bs, ok := buildByEnv[row.ID]; ok {
			buildSettings = &bs
		}

		data[i] = environment.ToResponse(environment.Params{
			Env:     row,
			Runtime: runtimeSettings,
			Build:   buildSettings,
			Regions: regionsByEnv[row.ID],
		})
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
