package middleware

import (
	"context"
	"net/http"
	"strings"
	"unsafe"

	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
)

// WithClickHouseLogging buffers a per-request record for ClickHouse on the
// local-instance path. The proxy package populates the tracking struct
// during proxy execution; this middleware reads it back and emits the row
// after next() returns. Cross-region requests do not populate tracking and
// are skipped — the peer frontline writes its own row.
//
// buf may be a noop processor when ClickHouse is not configured; the capture
// work still runs and the row is discarded by the noop sink.
func WithClickHouseLogging(buf *batch.BatchProcessor[schema.SentinelRequest], clk clock.Clock, frontlineID, region, platform string) zen.Middleware {
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

			// url.Query() re-parses RawQuery and allocates a map; skip it
			// for the common no-query case.
			var queryParams map[string][]string
			if req.URL.RawQuery != "" {
				queryParams = req.URL.Query()
			}

			buf.Buffer(schema.SentinelRequest{
				RequestID:       tracking.RequestID,
				Time:            tracking.StartTime.UnixMilli(),
				WorkspaceID:     tracking.WorkspaceID,
				EnvironmentID:   tracking.EnvironmentID,
				ProjectID:       tracking.ProjectID,
				SentinelID:      frontlineID,
				DeploymentID:    tracking.DeploymentID,
				InstanceID:      tracking.InstanceID,
				InstanceAddress: tracking.Address,
				Region:          region,
				Platform:        platform,
				Method:          strings.ToUpper(req.Method),
				Host:            req.Host,
				Path:            req.URL.Path,
				QueryString:     req.URL.RawQuery,
				QueryParams:     queryParams,
				RequestHeaders:  formatHeaders(req.Header),
				RequestBody:     unsafe.String(unsafe.SliceData(tracking.RequestBody), len(tracking.RequestBody)),
				ResponseStatus:  int32(s.StatusCode()),
				ResponseHeaders: formatHeaders(s.ResponseWriter().Header()),
				ResponseBody:    unsafe.String(unsafe.SliceData(tracking.ResponseBody), len(tracking.ResponseBody)),
				UserAgent:       req.UserAgent(),
				IPAddress:       s.Location(),
				TotalLatency:    totalLatency,
				InstanceLatency: instanceLatency,
				SentinelLatency: frontlineLatency,
			})

			return err
		}
	}
}

func formatHeader(key, value string) string {
	var b strings.Builder
	b.Grow(len(key) + 2 + len(value))
	b.WriteString(key)
	b.WriteString(": ")
	b.WriteString(value)
	return b.String()
}

func formatHeaders(headers http.Header) []string {
	n := 0
	for _, values := range headers {
		n += len(values)
	}
	result := make([]string, 0, n)
	for key, values := range headers {
		// EqualFold instead of ToLower: ToLower allocates a copy of every
		// header name on every request.
		if strings.EqualFold(key, "Authorization") {
			result = append(result, formatHeader(key, "[REDACTED]"))
		} else {
			for _, value := range values {
				result = append(result, formatHeader(key, value))
			}
		}
	}
	return result
}
