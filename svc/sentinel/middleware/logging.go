package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	handler "github.com/unkeyed/unkey/svc/sentinel/routes/proxy"
)

// WithSentinelLogging logs completed sentinel requests to ClickHouse.
// Timing/response data populated by handler via context during proxy execution.
func WithSentinelLogging(ch clickhouse.ClickHouse, clk clock.Clock, sentinelID, region string) zen.Middleware {
	return func(next zen.HandleFunc) zen.HandleFunc {
		return func(ctx context.Context, s *zen.Session) error {
			// nolint:exhaustruct
			tracking := &handler.SentinelRequestTracking{
				StartTime: clk.Now(),
			}
			ctx = handler.WithSentinelTracking(ctx, tracking)

			err := next(ctx, s)

			// Only log if deployment/instance data was set
			if tracking.Deployment != nil && tracking.Instance != nil {
				endTime := clk.Now()
				totalLatency := endTime.Sub(tracking.StartTime).Milliseconds()

				var instanceLatency, sentinelLatency int64
				if !tracking.InstanceStart.IsZero() && !tracking.InstanceEnd.IsZero() {
					instanceLatency = tracking.InstanceEnd.Sub(tracking.InstanceStart).Milliseconds()
					sentinelLatency = totalLatency - instanceLatency
				}

				req := s.Request()

				ch.BufferSentinelRequest(schema.SentinelRequest{
					RequestID:       tracking.RequestID,
					Time:            tracking.StartTime.UnixMilli(),
					WorkspaceID:     tracking.Deployment.WorkspaceID,
					EnvironmentID:   tracking.Deployment.EnvironmentID,
					ProjectID:       tracking.Deployment.ProjectID,
					SentinelID:      sentinelID,
					DeploymentID:    tracking.DeploymentID,
					InstanceID:      tracking.Instance.ID,
					InstanceAddress: tracking.Instance.Address,
					Region:          region,
					Method:          strings.ToUpper(req.Method),
					Host:            req.Host,
					Path:            req.URL.Path,
					QueryString:     req.URL.RawQuery,
					QueryParams:     req.URL.Query(),
					RequestHeaders:  formatHeaders(req.Header),
					RequestBody:     string(tracking.RequestBody),
					ResponseStatus:  tracking.ResponseStatus,
					ResponseHeaders: tracking.ResponseHeaders,
					ResponseBody:    string(tracking.ResponseBody),
					UserAgent:       req.UserAgent(),
					IPAddress:       s.Location(),
					TotalLatency:    totalLatency,
					InstanceLatency: instanceLatency,
					SentinelLatency: sentinelLatency,
				})
			}

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
	result := make([]string, 0, len(headers))
	for key, values := range headers {
		lk := strings.ToLower(key)
		if lk == "authorization" {
			result = append(result, formatHeader(key, "[REDACTED]"))
		} else {
			for _, value := range values {
				result = append(result, formatHeader(key, value))
			}
		}
	}
	return result
}
