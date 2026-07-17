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
type Input struct {
	Deployment             db.Deployment
	ProjectSlug            string
	AppSlug                string
	EnvironmentSlug        string
	AppCurrentDeploymentID string
	AppIsRolledBack        bool
	Steps                  []db.DeploymentStep
	Regions                []string
	Domains                []string
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
		Environment:      in.EnvironmentSlug,
		App:              in.AppSlug,
		Project:          in.ProjectSlug,
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

		// Optional fields set below: git/docker from the source discriminator,
		// error/domains only when Detailed.
		Git:     nil,
		Docker:  nil,
		Error:   nil,
		Domains: nil,
	}

	// A deployment is sourced from either git or a prebuilt image. git_commit_sha
	// is the discriminator: git builds set it (and also fill image with the built
	// output), image deploys leave it null.
	switch {
	case d.GitCommitSha.Valid && d.GitCommitSha.String != "":
		git := openapi.DeploymentGit{CommitSha: d.GitCommitSha.String, Branch: nil}
		if d.GitBranch.Valid && d.GitBranch.String != "" {
			git.Branch = ptr.P(d.GitBranch.String)
		}
		dep.Git = &git
	case d.Image.Valid && d.Image.String != "":
		dep.Docker = &openapi.DeploymentDocker{Image: d.Image.String}
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
