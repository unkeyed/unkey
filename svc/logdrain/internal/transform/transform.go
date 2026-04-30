// Package transform converts raw ClickHouse rows from runtime_logs_raw_v1
// and sentinel_requests_raw_v1 into the in-memory Record shape the sinks
// consume. It also applies the per-drain filter knobs (severity floor,
// status code, body redaction) so a worker's hot path never has to
// re-parse fields the transform already touched.
package transform

// RuntimeFilter mirrors the runtime half of log_drains.filters JSON. Zero
// value means no filtering.
type RuntimeFilter struct {
	// MinSeverity drops records whose severity_number is below this floor.
	// Empty string means "info" — same default Vector applies to lines
	// that omit a level.
	MinSeverity string
}

// RequestFilter mirrors the request half of log_drains.filters. The
// IncludeBodies flag is the only opt-in: by default request_body and
// response_body are stripped before they leave the transform stage so a
// misconfigured drain cannot accidentally exfiltrate user payloads.
type RequestFilter struct {
	// StatusMatchers is a list of strings matched against the status code
	// using the matcher language documented in MatchStatus (e.g. ">=400",
	// "5xx", "404"). Empty means accept all.
	StatusMatchers []string
	// ExcludePaths drops records whose path matches any prefix in the list.
	// Useful for /healthz spam.
	ExcludePaths []string
	// IncludeBodies opts into forwarding request_body and response_body.
	// Off by default — see package comment.
	IncludeBodies bool
}
