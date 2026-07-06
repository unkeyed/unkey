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

	environment, err := db.Query.FindEnvironmentByIdentifiers(ctx, h.DB.RO(), db.FindEnvironmentByIdentifiersParams{
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

	data := openapi.Environment{
		Id:               environment.ID,
		Slug:             environment.Slug,
		Description:      environment.Description,
		DeleteProtection: environment.DeleteProtection.Bool,
		CreatedAt:        environment.CreatedAt,
		UpdatedAt:        environment.UpdatedAt.Int64,
		Runtime:          nil,
		Build:            nil,
		Regions:          nil,
	}

	// Settings rows are created at deploy time, so an environment may exist
	// before any of them do. A missing row leaves those fields omitted.
	runtime, err := db.Query.FindAppRuntimeSettingsByAppAndEnv(ctx, h.DB.RO(), db.FindAppRuntimeSettingsByAppAndEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}
	if !db.IsNotFound(err) {
		rs := runtime.AppRuntimeSetting
		rt := openapi.EnvironmentRuntime{
			Port:             int(rs.Port),
			CpuMillicores:    int(rs.CpuMillicores),
			MemoryMib:        int(rs.MemoryMib),
			StorageMib:       int(rs.StorageMib),
			Command:          []string(rs.Command),
			ShutdownSignal:   openapi.EnvironmentRuntimeShutdownSignal(rs.ShutdownSignal),
			UpstreamProtocol: openapi.EnvironmentRuntimeUpstreamProtocol(rs.UpstreamProtocol),
			Healthcheck:      nil,
			OpenapiSpecPath:  nil,
		}
		if rs.OpenapiSpecPath.Valid {
			rt.OpenapiSpecPath = ptr.P(rs.OpenapiSpecPath.String)
		}
		if hc := rs.Healthcheck.Healthcheck; hc != nil {
			rt.Healthcheck = &openapi.EnvironmentHealthcheck{
				Method:              openapi.EnvironmentHealthcheckMethod(hc.Method),
				Path:                hc.Path,
				IntervalSeconds:     ptr.P(hc.IntervalSeconds),
				TimeoutSeconds:      ptr.P(hc.TimeoutSeconds),
				FailureThreshold:    ptr.P(hc.FailureThreshold),
				InitialDelaySeconds: ptr.P(hc.InitialDelaySeconds),
			}
		}
		data.Runtime = &rt
	}

	build, err := db.Query.FindAppBuildSettingByAppEnv(ctx, h.DB.RO(), db.FindAppBuildSettingByAppEnvParams{
		AppID:         environment.AppID,
		EnvironmentID: environment.ID,
	})
	if err != nil && !db.IsNotFound(err) {
		return fault.Wrap(
			err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("database error"),
			fault.Public("Failed to retrieve environment."),
		)
	}
	if !db.IsNotFound(err) {
		bs := openapi.EnvironmentBuild{
			RootDirectory: build.DockerContext,
			WatchPaths:    []string(build.WatchPaths),
			AutoDeploy:    build.AutoDeploy,
			Dockerfile:    nil,
			BuildCommand:  nil,
		}
		if build.Dockerfile.Valid {
			bs.Dockerfile = ptr.P(build.Dockerfile.String)
		}
		if build.BuildCommand.Valid {
			bs.BuildCommand = ptr.P(build.BuildCommand.String)
		}
		data.Build = &bs
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
	if len(regional) > 0 {
		regions := make([]openapi.EnvironmentRegion, 0, len(regional))
		for _, r := range regional {
			minReplicas := int(r.Replicas)
			maxReplicas := int(r.Replicas)
			if r.AutoscalingReplicasMin.Valid {
				minReplicas = int(r.AutoscalingReplicasMin.Int32)
			}
			if r.AutoscalingReplicasMax.Valid {
				maxReplicas = int(r.AutoscalingReplicasMax.Int32)
			}
			regions = append(regions, openapi.EnvironmentRegion{
				Name: r.RegionName,
				Replicas: openapi.Replicas{
					Min: minReplicas,
					Max: maxReplicas,
				},
			})
		}
		data.Regions = ptr.P(regions)
	}

	return s.JSON(http.StatusOK, Response{
		Meta: openapi.Meta{
			RequestId: s.RequestID(),
		},
		Data: data,
	})
}
