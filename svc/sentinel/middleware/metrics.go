package middleware

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// sentinelProxyErrorsTotal counts proxy errors by error type.
	//
	// Labels:
	//   - "error_type": "timeout", "conn_refused", "conn_reset", "dns_failure", "client_canceled", "other"
	sentinelProxyErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "sentinel_proxy_errors_total",
			Help: "Total number of proxy errors by error type.",
		},
		[]string{"error_type"},
	)
)

// categorizeProxyErrorTypeForMetrics returns a short label for the type of proxy error,
// suitable for use as a prometheus label value.
func categorizeProxyErrorTypeForMetrics(err error) string {
	if errors.Is(err, context.Canceled) {
		return "client_canceled"
	}
	if errors.Is(err, context.DeadlineExceeded) || os.IsTimeout(err) {
		return "timeout"
	}

	var netErr *net.OpError
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return "timeout"
		}
		if errors.Is(netErr.Err, syscall.ECONNREFUSED) {
			return "conn_refused"
		}
		if errors.Is(netErr.Err, syscall.ECONNRESET) {
			return "conn_reset"
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_failure"
	}

	return "other"
}
