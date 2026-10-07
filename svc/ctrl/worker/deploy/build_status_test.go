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
	buildSteps, closeAndCollect := collectRows[schema.BuildStepV1](t)
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

	w.processBuildStatus(statusCh, uid.New(uid.WorkspacePrefix), uid.New(uid.ProjectPrefix), uid.New(uid.DeploymentPrefix), newLogSequence())
	rows := closeAndCollect()

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

func TestProcessBuildStatusWritesLogRows(t *testing.T) {
	buildStepLogs, closeAndCollect := collectRows[schema.BuildStepLogV1](t)
	w := &Workflow{buildSteps: batch.NewNoop[schema.BuildStepV1](), buildStepLogs: buildStepLogs}

	workspaceID := uid.New(uid.WorkspacePrefix)
	projectID := uid.New(uid.ProjectPrefix)
	deploymentID := uid.New(uid.DeploymentPrefix)
	vertex := digest.FromString(uid.New("vertex"))
	now := time.Now()

	solve := func(seq *logSequence, logs ...*client.VertexLog) {
		statusCh := make(chan *client.SolveStatus, len(logs))
		for _, log := range logs {
			statusCh <- &client.SolveStatus{Logs: []*client.VertexLog{log}}
		}
		close(statusCh)
		w.processBuildStatus(statusCh, workspaceID, projectID, deploymentID, seq)
	}

	seq := newLogSequence()
	solve(seq,
		&client.VertexLog{Vertex: vertex, Stream: 1, Data: []byte("KEBAP stdout"), Timestamp: now},
		&client.VertexLog{Vertex: vertex, Stream: buildkitStderrStream, Data: []byte("KEBAP stderr"), Timestamp: now},
	)
	solve(seq,
		&client.VertexLog{Vertex: vertex, Stream: 1, Data: []byte("KEBAP second solve"), Timestamp: now},
	)
	rows := closeAndCollect()

	type logRow struct {
		message string
		stderr  bool
	}
	got := make([]logRow, 0, len(rows))
	for i, row := range rows {
		got = append(got, logRow{message: row.Message, stderr: row.Stderr})
		if i > 0 {
			require.Greater(t, row.Seq, rows[i-1].Seq, "seq must strictly increase within a solve and across solves that share a counter")
		}
	}
	require.Equal(t, []logRow{
		{message: "KEBAP stdout", stderr: false},
		{message: "KEBAP stderr", stderr: true},
		{message: "KEBAP second solve", stderr: false},
	}, got)
}

func TestProcessBuildStatusWritesStepEventLines(t *testing.T) {
	buildStepLogs, closeAndCollect := collectRows[schema.BuildStepLogV1](t)
	w := &Workflow{buildSteps: batch.NewNoop[schema.BuildStepV1](), buildStepLogs: buildStepLogs}

	started := time.Now()
	completed := started.Add(5300 * time.Millisecond)
	install := digest.FromString(uid.New("vertex"))
	cached := digest.FromString(uid.New("vertex"))
	failed := digest.FromString(uid.New("vertex"))
	instant := digest.FromString(uid.New("vertex"))
	almostInstant := started.Add(40 * time.Millisecond)

	statusCh := make(chan *client.SolveStatus, 10)
	statusCh <- &client.SolveStatus{Vertexes: []*client.Vertex{
		{Digest: install, Name: "[3/4] RUN npm ci", Started: &started},
		{Digest: cached, Name: "[2/4] COPY . .", Started: &started, Completed: &started, Cached: true},
	}}
	statusCh <- &client.SolveStatus{
		Vertexes: []*client.Vertex{{Digest: install, Name: "[3/4] RUN npm ci", Started: &started, Completed: &completed}},
		Logs:     []*client.VertexLog{{Vertex: install, Stream: 1, Data: []byte("KEBAP installed"), Timestamp: completed}},
	}
	statusCh <- &client.SolveStatus{Vertexes: []*client.Vertex{
		{Digest: instant, Name: "[1/4] FROM alpine", Started: &started, Completed: &almostInstant},
	}}
	statusCh <- &client.SolveStatus{Vertexes: []*client.Vertex{
		{Digest: failed, Name: "[4/4] RUN npm test", Started: &completed, Completed: &completed, Error: "KEBAP exit code: 1"},
	}}
	close(statusCh)

	w.processBuildStatus(statusCh, uid.New(uid.WorkspacePrefix), uid.New(uid.ProjectPrefix), uid.New(uid.DeploymentPrefix), newLogSequence())
	rows := closeAndCollect()

	type logRow struct {
		stepID  string
		time    int64
		message string
		stderr  bool
	}
	got := make([]logRow, 0, len(rows))
	for i, row := range rows {
		got = append(got, logRow{stepID: row.StepID, time: row.Time, message: row.Message, stderr: row.Stderr})
		if i > 0 {
			require.Greater(t, row.Seq, rows[i-1].Seq, "event lines take seq from the same counter as log rows")
		}
	}
	require.Equal(t, []logRow{
		{stepID: cached.String(), time: started.UnixMilli(), message: "CACHED", stderr: false},
		{stepID: install.String(), time: completed.UnixMilli(), message: "KEBAP installed", stderr: false},
		{stepID: install.String(), time: completed.UnixMilli(), message: "DONE 5.3s", stderr: false},
		{stepID: failed.String(), time: completed.UnixMilli(), message: "ERROR: KEBAP exit code: 1", stderr: true},
	}, got, "a start and a step that rounds to 0.0s add no line, and a completion line follows the output of its status")
}

func collectRows[T any](t *testing.T) (*batch.BatchProcessor[T], func() []T) {
	t.Helper()
	var mu sync.Mutex
	var rows []T
	processor := batch.New(batch.Config[T]{
		Name:          "collect_rows_test",
		BatchSize:     100,
		BufferSize:    100,
		FlushInterval: time.Hour,
		Consumers:     1,
		Drop:          false,
		Flush: func(_ context.Context, batch []T) {
			mu.Lock()
			defer mu.Unlock()
			rows = append(rows, batch...)
		},
	})
	return processor, func() []T {
		processor.Close()
		mu.Lock()
		defer mu.Unlock()
		return rows
	}
}
