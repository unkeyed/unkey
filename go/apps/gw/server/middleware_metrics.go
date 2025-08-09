package server

import (
	"context"
	"fmt"
	"strings"

	"github.com/unkeyed/unkey/go/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/go/pkg/fault"
)

// EventBuffer defines the interface for buffering events to be sent to ClickHouse.
type EventBuffer interface {
	BufferApiRequest(schema.ApiRequestV1)
}

// WithMetrics returns middleware that collects metrics about each request,
// including request counts, latencies, and status codes.
//
// If an EventBuffer is provided, it will also buffer request data for ClickHouse.
// We need some sort of config to determine what of the request and or response we should redacted, since we would potentially
// log sensitive information to our ClickHouse instance.
func WithMetrics(eventBuffer EventBuffer, region string) Middleware {
	return func(next HandleFunc) HandleFunc {
		return func(ctx context.Context, s *Session) error {
			// Set headers that don't depend on execution time BEFORE calling next
			s.w.Header().Set("X-Unkey-Request-Id", s.RequestID())
			s.w.Header().Set("X-Unkey-Region", region)

			nextErr := next(ctx, s)

			// Collect headers for logging
			requestHeaders := []string{}
			for k, vv := range s.r.Header {
				if strings.ToLower(k) == "authorization" {
					requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, "[REDACTED]"))
				} else {
					requestHeaders = append(requestHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
				}
			}

			responseHeaders := []string{}
			for k, vv := range s.w.Header() {
				responseHeaders = append(responseHeaders, fmt.Sprintf("%s: %s", k, strings.Join(vv, ",")))
			}

			// Record metrics
			// labelValues := []string{s.r.Method, s.r.URL.Path, strconv.Itoa(s.responseStatus)}
			// metrics.HTTPRequestBodySize.WithLabelValues(labelValues...).Observe(float64(len(s.requestBody)))
			// metrics.HTTPRequestTotal.WithLabelValues(labelValues...).Inc()
			// metrics.HTTPRequestLatency.WithLabelValues(labelValues...).Observe(s.Latency().Seconds())

			// Buffer to ClickHouse if enabled
			// We don't need this ATM
			// if eventBuffer != nil && s.r.Header.Get("X-Unkey-Metrics") != "disabled" {
			// 	// Extract IP address from headers
			// 	ips := strings.Split(s.r.Header.Get("X-Forwarded-For"), ",")
			// 	ipAddress := ""
			// 	if len(ips) > 0 {
			// 		ipAddress = strings.TrimSpace(ips[0])
			// 	}
			// 	if ipAddress == "" {
			// 		ipAddress = s.Location()
			// 	}

			// 	eventBuffer.BufferApiRequest(schema.ApiRequestV1{
			// 		WorkspaceID:     s.WorkspaceID,
			// 		RequestID:       s.RequestID(),
			// 		Time:            s.startTime.UnixMilli(),
			// 		Host:            s.r.Host,
			// 		Method:          s.r.Method,
			// 		Path:            s.r.URL.Path,
			// 		RequestHeaders:  requestHeaders,
			// 		RequestBody:     string(s.requestBody),
			// 		ResponseStatus:  s.responseStatus,
			// 		ResponseHeaders: responseHeaders,
			// 		ResponseBody:    string(s.responseBody),
			// 		Error:           getErrorMessage(nextErr),
			// 		ServiceLatency:  s.Latency().Milliseconds(),
			// 		UserAgent:       s.UserAgent(),
			// 		IpAddress:       ipAddress,
			// 		Country:         "",
			// 		City:            "",
			// 		Colo:            "",
			// 		Continent:       "",
			// 	})
			// }

			return nextErr
		}
	}
}

// getErrorMessage extracts the user-facing error message if available.
func getErrorMessage(err error) string {
	if err == nil {
		return ""
	}
	return fault.UserFacingMessage(err)
}
