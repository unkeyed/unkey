package transform

import (
	"encoding/json"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

// Runtime converts a runtime_logs_raw_v1 row into the in-memory Record
// shape, applying the runtime-half of the drain's filter spec. Returns
// (record, true) when the record should be forwarded; (zero, false) when
// the filter rejects it.
func Runtime(row schema.RuntimeLog, f RuntimeFilter) (sinks.Record, bool) {
	var zero sinks.Record
	if !severityPasses(row.Severity, f.MinSeverity) {
		return zero, false
	}

	// Attributes is stored as a JSON-marshaled string in ClickHouse to
	// match the InstanceCheckpoint convention. We unmarshal here once so
	// every sink operates on a typed map; an unparseable payload is
	// silently dropped (the data plane already handled fallback parsing
	// upstream in Vector — anything reaching us is well-formed JSON or an
	// empty string).
	var attrs map[string]any
	if row.Attributes != "" {
		var parsed map[string]any
		if err := json.Unmarshal([]byte(row.Attributes), &parsed); err == nil && len(parsed) > 0 {
			attrs = parsed
		}
	}

	return sinks.Record{
		Kind:          sinks.RecordRuntime,
		TimeMs:        row.Time,
		SeverityText:  row.Severity,
		WorkspaceID:   row.WorkspaceID,
		ProjectID:     row.ProjectID,
		EnvironmentID: row.EnvironmentID,
		AppID:         row.AppID,
		DeploymentID:  row.DeploymentID,
		Region:        row.Region,
		Platform:      row.Platform,
		K8sPodName:    row.K8sPodName,
		Body:          row.Message,
		Attributes:    attrs,
		// transform/ is provider-agnostic conversion; the cursor
		// position (set by the coordinator's fetchRuntime when reading
		// from CH) is not visible here. Tests construct records
		// directly without a real cursor, so zero is the correct
		// sentinel.
		CursorTimeMs: 0,
		// LastID is the sink-side dedup key; coordinator-level cursor
		// pagination already prevents duplicates today, so leave it
		// empty until tests/callers that need an Idempotency-Key fill
		// in the row's log_id.
		LastID: "",
	}, true
}

// severityPasses returns false when the row's level is below the floor.
// Empty min lets every level through; an unrecognised min defaults to
// "info" so a misconfiguration never silently swallows errors.
func severityPasses(rowSeverity, min string) bool {
	if min == "" {
		return true
	}

	return sinks.SeverityNumber(rowSeverity) >= sinks.SeverityNumber(min)
}
