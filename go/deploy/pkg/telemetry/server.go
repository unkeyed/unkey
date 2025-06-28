package telemetry

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServerConfig holds configuration for the Prometheus metrics HTTP server.
//
// The metrics server runs separately from the main application server to provide
// monitoring endpoints without requiring authentication or TLS.
type MetricsServerConfig struct {
	// Interface specifies the network interface to bind to (e.g., "127.0.0.1", "0.0.0.0").
	Interface string
	// Port specifies the TCP port for the metrics server.
	Port string
	// HealthHandler provides the /health endpoint handler. Optional.
	HealthHandler http.HandlerFunc
	// MetricsHandler provides the /metrics endpoint handler. Defaults to promhttp.Handler().
	MetricsHandler http.Handler
	// Logger is used for server lifecycle and error logging.
	Logger *slog.Logger
}

// NewMetricsServer creates a new HTTP server for Prometheus metrics and health checks.
//
// The server exposes /metrics for Prometheus scraping and optionally /health for
// health checks. It runs without TLS and uses conservative timeout settings
// suitable for monitoring workloads.
func NewMetricsServer(cfg *MetricsServerConfig) *http.Server {
	mux := http.NewServeMux()

	// Use provided metrics handler or default to promhttp
	metricsHandler := cfg.MetricsHandler
	if metricsHandler == nil {
		metricsHandler = promhttp.Handler()
	}
	mux.Handle("/metrics", metricsHandler)

	// Add health endpoint if handler provided
	if cfg.HealthHandler != nil {
		mux.HandleFunc("/health", cfg.HealthHandler)
	}

	addr := fmt.Sprintf("%s:%s", cfg.Interface, cfg.Port)

	return &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

// StartMetricsServer starts the metrics server in a background goroutine.
//
// The server begins listening immediately and logs startup information including
// the bound address and whether it's restricted to localhost. Server errors
// are logged but do not cause the function to return an error.
func StartMetricsServer(cfg *MetricsServerConfig) {
	server := NewMetricsServer(cfg)

	go func() {
		localhostOnly := cfg.Interface == "127.0.0.1" || cfg.Interface == "localhost"
		cfg.Logger.Info("starting prometheus metrics server",
			slog.String("address", server.Addr),
			slog.Bool("localhost_only", localhostOnly),
		)

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			cfg.Logger.Error("prometheus server failed",
				slog.String("error", err.Error()),
			)
		}
	}()
}
