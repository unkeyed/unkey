package deployment

import (
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

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

	switch d.Source {
	case db.DeploymentsSourceGit:
		if d.GitCommitSha.Valid && d.GitCommitSha.String != "" {
			git := openapi.DeploymentGit{CommitSha: d.GitCommitSha.String, Branch: nil}
			if d.GitBranch.Valid && d.GitBranch.String != "" {
				git.Branch = ptr.P(d.GitBranch.String)
			}
			dep.Git = &git
		}
	case db.DeploymentsSourceOci:
		image := d.ImageRequested
		if !image.Valid || image.String == "" {
			image = d.ImageResolved
		}
		if !image.Valid || image.String == "" {
			image = d.Image
		}
		if image.Valid && image.String != "" {
			dep.Docker = &openapi.DeploymentDocker{Image: image.String}
		}
	case db.DeploymentsSourceUnknown:
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
