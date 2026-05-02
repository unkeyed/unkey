package coordinator

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/svc/logdrain/internal/metrics"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

// runtimeQuery reads from runtime_logs_raw_v2 — v2's sort key
// (workspace_id, project_id, environment_id, inserted_at, time) is
// fully aligned with this query's WHERE + ORDER BY.
//
// The cursor predicate is the prune-friendly OR-form
//
//	inserted_at > T OR (inserted_at = T AND log_id > L)
//
// rather than a single tuple compare. The first clause is a plain
// `inserted_at > T` — sort-key-prefix prunable, so ClickHouse skips
// every granule whose max(inserted_at) ≤ T without reading it. The
// second clause only fires inside the (typically single) granule that
// contains rows at exactly inserted_at = T, where `log_id` (a stored
// String column) is the unambiguous in-block tiebreaker.
//
// `log_id` is Vector's UUID-v7-shaped per-row id, so unlike the prior
// cityHash64(deployment_id, time, message) fingerprint it is strictly
// unique per row — no theoretical collision risk on the cursor — and
// the comparison is against a stored column rather than a computed
// expression, so the optimizer doesn't have to inline cityHash64 for
// every row.
const runtimeQuery = `
SELECT
  inserted_at, log_id,
  time, severity, message,
  workspace_id, project_id, environment_id, app_id, deployment_id,
  k8s_pod_name, region, platform,
  toJSONString(attributes) AS attributes
FROM default.runtime_logs_raw_v2
WHERE workspace_id = {workspace_id:String}
  AND environment_id = {environment_id:String}
  AND ({project_id:String} = '' OR project_id = {project_id:String})
  AND (
    inserted_at > {prev_watermark:Int64}
    OR (inserted_at = {prev_watermark:Int64}
        AND log_id > {prev_last_id:String})
  )
ORDER BY inserted_at, log_id
LIMIT {max_rows:UInt32}
`

// requestQuery uses the same prune-friendly OR-form as runtimeQuery:
// `time > T` is sort-key-prefix prunable, the request_id clause only
// fires inside the granule(s) at exactly `time = T`. request_id is a
// stored String column already unique per row, so unlike the prior
// cityHash64(request_id) form it doesn't force the optimizer to inline
// a hash on every row.
const requestQuery = `
SELECT
  time, request_id,
  workspace_id, project_id, environment_id,
  deployment_id, instance_id, instance_address,
  region, platform, method, host, path, response_status,
  user_agent, ip_address, total_latency
FROM default.sentinel_requests_raw_v1
WHERE workspace_id = {workspace_id:String}
  AND environment_id = {environment_id:String}
  AND ({project_id:String} = '' OR project_id = {project_id:String})
  AND (
    time > {prev_watermark:Int64}
    OR (time = {prev_watermark:Int64}
        AND request_id > {prev_last_id:String})
  )
ORDER BY time, request_id
LIMIT {max_rows:UInt32}
`

// fetchBatch runs the per-source CH query, converts rows to sinks.Record,
// and returns the (timeMs, lastID) at the tail of the returned slice. The
// caller uses that as the per-drain cursor advance target after a
// successful Send.
func (c *Coordinator) fetchBatch(ctx context.Context, g Group, cur cursor) ([]sinks.Record, int64, string, error) {
	queryStart := time.Now()
	defer func() {
		metrics.ClickHouseQueryDuration.
			WithLabelValues(string(g.Source), strconv.Itoa(c.cfg.Ordinal)).
			Observe(time.Since(queryStart).Seconds())
	}()

	switch g.Source {
	case SourceRuntime:
		return c.fetchRuntime(ctx, g, cur)
	case SourceRequest:
		return c.fetchRequest(ctx, g, cur)
	default:
		return nil, 0, "", fmt.Errorf("unknown source %q", g.Source)
	}
}

func (c *Coordinator) fetchRuntime(ctx context.Context, g Group, cur cursor) ([]sinks.Record, int64, string, error) {
	return runCursorQuery(ctx, c, g, cur, runtimeQuery, toRuntimeRecord)
}

func (c *Coordinator) fetchRequest(ctx context.Context, g Group, cur cursor) ([]sinks.Record, int64, string, error) {
	return runCursorQuery(ctx, c, g, cur, requestQuery, toRequestRecord)
}

