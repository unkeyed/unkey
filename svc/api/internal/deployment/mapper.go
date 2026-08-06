// Package deployment maps a stored deployment row onto the openapi.Deployment
// wire type shared by the deployment read endpoints (getDeployment,
// listDeployments).
package deployment

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// Input is everything ToResponse needs. State carries the env/app columns the
// wire type needs but the deployments row does not hold (slugs and the app live
// pointer); both read handlers resolve them (per-deployment on get, batched on
// list) so the mapper never queries.
type Input struct {
	Deployment db.Deployment
	State      db.ListDeploymentEnvAndAppStateRow
	Steps      []db.DeploymentStep
	Regions    []string
	Domains    []string
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

	isCurrent := in.State.AppCurrentDeploymentID.String != "" && in.State.AppCurrentDeploymentID.String == d.ID

	// regions is a required field, so it must marshal as [] not null when the
	// deployment has no scheduled regions yet.
	regions := in.Regions
	if regions == nil {
		regions = []string{}
	}

	dep := openapi.Deployment{
		Id:               d.ID,
		Status:           openapi.DeploymentStatus(d.Status),
		IsCurrent:        isCurrent,
		Environment:      in.State.EnvironmentSlug,
		App:              in.State.AppSlug,
		Project:          in.State.ProjectSlug,
		AvailableActions: availableActions(in),
		Regions:          regions,
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

		Git:     nil,
		Docker:  nil,
		Error:   nil,
		Domains: nil,
	}

	setGitSource := func() {
		git := openapi.DeploymentGit{CommitSha: d.GitCommitSha.String, Branch: nil}
		if d.GitBranch.Valid && d.GitBranch.String != "" {
			git.Branch = ptr.P(d.GitBranch.String)
		}
		dep.Git = &git
	}
	setDockerSource := func() {
		image := d.RequestedImage
		if !image.Valid || image.String == "" {
			image = d.Image
		}
		if image.Valid && image.String != "" {
			dep.Docker = &openapi.DeploymentDocker{Image: image.String}
		}
	}

	switch d.Source {
	case db.DeploymentsSourceGitBuild:
		if d.GitCommitSha.Valid && d.GitCommitSha.String != "" {
			setGitSource()
		}
	case db.DeploymentsSourceDockerImage:
		setDockerSource()
	case db.DeploymentsSourceUnknown:
		// Historical rows predate explicit provenance. Preserve the previous
		// git-SHA discriminator until they can be classified safely.
		if d.GitCommitSha.Valid && d.GitCommitSha.String != "" {
			setGitSource()
		} else {
			setDockerSource()
		}
	default:
		// Future source variants remain neutral until mapped explicitly.
	}

	if failure := deriveError(d.Status, in.Steps); failure != nil {
		dep.Error = failure
	}

	domains := in.Domains
	if domains == nil {
		domains = []string{}
	}
	dep.Domains = ptr.P(domains)

	return dep
}
