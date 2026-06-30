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
		minReplicas := int(r.Replicas)
		maxReplicas := int(r.Replicas)
		if r.AutoscalingReplicasMin.Valid {
			minReplicas = int(r.AutoscalingReplicasMin.Int32)
		}
		if r.AutoscalingReplicasMax.Valid {
			maxReplicas = int(r.AutoscalingReplicasMax.Int32)
		}
		regionsByEnv[r.EnvironmentID] = append(regionsByEnv[r.EnvironmentID], openapi.EnvironmentRegion{
			Name: r.RegionName,
			Replicas: openapi.ReplicaBounds{
				Min: minReplicas,
				Max: maxReplicas,
			},
		})
	}

	data := make([]openapi.Environment, len(rows))
	for i, row := range rows {
		env := openapi.Environment{
			Id:               row.ID,
			ProjectId:        row.ProjectID,
			AppId:            row.AppID,
			Slug:             row.Slug,
			Description:      row.Description,
			DeleteProtection: row.DeleteProtection.Bool,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt.Int64,
			Port:             nil,
			CpuMillicores:    nil,
			MemoryMib:        nil,
			StorageMib:       nil,
			Command:          nil,
			Healthcheck:      nil,
			ShutdownSignal:   nil,
			UpstreamProtocol: nil,
			OpenapiSpecPath:  nil,
			Dockerfile:       nil,
			RootDirectory:    nil,
			BuildCommand:     nil,
			WatchPaths:       nil,
			AutoDeploy:       nil,
			Regions:          nil,
		}

		if rs, ok := runtimeByEnv[row.ID]; ok {
			env.Port = ptr.P(int(rs.Port))
			env.CpuMillicores = ptr.P(int(rs.CpuMillicores))
			env.MemoryMib = ptr.P(int(rs.MemoryMib))
			env.StorageMib = ptr.P(int(rs.StorageMib))
			env.Command = ptr.P([]string(rs.Command))
			env.ShutdownSignal = ptr.P(openapi.EnvironmentShutdownSignal(rs.ShutdownSignal))
			env.UpstreamProtocol = ptr.P(openapi.EnvironmentUpstreamProtocol(rs.UpstreamProtocol))
			if rs.OpenapiSpecPath.Valid {
				env.OpenapiSpecPath = ptr.P(rs.OpenapiSpecPath.String)
			}
			if hc := rs.Healthcheck.Healthcheck; hc != nil {
				env.Healthcheck = &openapi.Healthcheck{
					Method:              openapi.HealthcheckMethod(hc.Method),
					Path:                hc.Path,
					IntervalSeconds:     ptr.P(hc.IntervalSeconds),
					TimeoutSeconds:      ptr.P(hc.TimeoutSeconds),
					FailureThreshold:    ptr.P(hc.FailureThreshold),
					InitialDelaySeconds: ptr.P(hc.InitialDelaySeconds),
				}
			}
		}

		if bs, ok := buildByEnv[row.ID]; ok {
			if bs.Dockerfile.Valid {
				env.Dockerfile = ptr.P(bs.Dockerfile.String)
			}
			env.RootDirectory = ptr.P(bs.DockerContext)
			if bs.BuildCommand.Valid {
				env.BuildCommand = ptr.P(bs.BuildCommand.String)
			}
			env.WatchPaths = ptr.P([]string(bs.WatchPaths))
			env.AutoDeploy = ptr.P(bs.AutoDeploy)
		}

		if regions := regionsByEnv[row.ID]; len(regions) > 0 {
			env.Regions = ptr.P(regions)
		}

		data[i] = env
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
