package clickhouse

import (
	"context"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/unkeyed/unkey/pkg/fault"
)

// Must match step.maxLength in the v2 spec's BuildLogEntry
const BuildStepNameRunesMax = 256

const buildLogsPageBytesMax = 1024 * 1024

// ctrl writes step rows and log rows through separate batches, so a fresh
// entry can land before its step row. A page ends before such an entry until
// the row arrives or the entry is this old, so a missing row cannot stall it
const buildLogStepRowWait = 10 * time.Second

type buildLogRow struct {
	Seq         uint64 `ch:"seq"`
	Time        int64  `ch:"time"`
	StepID      string `ch:"step_id"`
	Stderr      bool   `ch:"stderr"`
	Message     string `ch:"message"`
	FetchedRows uint64 `ch:"fetched_rows"`
}

type buildStepNameRow struct {
	StepID string `ch:"step_id"`
	Name   string `ch:"name"`
}

type GetBuildLogsRequest struct {
	WorkspaceID  string
	ProjectID    string
	DeploymentID string
	// StepID restricts the entries to one step. Empty returns every step
	StepID string
	// AfterSeq returns only entries with a greater Seq. 0 starts at the first
	AfterSeq uint64
	Limit    int
	// Now is the read time. An entry newer than Now minus buildLogStepRowWait
	// waits for its step row
	Now time.Time
}

type BuildLogsPage struct {
	Entries []BuildLogEntry
	HasMore bool
}

// BuildLogEntry is one BuildKit output chunk of a deployment's build. Time is
// unix milliseconds. Step is the step name cut to BuildStepNameRunesMax runes,
// or empty when no build_steps_v1 row matched StepID within
// buildLogStepRowWait. Stderr is true for a stderr chunk
type BuildLogEntry struct {
	Seq     uint64
	Time    int64
	StepID  string
	Step    string
	Stderr  bool
	Message string
}

func (c *Client) GetBuildLogs(ctx context.Context, req GetBuildLogsRequest) (BuildLogsPage, error) {
	rows, err := Select[buildLogRow](ctx, c.conn, `
		-- Keep a row while the messages before it total less than page_bytes_max,
		-- so the first row always fits and a client never gets stuck at its cursor
		SELECT seq, time, step_id, stderr, message, fetched_rows
		FROM (
			-- A window value cannot be filtered in the SELECT that computes it.
			-- fetched_rows counts the rows before the byte cut, so a cut sets hasMore
			SELECT seq, time, step_id, stderr, message,
				sum(length(message)) OVER (ORDER BY seq ROWS BETWEEN UNBOUNDED PRECEDING AND 1 PRECEDING) AS bytes_before,
				count() OVER () AS fetched_rows
			FROM (
				-- Limit before the window so it only sums this page. The extra row
				-- shows whether more rows exist
				SELECT seq, time, step_id, stderr, message
				FROM default.build_step_logs_v1
				WHERE workspace_id = {workspace_id:String}
				  AND project_id = {project_id:String}
				  AND deployment_id = {deployment_id:String}
				  AND ({step_id:String} = '' OR step_id = {step_id:String})
				  AND seq > {after_seq:UInt64}
				ORDER BY seq
				LIMIT {limit_plus_one:UInt64}
			)
		)
		WHERE bytes_before < {page_bytes_max:UInt64}
		ORDER BY seq`, map[string]string{
		"workspace_id":   req.WorkspaceID,
		"project_id":     req.ProjectID,
		"deployment_id":  req.DeploymentID,
		"step_id":        req.StepID,
		"after_seq":      strconv.FormatUint(req.AfterSeq, 10),
		"limit_plus_one": strconv.Itoa(req.Limit + 1),
		"page_bytes_max": strconv.Itoa(buildLogsPageBytesMax),
	})
	if err != nil {
		return BuildLogsPage{}, fault.Wrap(err, fault.Internal("failed to query build step logs"))
	}
	if len(rows) == 0 {
		return BuildLogsPage{Entries: []BuildLogEntry{}, HasMore: false}, nil
	}

	fetchedRows := rows[0].FetchedRows
	rows = rows[:min(len(rows), req.Limit)]

	stepIDs := make([]string, 0, len(rows))
	seenStepIDs := make(map[string]struct{}, len(rows))
	for _, row := range rows {
		if _, ok := seenStepIDs[row.StepID]; !ok {
			seenStepIDs[row.StepID] = struct{}{}
			stepIDs = append(stepIDs, row.StepID)
		}
	}

	// The completed row wins over the start row, and a retry's row over the
	// first attempt's
	steps, err := Select[buildStepNameRow](ctx, c.conn, `
		SELECT step_id, name
		FROM default.build_steps_v1
		WHERE workspace_id = {workspace_id:String}
		  AND project_id = {project_id:String}
		  AND deployment_id = {deployment_id:String}
		  AND step_id IN {step_ids:Array(String)}
		ORDER BY started_at DESC, completed_at DESC
		LIMIT 1 BY step_id`, map[string]string{
		"workspace_id":  req.WorkspaceID,
		"project_id":    req.ProjectID,
		"deployment_id": req.DeploymentID,
		"step_ids":      StringArrayParam(stepIDs),
	})
	if err != nil {
		return BuildLogsPage{}, fault.Wrap(err, fault.Internal("failed to query build step names"))
	}

	stepNames := make(map[string]string, len(steps))
	for _, step := range steps {
		name := step.Name
		if utf8.RuneCountInString(name) > BuildStepNameRunesMax {
			name = string([]rune(name)[:BuildStepNameRunesMax])
		}
		stepNames[step.StepID] = name
	}

	stepRowWaitStart := req.Now.Add(-buildLogStepRowWait).UnixMilli()
	entries := make([]BuildLogEntry, 0, len(rows))
	for _, row := range rows {
		if _, ok := stepNames[row.StepID]; !ok && row.Time > stepRowWaitStart {
			return BuildLogsPage{Entries: entries, HasMore: false}, nil
		}
		entries = append(entries, BuildLogEntry{
			Seq:     row.Seq,
			Time:    row.Time,
			StepID:  row.StepID,
			Step:    stepNames[row.StepID],
			Stderr:  row.Stderr,
			Message: row.Message,
		})
	}

	return BuildLogsPage{
		Entries: entries,
		HasMore: fetchedRows > uint64(len(entries)),
	}, nil
}
