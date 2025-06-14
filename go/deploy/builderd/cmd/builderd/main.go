package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/builderd/gen/proto/builder/v1/builderv1connect"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// version is set at build time via ldflags
var version = "0.0.4"

// AIDEV-NOTE: Enhanced version management with debug.ReadBuildInfo fallback
// Handles production builds (ldflags), development builds (git commit), and module builds
// getVersion returns the version string, with fallback to debug.ReadBuildInfo
func getVersion() string {
	// If version was set via ldflags (production builds), use it
	if version != "" && version != "0.0.3" {
		return version
	}

	// Fallback to debug.ReadBuildInfo for development/module builds
	if info, ok := debug.ReadBuildInfo(); ok {
		// Use the module version if available
		if info.Main.Version != "(devel)" && info.Main.Version != "" {
			return info.Main.Version
		}

		// Try to get version from VCS info
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && len(setting.Value) >= 7 {
				return "dev-" + setting.Value[:7] // First 7 chars of commit hash
			}
		}

		// Last resort: indicate it's a development build
		return "dev"
	}

	// Final fallback
	return version
}

func main() {
	// Track application start time for uptime calculations
	startTime := time.Now()
	
	// Create root context for coordinated shutdown
	rootCtx, rootCancel := context.WithCancel(context.Background())
	defer rootCancel()
	
	// Atomic state tracking for shutdown coordination
	var (
		shutdownStarted int64
		shutdownMutex   sync.Mutex
	)

	// Parse command-line flags
	var (
		showHelp    = flag.Bool("help", false, "Show help information")
		showVersion = flag.Bool("version", false, "Show version information")
	)
	flag.Parse()

	// Handle help and version flags
	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	if *showVersion {
		printVersion()
		os.Exit(0)
	}

	// Initialize structured logger with JSON output
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Log startup
	logger.Info("starting builderd service",
		slog.String("version", getVersion()),
		slog.String("go_version", runtime.Version()),
	)

	// Load configuration
	cfg, err := config.LoadConfigWithLogger(logger)
	if err != nil {
		logger.Error("failed to load configuration",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	
	// Configuration validation handled in LoadConfig

	logger.Info("configuration loaded",
		slog.String("address", cfg.Server.Address),
		slog.String("port", cfg.Server.Port),
		slog.String("storage_backend", cfg.Storage.Backend),
		slog.Bool("otel_enabled", cfg.OpenTelemetry.Enabled),
		slog.Bool("tenant_isolation", cfg.Tenant.IsolationEnabled),
		slog.Int("max_concurrent_builds", cfg.Builder.MaxConcurrentBuilds),
	)

	// Initialize OpenTelemetry with root context
	providers, err := observability.InitProviders(rootCtx, cfg, getVersion())
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	// OpenTelemetry shutdown will be handled in coordinated shutdown

	if cfg.OpenTelemetry.Enabled {
		logger.Info("OpenTelemetry initialized",
			slog.String("service_name", cfg.OpenTelemetry.ServiceName),
			slog.String("service_version", cfg.OpenTelemetry.ServiceVersion),
			slog.Float64("sampling_rate", cfg.OpenTelemetry.TracingSamplingRate),
			slog.String("otlp_endpoint", cfg.OpenTelemetry.OTLPEndpoint),
			slog.Bool("prometheus_enabled", cfg.OpenTelemetry.PrometheusEnabled),
			slog.Bool("high_cardinality_enabled", cfg.OpenTelemetry.HighCardinalityLabelsEnabled),
		)
	}

	// Initialize build metrics if OpenTelemetry is enabled
	var buildMetrics *observability.BuildMetrics
	if cfg.OpenTelemetry.Enabled {
		var err error
		buildMetrics, err = observability.NewBuildMetrics(logger, cfg.OpenTelemetry.HighCardinalityLabelsEnabled)
		if err != nil {
			logger.Warn("failed to initialize build metrics, entering degraded mode",
				slog.String("error", err.Error()),
			)
			// Continue without metrics rather than failing completely
		} else {
			logger.Info("build metrics initialized",
				slog.Bool("high_cardinality_enabled", cfg.OpenTelemetry.HighCardinalityLabelsEnabled),
			)
		}
	}

	// TODO: Initialize database
	// TODO: Initialize storage backend
	// TODO: Initialize Docker client
	// TODO: Initialize tenant manager
	// TODO: Initialize build executor registry

	// Create builder service
	builderService := service.NewBuilderService(logger, buildMetrics, cfg)

	// Create ConnectRPC handler with interceptors
	interceptors := []connect.Interceptor{
		observability.NewTenantAuthInterceptor(logger), // Authentication must come first
		observability.NewLoggingInterceptor(logger),
	}

	// Add OTEL interceptor if enabled
	if cfg.OpenTelemetry.Enabled {
		interceptors = append(interceptors, observability.NewOTELInterceptor())
	}

	mux := http.NewServeMux()
	path, handler := builderv1connect.NewBuilderServiceHandler(builderService,
		connect.WithInterceptors(interceptors...),
	)
	mux.Handle(path, handler)

	// Add rate-limited health check endpoint with proper JSON
	healthHandler := newRateLimitedHandler(createHealthHandler(startTime, logger), cfg.Server.RateLimit)
	mux.Handle("/health", healthHandler)

	// Stats are available through proper Prometheus metrics at /metrics

	// Add Prometheus metrics endpoint if enabled
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		mux.Handle("/metrics", providers.PrometheusHTTP)
		logger.Info("Prometheus metrics endpoint enabled",
			slog.String("path", "/metrics"),
		)
	}

	// Create HTTP server address
	addr := cfg.Server.Address + ":" + cfg.Server.Port
	
	// Service health validation after initialization
	if err := validateServiceHealth(logger, cfg, buildMetrics); err != nil {
		logger.Error("service health validation failed",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Wrap handler with OTEL HTTP middleware if enabled
	var httpHandler http.Handler = mux
	if cfg.OpenTelemetry.Enabled {
		httpHandler = otelhttp.NewHandler(mux, "http",
			otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
				return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
			}),
		)
	}

	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}),
	}

	// Use errgroup for coordinated goroutine management
	g, gCtx := errgroup.WithContext(rootCtx)
	
	// Start main server with proper error coordination
	g.Go(func() error {
		logger.Info("starting http server",
			slog.String("address", addr),
		)
		
		// Start server in a way that respects context cancellation
		errCh := make(chan error, 1)
		go func() {
			errCh <- server.ListenAndServe()
		}()
		
		select {
		case err := <-errCh:
			if err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("server failed: %w", err)
			}
			return nil
		case <-gCtx.Done():
			return gCtx.Err()
		}
	})

	// Note: Prometheus metrics are exposed on the main port at /metrics
	// No need for a separate Prometheus server

	// Implement proper signal handling with buffered channel
	sigChan := make(chan os.Signal, 2) // Buffer for multiple signals
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)
	
	// Handle shutdown coordination
	g.Go(func() error {
		select {
		case sig := <-sigChan:
			logger.Info("received shutdown signal",
				slog.String("signal", sig.String()),
			)
			return fmt.Errorf("shutdown signal received: %s", sig)
		case <-gCtx.Done():
			return gCtx.Err()
		}
	})
	
	// Wait for any goroutine to complete/fail
	if err := g.Wait(); err != nil {
		logger.Info("initiating graceful shutdown",
			slog.String("reason", err.Error()),
		)
	}
	
	// Coordinated shutdown with proper ordering
	performGracefulShutdown(logger, server, providers, &shutdownStarted, &shutdownMutex, cfg.Server.ShutdownTimeout)
}