func toRuntimeRecord(r runtimeRow) sinks.Record {
	return sinks.Record{
		Kind:          sinks.RecordRuntime,
		TimeMs:        r.Time,
		SeverityText:  r.Severity,
		WorkspaceID:   r.WorkspaceID,
		ProjectID:     r.ProjectID,
		EnvironmentID: r.EnvironmentID,
		AppID:         r.AppID,
		DeploymentID:  r.DeploymentID,
		Region:        r.Region,
		Platform:      r.Platform,
		K8sPodName:    r.K8sPodName,
		Body:          r.Message,
		Attributes:    parseAttrs(r.Attributes),
		// CursorTimeMs is `inserted_at` (CH's ingest timestamp), not
		// the log's emission `time`. inserted_at is monotonically
		// stable in a way `time` isn't (clock skew, late-arriving
		// out-of-order Vector batches), so the cursor uses it for
		// pagination. The per-drain fan-out compares (CursorTimeMs,
		// LastID) against the drain's individual cursor.
		CursorTimeMs: r.InsertedAt,
		// log_id is Vector's stable per-row id; doubles as the
		// cursor tiebreaker and the Idempotency-Key for providers
		// that support per-event dedup.
		LastID: r.LogID,
	}
}

func toRequestRecord(r requestRow) sinks.Record {
	return sinks.Record{
		Kind: sinks.RecordRequest,
		// Request rows have no severity (every entry is a
		// completed HTTP transaction); leaving SeverityText empty
		// makes sinks.SeverityNumber default to INFO downstream.
		SeverityText:  "",
		TimeMs:        r.Time,
		WorkspaceID:   r.WorkspaceID,
		ProjectID:     r.ProjectID,
		EnvironmentID: r.EnvironmentID,
		// AppID and K8sPodName don't apply to request rows —
		// sentinel observes the proxy edge, not the workload pod.
		AppID:        "",
		K8sPodName:   "",
		DeploymentID: r.DeploymentID,
		Region:       r.Region,
		Platform:     r.Platform,
		Body:         fmt.Sprintf("%s %s %d", r.Method, r.Path, r.ResponseStatus),
		// Sentinel rows have no separate ingest timestamp; `time`
		// is what the (time, request_id) cursor pairs with.
		CursorTimeMs: r.Time,
		// request_id doubles as the cursor tiebreaker and the
		// Idempotency-Key for providers that support per-event
		// dedup.
		LastID: r.RequestID,
		Attributes: map[string]any{
			"request_id":       r.RequestID,
			"method":           r.Method,
			"host":             r.Host,
			"path":             r.Path,
			"response_status":  r.ResponseStatus,
			"user_agent":       r.UserAgent,
			"ip_address":       r.IPAddress,
			"total_latency_ms": r.TotalLatency,
			"instance_id":      r.InstanceID,
			"instance_address": r.InstanceAddress,
		},
	}
}

// runCursorQuery is the shared scaffold behind fetchRuntime / fetchRequest.
// Both sources share a parameter shape (workspace/project/env + the
// (prev_watermark, prev_last_id) cursor + max_rows) and the same metric
// hooks; the caller supplies only the source-specific row type, query
// template, and row → sinks.Record mapper.
//
// Returning the last record's CursorTimeMs/LastID as the new cursor keeps
// "what advance to commit" colocated with "what we just sent" — the
// transform is the only place either source needs to think about which
// row column drives the watermark.
func runCursorQuery[T any](
	ctx context.Context,
	c *Coordinator,
	g Group,
	cur cursor,
	query string,
	transform func(T) sinks.Record,
) ([]sinks.Record, int64, string, error) {
	rows, err := clickhouse.Select[T](ctx, c.ch.Conn(), query, map[string]string{
		"workspace_id":   g.Workspace,
		"project_id":     g.Project,
		"environment_id": g.Env,
		"prev_watermark": strconv.FormatInt(cur.timeMs, 10),
		"prev_last_id":   cur.lastID,
		"max_rows":       strconv.Itoa(c.cfg.MaxBatchRecords),
	})
	if err != nil {
		metrics.ClickHouseQueryErrors.WithLabelValues(classifyCHError(err)).Inc()
		return nil, 0, "", err
	}
	metrics.ClickHouseRecordsFetched.WithLabelValues(string(g.Source)).Add(float64(len(rows)))
	if len(rows) == 0 {
		return nil, cur.timeMs, cur.lastID, nil
	}
	out := make([]sinks.Record, len(rows))
	for i, r := range rows {
		out[i] = transform(r)
	}
	last := out[len(out)-1]
	return out, last.CursorTimeMs, last.LastID, nil
}
