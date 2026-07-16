// Package deployment maps a stored deployment row onto the openapi.Deployment
// wire type shared by the deployment read endpoints (getDeployment,
// listDeployments).
package deployment

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Input is everything ToResponse needs. EnvironmentSlug and the app live-pointer
// fields are columns the wire type needs but the deployments row does not hold;
// both read handlers resolve them (per-deployment on get, batched on list) so
// the mapper never queries.
//
// Detailed gates the two get-only fields (failure, domains): getDeployment sets
// it with Steps and Domains loaded; listDeployments leaves it false so those
// fields are omitted. Without this flag a failed deployment in a list would
// report a bogus `unknown` failure just because its steps were never loaded.
type Input struct {
	Deployment             db.Deployment
	EnvironmentSlug        string
	AppCurrentDeploymentID string
	AppIsRolledBack        bool
	Detailed               bool
	Steps                  []db.DeploymentStep
	Domains                []db.ListDeploymentDomainsRow
}

func ToResponse(in Input) openapi.Deployment {
	d := in.Deployment

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

	// isCurrent means "the app currently routes traffic to this deployment": the
	// app's current pointer, regardless of how it got there (a rolled-back app
	// still serves its current deployment).
	isCurrent := in.AppCurrentDeploymentID != "" && in.AppCurrentDeploymentID == d.ID

	dep := openapi.Deployment{
		Id:            d.ID,
		Status:        openapi.DeploymentStatus(d.Status),
		DesiredState:  openapi.DeploymentDesiredState(d.DesiredState),
		IsCurrent:     isCurrent,
		EnvironmentId: d.EnvironmentID,
		AppId:         d.AppID,
		ProjectId:     d.ProjectID,
		AvailableActions: availableActions(
			d.Status,
			d.DesiredState,
			in.EnvironmentSlug,
			d.ID,
			in.AppCurrentDeploymentID,
			in.AppIsRolledBack,
		),
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

	// A deployment is sourced from either git or a prebuilt image. git_commit_sha
	// is the discriminator: git builds set it (and also fill image with the built
	// output), image deploys leave it null.
	switch {
	case d.GitCommitSha.Valid && d.GitCommitSha.String != "":
		git := openapi.DeploymentGit{CommitSha: d.GitCommitSha.String}
		if d.GitBranch.Valid && d.GitBranch.String != "" {
			git.Branch = ptr.P(d.GitBranch.String)
		}
		dep.Git = &git
	case d.Image.Valid && d.Image.String != "":
		dep.Docker = &openapi.DeploymentDocker{Image: d.Image.String}
	}

	// failure and domains are get-only. Deriving them requires the per-deployment
	// step and route queries that listDeployments skips, so gate on Detailed
	// rather than on whether the slices happen to be nil.
	if in.Detailed {
		if failure := deriveFailure(d.Status, in.Steps); failure != nil {
			dep.Failure = failure
		}
		dep.Domains = ptr.P(mapDomains(in.Domains))
	}

	return dep
}