// printUsage displays help information
func printUsage() {
	fmt.Printf("Builderd - Multi-Tenant Build Service\n\n")
	fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nEnvironment Variables:\n")
	fmt.Printf("  UNKEY_BUILDERD_PORT                         Server port (default: 8082)\n")
	fmt.Printf("  UNKEY_BUILDERD_ADDRESS                      Bind address (default: 0.0.0.0)\n")
	fmt.Printf("  UNKEY_BUILDERD_SHUTDOWN_TIMEOUT             Graceful shutdown timeout (default: 15s)\n")
	fmt.Printf("  UNKEY_BUILDERD_RATE_LIMIT                   Health endpoint rate limit/sec (default: 100)\n")
	fmt.Printf("  UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS        Max concurrent builds (default: 5)\n")
	fmt.Printf("  UNKEY_BUILDERD_BUILD_TIMEOUT                Build timeout (default: 15m)\n")
	fmt.Printf("  UNKEY_BUILDERD_STORAGE_BACKEND              Storage backend (local, s3, gcs)\n")
	fmt.Printf("  UNKEY_BUILDERD_STORAGE_RETENTION_DAYS       Storage retention days (default: 30)\n")
	fmt.Printf("  UNKEY_BUILDERD_DOCKER_MAX_IMAGE_SIZE_GB     Max Docker image size (default: 5)\n")
	fmt.Printf("  UNKEY_BUILDERD_TENANT_ISOLATION_ENABLED     Enable tenant isolation (default: true)\n")
	fmt.Printf("\nDatabase Configuration:\n")
	fmt.Printf("  UNKEY_BUILDERD_DATABASE_TYPE                Database type (default: sqlite)\n")
	fmt.Printf("  UNKEY_BUILDERD_DATABASE_DATA_DIR            SQLite data directory (default: /opt/builderd/data)\n")
	fmt.Printf("\nOpenTelemetry Configuration:\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_ENABLED                 Enable OpenTelemetry (default: false)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_SERVICE_NAME            Service name (default: builderd)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_SERVICE_VERSION         Service version (default: 0.0.1)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_SAMPLING_RATE           Trace sampling rate 0.0-1.0 (default: 1.0)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_ENDPOINT                OTLP endpoint (default: localhost:4318)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_PROMETHEUS_ENABLED      Enable Prometheus metrics (default: true)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_HIGH_CARDINALITY_ENABLED  Enable high-cardinality labels (default: false)\n")
	fmt.Printf("\nDescription:\n")
	fmt.Printf("  Builderd processes various source types (Docker images, Git repositories,\n")
	fmt.Printf("  archives) and produces optimized rootfs images for microVM execution.\n")
	fmt.Printf("  It supports multi-tenant isolation, resource quotas, and comprehensive\n")
	fmt.Printf("  observability with OpenTelemetry.\n\n")
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  /builder.v1.BuilderService/*  - ConnectRPC builder service\n")
	fmt.Printf("  /health                       - Health check endpoint (rate limited)\n")
	fmt.Printf("  /metrics                      - Prometheus metrics (if enabled)\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s                                    # Default settings (port 8082)\n", os.Args[0])
	fmt.Printf("  UNKEY_BUILDERD_OTEL_ENABLED=true %s        # Enable telemetry\n", os.Args[0])
	fmt.Printf("  UNKEY_BUILDERD_MAX_CONCURRENT_BUILDS=10 %s # Allow 10 concurrent builds\n", os.Args[0])
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("Builderd - Multi-Tenant Build Service\n")
	fmt.Printf("Version: %s\n", getVersion())
	fmt.Printf("Built with: %s\n", runtime.Version())
}

// Additional configuration validation (beyond config package)
func validateConfiguration(cfg *config.Config) error {
	// Additional runtime-specific validations can be added here
	// Core validations are handled in the config package
	return nil
}

// Rate-limited handler using token bucket algorithm for better efficiency
type rateLimitedHandler struct {
	handler http.Handler
	limiter *rate.Limiter
	logger  *slog.Logger
}

func newRateLimitedHandler(handler http.Handler, rateLimit int) *rateLimitedHandler {
	// Allow burst of 10 requests, then limit to rateLimit per second
	return &rateLimitedHandler{
		handler: handler,
		limiter: rate.NewLimiter(rate.Limit(rateLimit), 10), // 10 request burst
	}
}

func (rl *rateLimitedHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !rl.limiter.Allow() {
		// Rate limit exceeded
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%v", rl.limiter.Limit()))
		w.Header().Set("Retry-After", "1")
		w.WriteHeader(http.StatusTooManyRequests)
		w.Write([]byte(`{"error":"rate limit exceeded","status":429}`))
		return
	}
	rl.handler.ServeHTTP(w, r)
}

// Cached health check handler to prevent memory allocation on each request
func createHealthHandler(startTime time.Time, logger *slog.Logger) http.Handler {
	type healthResponse struct {
		Status        string  `json:"status"`
		Service       string  `json:"service"`
		Version       string  `json:"version"`
		UptimeSeconds float64 `json:"uptime_seconds"`
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Create response struct instead of string formatting
		response := healthResponse{
			Status:        "ok",
			Service:       "builderd",
			Version:       getVersion(),
			UptimeSeconds: time.Since(startTime).Seconds(),
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		
		// Use json.Encoder for better performance
		if err := json.NewEncoder(w).Encode(response); err != nil {
			logger.Error("failed to encode health response",
				slog.String("error", err.Error()),
			)
		}
	})
}

// Service health validation function
func validateServiceHealth(logger *slog.Logger, cfg *config.Config, buildMetrics *observability.BuildMetrics) error {
	// Validate critical configuration
	if cfg.Server.Port == "" {
		return fmt.Errorf("server port not configured")
	}
	
	if cfg.Builder.MaxConcurrentBuilds <= 0 {
		return fmt.Errorf("invalid max concurrent builds: %d", cfg.Builder.MaxConcurrentBuilds)
	}
	
	if cfg.Server.ShutdownTimeout <= 0 {
		return fmt.Errorf("invalid shutdown timeout: %v", cfg.Server.ShutdownTimeout)
	}
	
	if cfg.Server.RateLimit <= 0 {
		return fmt.Errorf("invalid rate limit: %d", cfg.Server.RateLimit)
	}
	
	// Check if required directories are accessible
	requiredDirs := []string{
		cfg.Builder.ScratchDir,
		cfg.Builder.RootfsOutputDir,
		cfg.Builder.WorkspaceDir,
	}
	
	for _, dir := range requiredDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("cannot create/access directory %s: %w", dir, err)
		}
	}
	
	logger.Info("service health validation passed",
		slog.String("status", "healthy"),
		slog.Bool("metrics_available", buildMetrics != nil),
		slog.Int("rate_limit", cfg.Server.RateLimit),
		slog.Duration("shutdown_timeout", cfg.Server.ShutdownTimeout),
	)
	
	return nil
}

// Coordinated graceful shutdown function
func performGracefulShutdown(logger *slog.Logger, server *http.Server, providers *observability.Providers, shutdownStarted *int64, shutdownMutex *sync.Mutex, shutdownTimeout time.Duration) {
	// Ensure shutdown only happens once
	if !atomic.CompareAndSwapInt64(shutdownStarted, 0, 1) {
		logger.Warn("shutdown already in progress")
		return
	}

	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()

	logger.Info("performing graceful shutdown")

	// Create shutdown context with configurable timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	// Use errgroup for coordinated shutdown
	g, gCtx := errgroup.WithContext(shutdownCtx)

	// Shutdown HTTP server
	g.Go(func() error {
		logger.Info("shutting down HTTP server")
		if err := server.Shutdown(gCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown failed: %w", err)
		}
		logger.Info("HTTP server shutdown complete")
		return nil
	})

	// Shutdown OpenTelemetry providers
	if providers != nil {
		g.Go(func() error {
			logger.Info("shutting down OpenTelemetry providers")
			if err := providers.Shutdown(gCtx); err != nil {
				return fmt.Errorf("OpenTelemetry shutdown failed: %w", err)
			}
			logger.Info("OpenTelemetry shutdown complete")
			return nil
		})
	}

	// Wait for all shutdown operations to complete
	if err := g.Wait(); err != nil {
		logger.Error("graceful shutdown completed with errors",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	logger.Info("graceful shutdown completed successfully")
}
