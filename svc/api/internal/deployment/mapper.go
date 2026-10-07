package deployment

import (
	"strings"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/deploy/deployactor"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

type Input struct {
	Deployment db.ListDeploymentsRow
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
			IntervalSeconds:     new(hc.IntervalSeconds),
			TimeoutSeconds:      new(hc.TimeoutSeconds),
			FailureThreshold:    new(hc.FailureThreshold),
			InitialDelaySeconds: new(hc.InitialDelaySeconds),
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
		Trigger: openapi.DeploymentTrigger{
			Via:   triggerVia(d.Trigger),
			Actor: triggerActor(d.Trigger, d.TriggeredBy.String),
		},
		FinishedAt: finishedAt(d.Status, in.Steps),
		CreatedAt:  d.CreatedAt,
		UpdatedAt:  d.UpdatedAt.Int64,

		Git:     nil,
		Docker:  nil,
		Error:   nil,
		Domains: nil,
	}

	switch d.Source {
	case db.DeploymentsSourceGit:
		if d.GitCommitSha.Valid && d.GitCommitSha.String != "" {
			git := openapi.DeploymentGit{
				CommitSha:              d.GitCommitSha.String,
				Branch:                 nil,
				CommitMessage:          nil,
				CommitTimestamp:        nil,
				AuthorHandle:           nil,
				AuthorAvatarUrl:        nil,
				PrNumber:               nil,
				ForkRepositoryFullName: nil,
			}
			if d.GitBranch.Valid && d.GitBranch.String != "" {
				git.Branch = new(d.GitBranch.String)
			}
			if d.GitCommitMessage.Valid && d.GitCommitMessage.String != "" {
				git.CommitMessage = new(d.GitCommitMessage.String)
			}
			if d.GitCommitTimestamp.Valid && d.GitCommitTimestamp.Int64 > 0 {
				git.CommitTimestamp = new(d.GitCommitTimestamp.Int64)
			}
			if d.GitCommitAuthorHandle.Valid && d.GitCommitAuthorHandle.String != "" {
				git.AuthorHandle = new(d.GitCommitAuthorHandle.String)
			}
			if d.GitCommitAuthorAvatarUrl.Valid && d.GitCommitAuthorAvatarUrl.String != "" {
				git.AuthorAvatarUrl = new(d.GitCommitAuthorAvatarUrl.String)
			}
			if d.PrNumber.Valid && d.PrNumber.Int64 > 0 {
				git.PrNumber = new(int(d.PrNumber.Int64))
			}
			if d.ForkRepositoryFullName.Valid && d.ForkRepositoryFullName.String != "" {
				git.ForkRepositoryFullName = new(d.ForkRepositoryFullName.String)
			}
			dep.Git = &git
		}
	case db.DeploymentsSourceOci:
		image := d.ImageRequested
		if !image.Valid || image.String == "" {
			image = d.ImageResolved
		}
		if image.Valid && image.String != "" {
			docker := openapi.DeploymentDocker{Image: image.String, ResolvedImage: nil}
			if d.ImageResolved.Valid && d.ImageResolved.String != "" {
				docker.ResolvedImage = new(d.ImageResolved.String)
			}
			dep.Docker = &docker
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
	dep.Domains = new(domains)

	return dep
}

func triggerVia(trigger db.DeploymentsTrigger) openapi.DeploymentTriggerVia {
	switch trigger {
	case db.DeploymentsTriggerGithub:
		return openapi.DeploymentTriggerViaGithub
	case db.DeploymentsTriggerApi:
		return openapi.DeploymentTriggerViaApi
	case db.DeploymentsTriggerCli:
		return openapi.DeploymentTriggerViaCli
	case db.DeploymentsTriggerDashboard:
		return openapi.DeploymentTriggerViaDashboard
	case db.DeploymentsTriggerUnkey:
		return openapi.DeploymentTriggerViaUnkey
	case db.DeploymentsTriggerUnknown:
		return openapi.DeploymentTriggerViaUnknown
	}
	return openapi.DeploymentTriggerViaUnknown
}

// The api, cli, and dashboard triggers come from a client header, so the id
// format decides the type. A GitHub login cannot contain the underscore that key
// and user ids carry
func triggerActor(trigger db.DeploymentsTrigger, triggeredBy string) *openapi.DeploymentTriggerActor {
	if triggeredBy == "" {
		return nil
	}

	actorType := openapi.DeploymentTriggerActorTypeUnknown
	switch {
	case triggeredBy == deployactor.OpsID:
		actorType = openapi.DeploymentTriggerActorTypeSystem
	case strings.HasPrefix(triggeredBy, string(uid.KeyPrefix)+"_"):
		actorType = openapi.DeploymentTriggerActorTypeRootKey
	case strings.HasPrefix(triggeredBy, "user_"):
		actorType = openapi.DeploymentTriggerActorTypeUser
	case trigger == db.DeploymentsTriggerGithub:
		actorType = openapi.DeploymentTriggerActorTypeGithub
	}

	return &openapi.DeploymentTriggerActor{Type: actorType, Id: triggeredBy}
}

// Steps run in separate transactions, so a running pipeline can have every
// recorded step ended. A finalizing step marks a finished pipeline and keeps the
// value while a wake sets the status back to deploying
func finishedAt(status mysqltype.DeploymentsStatus, steps []db.DeploymentStep) *int64 {
	var latest int64
	finalized := status.IsTerminal()
	for _, step := range steps {
		if !step.EndedAt.Valid {
			return nil
		}
		latest = max(latest, step.EndedAt.Int64)
		finalized = finalized || step.Step == db.DeploymentStepsStepFinalizing
	}
	if !finalized || latest == 0 {
		return nil
	}
	return new(latest)
}
