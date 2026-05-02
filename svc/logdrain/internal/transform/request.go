package transform

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/svc/logdrain/internal/sinks"
)

// Request converts a sentinel_requests_raw_v1 row into the in-memory
// Record shape with HTTP fields projected onto OTLP semantic conventions.
// Returns (record, true) on accept, (zero, false) on filter reject.
//
// Body redaction is on by default: request_body and response_body are
// dropped unless f.IncludeBodies is explicitly true. The headers arrays
// are forwarded as-is — if a customer worries about token leakage in
// Authorization headers, they should add a per-key sanitization step at
// the Sentinel layer rather than rely on the drain to redact a list whose
// shape we cannot know in advance.
func Request(row schema.SentinelRequest, f RequestFilter) (sinks.Record, bool) {
	var zero sinks.Record
	if !statusMatches(row.ResponseStatus, f.StatusMatchers) {
		return zero, false
	}

	if pathExcluded(row.Path, f.ExcludePaths) {
		return zero, false
	}

	severity := severityFromStatus(row.ResponseStatus)
	body := fmt.Sprintf("%s %s %d", row.Method, row.Path, row.ResponseStatus)

	attrs := map[string]any{
		"http.request.method":       row.Method,
		"url.path":                  row.Path,
		"url.query":                 row.QueryString,
		"http.request.host":         row.Host,
		"http.response.status_code": int64(row.ResponseStatus),
		"http.user_agent":           row.UserAgent,
		"client.address":            row.IPAddress,
		"unkey.request_id":          row.RequestID,
		"unkey.sentinel_id":         row.SentinelID,
		"unkey.instance_id":         row.InstanceID,
		"unkey.instance_address":    row.InstanceAddress,
		"unkey.total_latency_ms":    row.TotalLatency,
		"unkey.instance_latency_ms": row.InstanceLatency,
		"unkey.sentinel_latency_ms": row.SentinelLatency,
	}

	if len(row.RequestHeaders) > 0 {
		attrs["http.request.headers"] = row.RequestHeaders
	}

	if len(row.ResponseHeaders) > 0 {
		attrs["http.response.headers"] = row.ResponseHeaders
	}

	if f.IncludeBodies {
		if row.RequestBody != "" {
			attrs["http.request.body"] = row.RequestBody
		}

		if row.ResponseBody != "" {
			attrs["http.response.body"] = row.ResponseBody
		}
	}

	return sinks.Record{
		Kind:          sinks.RecordRequest,
		TimeMs:        row.Time,
		SeverityText:  severity,
		WorkspaceID:   row.WorkspaceID,
		ProjectID:     row.ProjectID,
		EnvironmentID: row.EnvironmentID,
		// Sentinel access logs do not carry app_id today; the column lives
		// on the runtime side. Forwarding empty so the field is consistent.
		AppID:        "",
		DeploymentID: row.DeploymentID,
		Region:       row.Region,
		Platform:     row.Platform,
		// k8s_pod_name lives on runtime logs; for request logs the
		// instance address (set in attributes) is the closest analog.
		K8sPodName: "",
		Body:       body,
		Attributes: attrs,
		// transform/ doesn't observe the CH-level cursor; coordinator
		// fetchRequest sets it. See runtime.go for the same note.
		CursorTimeMs: 0,
		// request_id doubles as the cursor tiebreaker and the
		// per-event Idempotency-Key for providers that support it.
		LastID: row.RequestID,
	}, true
}

// severityFromStatus maps an HTTP status into the severity_text the
// providers expect. 5xx is "error", 4xx is "warn", everything else is
// "info" — matches what every major APM does out of the box.
func severityFromStatus(status int32) string {
	switch {
	case status >= 500:
		return "error"
	case status >= 400:
		return "warn"
	default:
		return "info"
	}
}

// statusMatches accepts the row when the matcher list is empty, or when
// any matcher matches. Supported forms (case-insensitive):
//
//	"200"      exact
//	">=400"    inclusive lower bound
//	">500"     exclusive lower bound
//	"<300"     exclusive upper bound
//	"<=299"    inclusive upper bound
//	"5xx"      class match (1xx through 5xx)
func statusMatches(status int32, matchers []string) bool {
	if len(matchers) == 0 {
		return true
	}

	for _, m := range matchers {
		if matchOne(int(status), m) {
			return true
		}
	}

	return false
}

func matchOne(status int, m string) bool {
	m = strings.ToLower(strings.TrimSpace(m))
	if m == "" {
		return false
	}

	if strings.HasSuffix(m, "xx") && len(m) == 3 {
		class, err := strconv.Atoi(m[:1])
		if err != nil {
			return false
		}
		return status >= class*100 && status < (class+1)*100
	}

	switch {
	case strings.HasPrefix(m, ">="):
		n, err := strconv.Atoi(m[2:])
		return err == nil && status >= n
	case strings.HasPrefix(m, "<="):
		n, err := strconv.Atoi(m[2:])
		return err == nil && status <= n
	case strings.HasPrefix(m, ">"):
		n, err := strconv.Atoi(m[1:])
		return err == nil && status > n
	case strings.HasPrefix(m, "<"):
		n, err := strconv.Atoi(m[1:])
		return err == nil && status < n
	}

	n, err := strconv.Atoi(m)
	return err == nil && status == n
}

func pathExcluded(path string, excludes []string) bool {
	for _, p := range excludes {
		if p != "" && strings.HasPrefix(path, p) {
			return true
		}
	}

	return false
}
