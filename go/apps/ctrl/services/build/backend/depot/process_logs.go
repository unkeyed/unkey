package depot

import (
	"github.com/moby/buildkit/client"
	"github.com/opencontainers/go-digest"
	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
)

func (s *Depot) processBuildStatus(
	statusCh <-chan *client.SolveStatus,
	workspaceID, projectID, deploymentID string,
) {
	aggregator := NewBuildStepAggregator()
	for status := range statusCh {
		aggregator.ProcessStatus(
			status,
			workspaceID,
			projectID,
			deploymentID,
			s.clickhouse.BufferBuildStep,
			s.clickhouse.BufferBuildStepLog,
		)
	}
}

type BuildStepAggregator struct {
	completed        map[digest.Digest]bool
	verticesWithLogs map[digest.Digest]bool
}

func NewBuildStepAggregator() *BuildStepAggregator {
	return &BuildStepAggregator{
		completed:        make(map[digest.Digest]bool),
		verticesWithLogs: make(map[digest.Digest]bool),
	}
}

func (a *BuildStepAggregator) ProcessStatus(
	status *client.SolveStatus,
	workspaceID, projectID, deploymentID string,
	cbStep func(schema.BuildStepV1),
	cbLog func(schema.BuildStepLogV1),
) {
	for _, log := range status.Logs {
		a.verticesWithLogs[log.Vertex] = true
	}

	for _, vertex := range status.Vertexes {
		if vertex.Completed != nil && !a.completed[vertex.Digest] {
			a.completed[vertex.Digest] = true

			cbStep(schema.BuildStepV1{
				Error:        vertex.Error,
				StartedAt:    vertex.Started.UnixMilli(),
				CompletedAt:  vertex.Completed.UnixMilli(),
				WorkspaceID:  workspaceID,
				ProjectID:    projectID,
				DeploymentID: deploymentID,
				StepID:       string(vertex.Digest),
				Name:         vertex.Name,
				Cache:        vertex.Cached,
				HasLogs:      a.verticesWithLogs[vertex.Digest],
			})
		}
	}

	for _, log := range status.Logs {
		cbLog(schema.BuildStepLogV1{
			WorkspaceID:  workspaceID,
			ProjectID:    projectID,
			DeploymentID: deploymentID,
			StepID:       string(log.Vertex),
			Time:         log.Timestamp.UnixMilli(),
			Message:      string(log.Data),
		})
	}
}
