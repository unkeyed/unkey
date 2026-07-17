// Package deployment maps a stored deployment row onto the openapi.Deployment
// wire type shared by the deployment read endpoints (getDeployment,
// listDeployments).
package deployment

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func ToResponse(d db.Deployment) openapi.Deployment {
	command := []string(d.Command)
	if command == nil {
		command = []string{}
	}

	var healthcheck *openapi.EnvironmentHealthcheck
	if hc := d.Healthcheck.Healthcheck; hc != nil {
		healthcheck = &openapi.EnvironmentHealthcheck{
			Method:              openapi.EnvironmentHealthcheckMethod(hc.Method),
			Path:                hc.Path,
			IntervalSeconds:     ptr.P(hc.IntervalSeconds),
			TimeoutSeconds:      ptr.P(hc.TimeoutSeconds),
			FailureThreshold:    ptr.P(hc.FailureThreshold),
			InitialDelaySeconds: ptr.P(hc.InitialDelaySeconds),
		}
	}

	return openapi.Deployment{
		Id:     d.ID,
		Status: openapi.DeploymentStatus(d.Status),
		Runtime: openapi.DeploymentRuntime{
			VCpus:            float64(d.CpuMillicores) / 1000,
			MemoryMib:        int(d.MemoryMib),
			StorageMib:       int(d.StorageMib),
			Port:             int(d.Port),
			Command:          command,
			ShutdownSignal:   openapi.EnvironmentShutdownSignal(d.ShutdownSignal),
			UpstreamProtocol: openapi.EnvironmentUpstreamProtocol(d.UpstreamProtocol),
			Healthcheck:      healthcheck,
		},
		CreatedAt: d.CreatedAt,
		UpdatedAt: d.UpdatedAt.Int64,
	}
}
