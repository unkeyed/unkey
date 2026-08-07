package middleware

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"unsafe"

	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/redaction"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

// WithClickHouseLogging buffers a per-request record for ClickHouse on the
// local-instance path. The proxy package populates the tracking struct
// during proxy execution; this middleware reads it back and emits the row
// after next() returns. Cross-region requests do not populate tracking and
// are skipped — the peer frontline writes its own row.
func WithClickHouseLogging(buf *batch.BatchProcessor[schema.FrontlineRequest], clk clock.Clock, frontlineID, region, platform string) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			//nolint:exhaustruct
			tracking := &proxy.RequestTracking{
				StartTime: clk.Now(),
			}
			ctx = proxy.WithRequestTracking(ctx, tracking)

			err := next(ctx, s)

			// Tracking is only populated on the local-instance path; the
			// handler stamps DeploymentID/InstanceID before forwarding. If
			// those are empty the request was forwarded cross-region and
			// the peer logs it.
			//
			// The base row is written unconditionally — the traffic and
			// latency charts depend on it. Headers, query data, and bodies
			// are only included when an enabled logging policy opted the
			// request in via the tracking.Log* capture flags.
			if !s.ShouldLogRequestToClickHouse() || tracking.DeploymentID == "" || tracking.InstanceID == "" {
				return err
			}

			endTime := clk.Now()
			totalLatency := endTime.Sub(tracking.StartTime).Milliseconds()

			var instanceLatency, frontlineLatency int64
			if !tracking.InstanceStart.IsZero() && !tracking.InstanceEnd.IsZero() {
				instanceLatency = tracking.InstanceEnd.Sub(tracking.InstanceStart).Milliseconds()
				frontlineLatency = totalLatency - instanceLatency
			}

			req := s.Request()

			// Headers and query data are opt-in: URLs and headers routinely
			// carry secrets, so they only reach ClickHouse when an enabled
			// logging policy opted the request in.
			var queryString string
			var queryParams url.Values
			var requestHeaders, responseHeaders []string
			var userAgent, ipAddress string
			if tracking.LogRequestHeaders {
				// User agent is a request header and the client IP is
				// client-identifying data of the same sensitivity class, so
				// both ride on the request-headers opt-in rather than the
				// always-on base row.
				userAgent = req.UserAgent()
				ipAddress = s.Location()
				// Redact API keys delivered via custom header. The handler
				// records the configured KeyAuth locations on tracking;
				// without this, keys would be persisted verbatim in the
				// request log.
				requestHeaders = formatHeaders(req.Header, toSet(tracking.RedactedHeaders))
			}
			if tracking.LogQuery {
				// Redact API keys delivered via query parameter, mirroring
				// the header redaction above.
				secretParams := toSet(tracking.RedactedQueryParams)

				queryString = req.URL.RawQuery
				queryParams = req.URL.Query()
				if len(secretParams) > 0 {
					queryParams = redactQueryParams(queryParams, secretParams)
					// Re-encode rather than log the raw string so the secret
					// never reaches ClickHouse. This loses the original key
					// ordering and encoding, which is acceptable for a debug
					// log field.
					queryString = queryParams.Encode()
				}
			}
			if tracking.LogResponseHeaders {
				responseHeaders = formatHeaders(s.ResponseWriter().Header(), nil)
			}

			buf.Buffer(schema.FrontlineRequest{
				RequestID:        tracking.RequestID,
				Time:             tracking.StartTime.UnixMilli(),
				WorkspaceID:      tracking.WorkspaceID,
				ProjectID:        tracking.ProjectID,
				AppID:            tracking.AppID,
				EnvironmentID:    tracking.EnvironmentID,
				FrontlineID:      frontlineID,
				DeploymentID:     tracking.DeploymentID,
				InstanceID:       tracking.InstanceID,
				InstanceAddress:  tracking.Address,
				Region:           region,
				Platform:         platform,
				Method:           strings.ToUpper(req.Method),
				Host:             req.Host,
				Path:             req.URL.Path,
				QueryString:      queryString,
				QueryParams:      queryParams,
				RequestHeaders:   requestHeaders,
				RequestBody:      redactBody(tracking.RequestBody, tracking.BodyRedactors),
				ResponseStatus:   int32(s.StatusCode()),
				ResponseHeaders:  responseHeaders,
				ResponseBody:     redactBody(tracking.ResponseBody, tracking.BodyRedactors),
				UserAgent:        userAgent,
				IPAddress:        ipAddress,
				TotalLatency:     totalLatency,
				InstanceLatency:  instanceLatency,
				FrontlineLatency: frontlineLatency,
			})

			return err
		}
	}
}

func redactBody(body []byte, redactors []*redaction.Redactor) string {
	for _, r := range redactors {
		body = r.Redact(body)
	}
	if len(body) == 0 {
		return ""
	}
	// Captured and redacted bodies are immutable after this point. The string
	// stored in the batch keeps the backing bytes alive. Avoid a second
	// full-body allocation when converting []byte to string.
	return unsafe.String(unsafe.SliceData(body), len(body))
}

func formatHeader(key, value string) string {
	var b strings.Builder
	b.Grow(len(key) + 2 + len(value))
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	return b.String()
}

// formatHeaders serializes headers for logging. The Authorization header is
// always redacted; values of any header whose lowercased name is in secret
// (configured KeyAuth header locations) are redacted too.
func formatHeaders(headers http.Header, secret map[string]struct{}) []string {
	result := make([]string, 0, len(headers))
	for key, values := range headers {
		lk := strings.ToLower(key)
		if _, isSecret := secret[lk]; lk == "authorization" || isSecret {
			result = append(result, formatHeader(key, "[REDACTED]"))
			continue
		}
		for _, value := range values {
			result = append(result, formatHeader(key, value))
		}
	}
	return result
}

// toSet builds a lookup set from a slice of names, returning nil for an empty
// input so callers can cheaply skip redaction work.
func toSet(names []string) map[string]struct{} {
	if len(names) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[n] = struct{}{}
	}
	return set
}

// redactQueryParams returns a copy of values with the value of every parameter
// named in secret replaced by [REDACTED]. The input is not mutated.
func redactQueryParams(values url.Values, secret map[string]struct{}) url.Values {
	redacted := make(url.Values, len(values))
	for key, vs := range values {
		if _, isSecret := secret[key]; isSecret {
			redacted[key] = []string{"[REDACTED]"}
			continue
		}
		redacted[key] = vs
	}
	return redacted
}
