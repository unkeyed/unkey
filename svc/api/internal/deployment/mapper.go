package deployment

import (
	"context"
	"database/sql"
	"strings"

	mysqltype "github.com/unkeyed/unkey/pkg/mysql/types"

	"github.com/unkeyed/unkey/pkg/conc"
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

func ToResponse(ctx context.Context, database *db.Replica, workspaceID string, row db.ListDeploymentsRow) (openapi.Deployment, error) {
	responses, err := ToResponses(ctx, database, workspaceID, []db.ListDeploymentsRow{row})
	if err != nil {
		return openapi.Deployment{}, err //nolint:exhaustruct // no response on error
	}
	return responses[0], nil
}

// ToResponses loads the state, steps, regions, and domains of rows in four
// parallel queries, whatever the number of rows, and maps each row to its response
func ToResponses(ctx context.Context, database *db.Replica, workspaceID string, rows []db.ListDeploymentsRow) ([]openapi.Deployment, error) {
	if len(rows) == 0 {
		return []openapi.Deployment{}, nil
	}
	ids := make([]string, len(rows))
	for i, row := range rows {
		ids[i] = row.ID
	}

	var states []db.ListDeploymentEnvAndAppStateRow
	var steps []db.DeploymentStep
	var regions []db.ListDeploymentRegionsByIdsRow
	var domains []db.ListDeploymentDomainsByIdsRow
	err := conc.All(ctx,
		func(ctx context.Context) (err error) {
			states, err = db.Query.ListDeploymentEnvAndAppState(ctx, database, db.ListDeploymentEnvAndAppStateParams{WorkspaceID: workspaceID, DeploymentIds: ids})
			return err
		},
		func(ctx context.Context) (err error) {
			steps, err = db.Query.ListDeploymentStepsByIds(ctx, database, db.ListDeploymentStepsByIdsParams{WorkspaceID: workspaceID, DeploymentIds: ids})
			return err
		},
		func(ctx context.Context) (err error) {
			regions, err = db.Query.ListDeploymentRegionsByIds(ctx, database, db.ListDeploymentRegionsByIdsParams{WorkspaceID: workspaceID, DeploymentIds: ids})
			return err
		},
		func(ctx context.Context) (err error) {
			domains, err = db.Query.ListDeploymentDomainsByIds(ctx, database, db.ListDeploymentDomainsByIdsParams{WorkspaceID: workspaceID, DeploymentIds: ids})
			return err
		},
	)
	if err != nil {
		return nil, err
	}

	inputs := make(map[string]*Input, len(rows))
	for _, row := range rows {
		inputs[row.ID] = &Input{Deployment: row, State: db.ListDeploymentEnvAndAppStateRow{}, Steps: nil, Regions: nil, Domains: nil} //nolint:exhaustruct // zero state when the join misses
	}
	for _, s := range states {
		inputs[s.DeploymentID].State = s
	}
	for _, s := range steps {
		inputs[s.DeploymentID].Steps = append(inputs[s.DeploymentID].Steps, s)
	}
	for _, r := range regions {
		inputs[r.DeploymentID].Regions = append(inputs[r.DeploymentID].Regions, r.Region)
	}
	for _, d := range domains {
		inputs[d.DeploymentID].Domains = append(inputs[d.DeploymentID].Domains, d.Domain)
	}

	responses := make([]openapi.Deployment, len(rows))
	for i, row := range rows {
		responses[i] = toResponse(*inputs[row.ID])
	}
	return responses, nil
}

func toResponse(in Input) openapi.Deployment {
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
				CommitSha:       d.GitCommitSha.String,
				Branch:          optionalString(d.GitBranch),
				CommitMessage:   optionalString(d.GitCommitMessage),
				CommitTimestamp: nil,
				Author:          nil,
				PrNumber:        nil,
				ForkRepository:  optionalString(d.ForkRepositoryFullName),
			}
			if d.GitCommitAuthorHandle.Valid && d.GitCommitAuthorHandle.String != "" {
				git.Author = &openapi.DeploymentGitAuthor{
					Handle:    d.GitCommitAuthorHandle.String,
					AvatarUrl: optionalString(d.GitCommitAuthorAvatarUrl),
				}
			}
			if d.GitCommitTimestamp.Valid && d.GitCommitTimestamp.Int64 > 0 {
				git.CommitTimestamp = new(d.GitCommitTimestamp.Int64)
			}
			if d.PrNumber.Valid && d.PrNumber.Int64 > 0 {
				git.PrNumber = new(int(d.PrNumber.Int64))
			}
			dep.Git = &git
		}
	case db.DeploymentsSourceOci:
		image := d.ImageRequested
		if !image.Valid || image.String == "" {
			image = d.ImageResolved
		}
		if image.Valid && image.String != "" {
			dep.Docker = &openapi.DeploymentDocker{Image: image.String, ResolvedImage: optionalString(d.ImageResolved)}
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

func optionalString(v sql.NullString) *string {
	if !v.Valid || v.String == "" {
		return nil
	}
	return new(v.String)
}
