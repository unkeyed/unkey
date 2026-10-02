package clickhouse

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/testutil/containers"
	"github.com/unkeyed/unkey/pkg/uid"
)

func TestGetBuildLogs(t *testing.T) {
	chCfg := containers.ClickHouse(t)
	client, err := New(Config{URL: chCfg.DSN})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, client.Close()) })
	ctx := context.Background()

	now := time.Now().UnixMilli()
	firstSeq := uint64(now) * 1000

	newTarget := func() GetBuildLogsRequest {
		return GetBuildLogsRequest{
			WorkspaceID:  uid.New(uid.WorkspacePrefix),
			ProjectID:    uid.New(uid.ProjectPrefix),
			DeploymentID: uid.New(uid.DeploymentPrefix),
			StepID:       "",
			AfterSeq:     0,
			Limit:        100,
			// Most cases read after ctrl flushed every row
			Now: time.UnixMilli(now).Add(buildLogStepRowWait + time.Millisecond),
		}
	}
	newStepID := func() string {
		return "sha256:" + uid.New("")
	}
	insertStep := func(t *testing.T, target GetBuildLogsRequest, stepID, name string, startedAt, completedAt int64) {
		t.Helper()
		require.NoError(t, client.Exec(ctx,
			"INSERT INTO default.build_steps_v1 (step_id, started_at, completed_at, workspace_id, project_id, deployment_id, name, cached, error, has_logs) VALUES (?, ?, ?, ?, ?, ?, ?, false, '', true)",
			stepID, startedAt, completedAt, target.WorkspaceID, target.ProjectID, target.DeploymentID, name,
		))
	}
	// insertLogs writes count entries with seq from fromSeq and messages
	// "KEBAP 0" to "KEBAP <count-1>"
	insertLogs := func(t *testing.T, target GetBuildLogsRequest, stepID string, fromSeq uint64, count int, stderr bool) {
		t.Helper()
		require.NoError(t, client.Exec(ctx,
			"INSERT INTO default.build_step_logs_v1 (time, workspace_id, project_id, deployment_id, step_id, message, seq, stderr) SELECT toInt64(?), ?, ?, ?, ?, concat('KEBAP ', toString(number)), toUInt64(?) + number, ? FROM numbers(?)",
			now, target.WorkspaceID, target.ProjectID, target.DeploymentID, stepID, fromSeq, stderr, count,
		))
	}
	messages := func(entries []BuildLogEntry) []string {
		out := make([]string, 0, len(entries))
		for _, entry := range entries {
			out = append(out, entry.Message)
		}
		return out
	}

	t.Run("pages through entries in seq order", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertStep(t, target, stepID, "[3/4] RUN npm ci", now, now+1)
		insertLogs(t, target, stepID, firstSeq, 5, false)

		target.Limit = 2
		var pages [][]string
		var hasMore []bool
		for range 3 {
			res, err := client.GetBuildLogs(ctx, target)
			require.NoError(t, err)
			pages = append(pages, messages(res.Entries))
			hasMore = append(hasMore, res.HasMore)
			if len(res.Entries) > 0 {
				target.AfterSeq = res.Entries[len(res.Entries)-1].Seq
			}
		}
		require.Equal(t, [][]string{{"KEBAP 0", "KEBAP 1"}, {"KEBAP 2", "KEBAP 3"}, {"KEBAP 4"}}, pages)
		require.Equal(t, []bool{true, true, false}, hasMore)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Empty(t, res.Entries, "a poll after the last entry is empty")
		require.False(t, res.HasMore)
	})

	t.Run("filters by step", func(t *testing.T) {
		target := newTarget()
		install, build := newStepID(), newStepID()
		insertLogs(t, target, install, firstSeq, 2, false)
		insertLogs(t, target, build, firstSeq+2, 3, false)

		target.StepID = build
		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Equal(t, []string{"KEBAP 0", "KEBAP 1", "KEBAP 2"}, messages(res.Entries))
		for _, entry := range res.Entries {
			require.Equal(t, build, entry.StepID)
		}

		target.StepID = newStepID()
		res, err = client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Empty(t, res.Entries, "an unknown step has no entries")
	})

	t.Run("maps the stderr flag", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertLogs(t, target, stepID, firstSeq, 1, false)
		insertLogs(t, target, stepID, firstSeq+1, 1, true)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 2)
		require.False(t, res.Entries[0].Stderr)
		require.True(t, res.Entries[1].Stderr)
	})

	t.Run("a row from before the migration uses the default seq", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertStep(t, target, stepID, "[2/4] RUN apk add git", now, now+1)
		require.NoError(t, client.Exec(ctx,
			"INSERT INTO default.build_step_logs_v1 (time, workspace_id, project_id, deployment_id, step_id, message) VALUES (?, ?, ?, ?, ?, 'KEBAP legacy')",
			now, target.WorkspaceID, target.ProjectID, target.DeploymentID, stepID,
		))

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Equal(t, []BuildLogEntry{{
			Seq:     uint64(now) * 1000,
			Time:    now,
			StepID:  stepID,
			Step:    "[2/4] RUN apk add git",
			Stderr:  false,
			Message: "KEBAP legacy",
		}}, res.Entries)
	})

	t.Run("the byte budget cuts a page and keeps hasMore", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		entryBytes := 400 * 1024
		require.NoError(t, client.Exec(ctx,
			"INSERT INTO default.build_step_logs_v1 (time, workspace_id, project_id, deployment_id, step_id, message, seq) SELECT toInt64(?), ?, ?, ?, ?, repeat('KEBA', intDiv(toUInt64(?), 4)), toUInt64(?) + number FROM numbers(4)",
			now, target.WorkspaceID, target.ProjectID, target.DeploymentID, stepID, entryBytes, firstSeq,
		))

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 3, "the fourth entry starts after 1200 KiB, past the 1 MiB budget")
		require.True(t, res.HasMore)

		target.AfterSeq = res.Entries[2].Seq
		res, err = client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 1)
		require.False(t, res.HasMore)
	})

	t.Run("a retry's entries sort after the first attempt", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		retrySeq := firstSeq + 60_000_000
		insertLogs(t, target, stepID, retrySeq, 2, false)
		insertLogs(t, target, stepID, firstSeq, 2, true)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 4)
		require.Equal(t, []bool{true, true, false, false}, []bool{res.Entries[0].Stderr, res.Entries[1].Stderr, res.Entries[2].Stderr, res.Entries[3].Stderr})
	})

	t.Run("a running step returns its name and its entries so far", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertStep(t, target, stepID, "[4/4] RUN npm run build", now, 0)
		insertLogs(t, target, stepID, firstSeq, 2, false)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 2)
		require.Equal(t, "[4/4] RUN npm run build", res.Entries[0].Step)
	})

	t.Run("the completed row and a retry win the name lookup", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertStep(t, target, stepID, "KEBAP first attempt", now, 0)
		insertStep(t, target, stepID, "KEBAP first attempt done", now, now+1)
		insertStep(t, target, stepID, "KEBAP retry", now+60_000, 0)
		insertStep(t, target, stepID, "KEBAP retry done", now+60_000, now+60_001)
		insertLogs(t, target, stepID, firstSeq, 2, false)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 2, "one name row per step, so entries are not duplicated")
		require.Equal(t, "KEBAP retry done", res.Entries[0].Step)
	})

	t.Run("a page ends before a fresh entry whose step row has not arrived", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertStep(t, target, stepID, "[3/4] RUN npm ci", now, now+1)
		insertLogs(t, target, stepID, firstSeq, 2, false)
		insertLogs(t, target, newStepID(), firstSeq+2, 1, false)
		insertLogs(t, target, stepID, firstSeq+3, 1, false)
		target.Now = time.UnixMilli(now)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Equal(t, []string{"KEBAP 0", "KEBAP 1"}, messages(res.Entries))
		require.False(t, res.HasMore, "hasMore stays false so a client waits for the step row instead of refetching at once")
	})

	t.Run("an entry whose step row never arrives has an empty step after the wait", func(t *testing.T) {
		target := newTarget()
		insertLogs(t, target, newStepID(), firstSeq, 1, false)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Equal(t, []string{"KEBAP 0"}, messages(res.Entries))
		require.Empty(t, res.Entries[0].Step)
	})

	t.Run("cuts a long step name to 256 runes", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertStep(t, target, stepID, strings.Repeat("ü", BuildStepNameRunesMax+10), now, now+1)
		insertLogs(t, target, stepID, firstSeq, 1, false)

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 1)
		require.Equal(t, strings.Repeat("ü", BuildStepNameRunesMax), res.Entries[0].Step)
	})

	t.Run("excludes rows of other deployments and workspaces", func(t *testing.T) {
		target := newTarget()
		stepID := newStepID()
		insertLogs(t, target, stepID, firstSeq, 1, false)

		otherDeployment := target
		otherDeployment.DeploymentID = uid.New(uid.DeploymentPrefix)
		otherWorkspace := target
		otherWorkspace.WorkspaceID = uid.New(uid.WorkspacePrefix)
		for _, other := range []GetBuildLogsRequest{otherDeployment, otherWorkspace} {
			insertLogs(t, other, stepID, firstSeq, 3, false)
		}

		res, err := client.GetBuildLogs(ctx, target)
		require.NoError(t, err)
		require.Len(t, res.Entries, 1)
	})
}
