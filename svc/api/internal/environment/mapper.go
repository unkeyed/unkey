// Package environment maps stored app settings onto the openapi.Environment
// wire type shared by the environment read endpoints.
package environment

import (
	"database/sql"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Params holds the stored rows for a single environment. Runtime and Build are
// nil until the environment has been deployed and gained those settings;
// Regions is empty until regional settings are configured.
type Params struct {
	Env     db.Environment
	Runtime *db.AppRuntimeSetting
	Build   *db.AppBuildSetting
	Regions []openapi.EnvironmentRegion
}

// ToResponse builds the wire representation of an environment. Settings that
// are nil or empty leave their fields omitted.
func ToResponse(p Params) openapi.Environment {
	env := openapi.Environment{
		Id:               p.Env.ID,
		ProjectId:        p.Env.ProjectID,
		AppId:            p.Env.AppID,
		Slug:             p.Env.Slug,
		Description:      p.Env.Description,
		DeleteProtection: p.Env.DeleteProtection.Bool,
		CreatedAt:        p.Env.CreatedAt,
		UpdatedAt:        p.Env.UpdatedAt.Int64,
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

	if rs := p.Runtime; rs != nil {
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
			env.Healthcheck = &openapi.EnvironmentHealthcheck{
				Method:              openapi.EnvironmentHealthcheckMethod(hc.Method),
				Path:                hc.Path,
				IntervalSeconds:     ptr.P(hc.IntervalSeconds),
				TimeoutSeconds:      ptr.P(hc.TimeoutSeconds),
				FailureThreshold:    ptr.P(hc.FailureThreshold),
				InitialDelaySeconds: ptr.P(hc.InitialDelaySeconds),
			}
		}
	}

	if bs := p.Build; bs != nil {
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

	if len(p.Regions) > 0 {
		env.Regions = ptr.P(p.Regions)
	}

	return env
}

// Region builds the wire representation of a single deployment region.
func Region(name string, replicas int32, min, max sql.NullInt32) openapi.EnvironmentRegion {
	return openapi.EnvironmentRegion{
		Name:     name,
		Replicas: effectiveReplicaBounds(replicas, min, max),
	}
}

// effectiveReplicaBounds returns the min and max replicas for a region, preferring
// the attached autoscaling policy and falling back to the static replica count.
func effectiveReplicaBounds(replicas int32, min, max sql.NullInt32) openapi.Replicas {
	minReplicas := int(replicas)
	maxReplicas := int(replicas)
	if min.Valid {
		minReplicas = int(min.Int32)
	}
	if max.Valid {
		maxReplicas = int(max.Int32)
	}
	return openapi.Replicas{
		Min: minReplicas,
		Max: maxReplicas,
	}
}
