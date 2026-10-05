package deploy

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/moby/buildkit/client"
	"github.com/opencontainers/go-digest"
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestProcessBuildStatusWritesStepRows(t *testing.T) {
	var mu sync.Mutex
	var rows []schema.BuildStepV1
	buildSteps := batch.New(batch.Config[schema.BuildStepV1]{
		Name:          "build_steps_test",
		BatchSize:     100,
		BufferSize:    100,
		FlushInterval: time.Hour,
		Consumers:     1,
		Drop:          false,
		Flush: func(_ context.Context, batch []schema.BuildStepV1) {
			mu.Lock()
			defer mu.Unlock()
			rows = append(rows, batch...)
		},
	})
	w := &Workflow{buildSteps: buildSteps, buildStepLogs: batch.NewNoop[schema.BuildStepLogV1]()}

	started := time.Now()
	completed := started.Add(12 * time.Second)
	install := digest.FromString(uid.New("vertex"))
	cached := digest.FromString(uid.New("vertex"))

	statusCh := make(chan *client.SolveStatus, 10)
	statusCh <- &client.SolveStatus{Vertexes: []*client.Vertex{
		{Digest: install, Name: "[3/4] RUN npm ci", Started: &started},
		{Digest: cached, Name: "[2/4] COPY . .", Started: &started, Completed: &started, Cached: true},
	}}
	statusCh <- &client.SolveStatus{Vertexes: []*client.Vertex{
		{Digest: install, Name: "[3/4] RUN npm ci", Started: &started},
	}}
	statusCh <- &client.SolveStatus{Vertexes: []*client.Vertex{
		{Digest: install, Name: "[3/4] RUN npm ci", Started: &started, Completed: &completed},
	}}
	statusCh <- &client.SolveStatus{Vertexes: []*client.Vertex{
		{Digest: install, Name: "[3/4] RUN npm ci", Started: &completed},
	}}
	close(statusCh)

	w.processBuildStatus(statusCh, uid.New(uid.WorkspacePrefix), uid.New(uid.ProjectPrefix), uid.New(uid.DeploymentPrefix))
	buildSteps.Close()

	type stepRow struct {
		stepID      string
		startedAt   int64
		completedAt int64
		cached      bool
	}
	got := make([]stepRow, 0, len(rows))
	for _, row := range rows {
		got = append(got, stepRow{stepID: row.StepID, startedAt: row.StartedAt, completedAt: row.CompletedAt, cached: row.Cached})
	}
	require.Equal(t, []stepRow{
		{stepID: install.String(), startedAt: started.UnixMilli(), completedAt: 0},
		{stepID: cached.String(), startedAt: started.UnixMilli(), completedAt: started.UnixMilli(), cached: true},
		{stepID: install.String(), startedAt: started.UnixMilli(), completedAt: completed.UnixMilli()},
	}, got, "a running step gets one start row, a cached step only its completed row, and a later start report after completion adds nothing")
}
