package middleware

import (
	"context"
	"errors"
	"net"
	"os"
	"syscall"

	"github.com/prometheus/client_golang/prometheus"
)

// Metrics holds all Prometheus metrics for the sentinel middleware package.
type Metrics struct {
	ProxyErrorsTotal *prometheus.CounterVec
	RequestsTotal    *prometheus.CounterVec
	RequestDuration  *prometheus.HistogramVec
	ActiveRequests   *prometheus.GaugeVec
}

// NewMetrics creates and registers all middleware metrics with the given registerer.
func NewMetrics(reg prometheus.Registerer) *Metrics {
	m := &Metrics{
		ProxyErrorsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "proxy_errors_total",
				Help:      "Total number of proxy errors by error type.",
			},
			[]string{"error_type"},
		),
		RequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "requests_total",
				Help:      "Total number of requests processed by sentinel",
			},
			[]string{"status_code", "error_type", "region"},
		),
		RequestDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "request_duration_seconds",
				Help:      "Request duration in seconds",
				Buckets:   prometheus.DefBuckets,
			},
			[]string{"status_code", "error_type", "region"},
		),
		ActiveRequests: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Namespace: "unkey",
				Subsystem: "sentinel",
				Name:      "active_requests",
				Help:      "Number of requests currently being processed",
			},
			[]string{"region"},
		),
	}

	reg.MustRegister(m.ProxyErrorsTotal)
	reg.MustRegister(m.RequestsTotal)
	reg.MustRegister(m.RequestDuration)
	reg.MustRegister(m.ActiveRequests)

	return m
}

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
		if errors.Is(netErr.Err, syscall.EHOSTUNREACH) {
			return "service_unavailable"
		}
	}

	var dnsErr *net.DNSError
	if errors.As(err, &dnsErr) {
		return "dns_failure"
	}

	return "other"
}
