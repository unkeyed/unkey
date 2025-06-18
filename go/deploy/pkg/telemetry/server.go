package telemetry

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// MetricsServerConfig holds configuration for the metrics server
type MetricsServerConfig struct {
	Interface      string
	Port           string
	HealthHandler  http.HandlerFunc
	MetricsHandler http.Handler
	Logger         *slog.Logger
}

// NewMetricsServer creates a new HTTP server for Prometheus metrics and health checks
// This server runs on a separate port without TLS for monitoring purposes
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

// StartMetricsServer starts the metrics server in a goroutine
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