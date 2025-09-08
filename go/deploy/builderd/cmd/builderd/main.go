package main

import (
	"context"
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
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/assetmanager"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/assets"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/builderd/internal/service"
	healthpkg "github.com/unkeyed/unkey/go/deploy/pkg/health"
	"github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
	"github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1/builderdv1connect"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
	"golang.org/x/sync/errgroup"
	"golang.org/x/time/rate"
)

// version is set at build time via ldflags
var version = ""

// AIDEV-NOTE: Enhanced version management with debug.ReadBuildInfo fallback
// Handles production builds (ldflags), development builds (git commit), and module builds
// getVersion returns the version string, with fallback to debug.ReadBuildInfo
func getVersion() string {
	// If version was set via ldflags (production builds), use it
	if version != "" {
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
	//nolint:exhaustruct // Only Level field is needed for handler options
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

	// Initialize TLS provider (defaults to disabled)
	//nolint:exhaustruct // Only specified TLS fields are needed for this configuration
	tlsConfig := tlspkg.Config{
		Mode:             tlspkg.Mode(cfg.TLS.Mode),
		CertFile:         cfg.TLS.CertFile,
		KeyFile:          cfg.TLS.KeyFile,
		CAFile:           cfg.TLS.CAFile,
		SPIFFESocketPath: cfg.TLS.SPIFFESocketPath,
	}
	tlsProvider, err := tlspkg.NewProvider(rootCtx, tlsConfig)
	if err != nil {
		// AIDEV-NOTE: TLS/SPIFFE is now required - no fallback to disabled mode
		logger.Error("TLS initialization failed",
			"error", err,
			"mode", cfg.TLS.Mode)
		os.Exit(1)
	}
	defer tlsProvider.Close()

	logger.Info("TLS provider initialized",
		"mode", cfg.TLS.Mode,
		"spiffe_enabled", cfg.TLS.Mode == "spiffe")

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

	// Initialize assetmanagerd client
	assetClient, err := assetmanager.NewClient(cfg, logger, tlsProvider)
	if err != nil {
		logger.Error("failed to initialize assetmanagerd client",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Initialize base assets (kernel, rootfs) required for VM creation
	// AIDEV-NOTE: This ensures builderd can create VMs without external setup scripts
	if cfg.AssetManager.Enabled {
		logger.Info("initializing base VM assets")
		baseAssetInitCtx, cancel := context.WithTimeout(rootCtx, 10*time.Minute)
		defer cancel()

		// Use proper assets package with metrics and retry logic
		assetManager := assets.NewBaseAssetManager(logger, cfg, assetClient)
		if buildMetrics != nil {
			assetManager = assetManager.WithMetrics(buildMetrics)
		}
		if err := assetManager.InitializeBaseAssetsWithRetry(baseAssetInitCtx); err != nil {
			logger.Error("failed to initialize base assets",
				slog.String("error", err.Error()),
			)
			// Don't exit - continue with degraded functionality
			logger.Warn("continuing with degraded functionality - base assets may not be available")
		} else {
			logger.Info("base assets initialization completed")
		}
	}

	// Create builder service
	builderService := service.NewBuilderService(logger, buildMetrics, cfg, assetClient)

	// Configure shared interceptor options
	interceptorOpts := []interceptors.Option{
		interceptors.WithServiceName("builderd"),
		interceptors.WithLogger(logger),
		interceptors.WithActiveRequestsMetric(true),
		interceptors.WithRequestDurationMetric(false), // Match existing behavior
		interceptors.WithErrorResampling(true),
		interceptors.WithPanicStackTrace(true),
		interceptors.WithTenantAuth(true,
			// Exempt health check endpoints from tenant auth
			"/health.v1.HealthService/Check",
			// Exempt admin/stats endpoints from tenant auth
			"/builder.v1.BuilderService/GetBuildStats",
		),
	}

	// Add meter if OpenTelemetry is enabled
	if cfg.OpenTelemetry.Enabled {
		interceptorOpts = append(interceptorOpts, interceptors.WithMeter(otel.Meter("builderd")))
	}

	// Get default interceptors (tenant auth, metrics, logging)
	sharedInterceptors := interceptors.NewDefaultInterceptors("builderd", interceptorOpts...)

	// Convert UnaryInterceptorFunc to Interceptor
	var interceptorList []connect.Interceptor
	for _, interceptor := range sharedInterceptors {
		interceptorList = append(interceptorList, connect.Interceptor(interceptor))
	}

	mux := http.NewServeMux()
	path, handler := builderdv1connect.NewBuilderServiceHandler(builderService,
		connect.WithInterceptors(interceptorList...),
	)
	mux.Handle(path, handler)

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

	// Configure server with optional TLS and security timeouts
	server := &http.Server{
		Addr: addr,
		//nolint:exhaustruct // Default http2.Server configuration is sufficient
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}),
		// AIDEV-NOTE: Security timeouts to prevent slowloris attacks
		ReadTimeout:    30 * time.Second,  // Time to read request headers
		WriteTimeout:   30 * time.Second,  // Time to write response
		IdleTimeout:    120 * time.Second, // Keep-alive timeout
		MaxHeaderBytes: 1 << 20,           // 1MB max header size
	}

	// Apply TLS configuration if enabled
	serverTLSConfig, _ := tlsProvider.ServerTLSConfig()
	if serverTLSConfig != nil {
		server.TLSConfig = serverTLSConfig
		// For TLS, we need to use regular handler, not h2c
		server.Handler = httpHandler
	}

	// Use errgroup for coordinated goroutine management
	g, gCtx := errgroup.WithContext(rootCtx)

	// Start main server with proper error coordination
	g.Go(func() error {
		// AIDEV-NOTE: Start server with proper context cancellation to prevent startup goroutine deadlock
		errCh := make(chan error, 1)

		if serverTLSConfig != nil {
			logger.Info("starting HTTPS server with TLS",
				slog.String("address", addr),
				slog.String("tls_mode", cfg.TLS.Mode),
			)
			go func() {
				// Empty strings for cert/key paths - SPIFFE provides them in memory
				errCh <- server.ListenAndServeTLS("", "")
			}()
		} else {
			logger.Info("starting HTTP server without TLS",
				slog.String("address", addr),
			)
			go func() {
				errCh <- server.ListenAndServe()
			}()
		}

		select {
		case err := <-errCh:
			if err != nil && err != http.ErrServerClosed {
				return fmt.Errorf("server failed: %w", err)
			}
			return nil
		case <-gCtx.Done():
			// AIDEV-NOTE: Immediately shutdown server when context is cancelled to prevent deadlock
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := server.Shutdown(shutdownCtx); err != nil {
				logger.Warn("server shutdown during startup failed", slog.String("error", err.Error()))
			}
			return gCtx.Err()
		}
	})

	// Start Prometheus server on separate port if enabled
	var promServer *http.Server
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		// AIDEV-NOTE: Use configured interface, defaulting to localhost for security
		promAddr := fmt.Sprintf("%s:%s", cfg.OpenTelemetry.PrometheusInterface, cfg.OpenTelemetry.PrometheusPort)
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())
		// Add rate-limited health check endpoint with unified handler
		healthHandler := newRateLimitedHandler(healthpkg.Handler("builderd", getVersion(), startTime), cfg.Server.RateLimit)
		promMux.Handle("/health", healthHandler)

		promServer = &http.Server{
			Addr:         promAddr,
			Handler:      promMux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		g.Go(func() error {
			localhostOnly := cfg.OpenTelemetry.PrometheusInterface == "127.0.0.1" || cfg.OpenTelemetry.PrometheusInterface == "localhost"
			logger.Info("starting prometheus metrics server",
				slog.String("address", promAddr),
				slog.Bool("localhost_only", localhostOnly),
			)

			// AIDEV-NOTE: Start Prometheus server with context cancellation support
			errCh := make(chan error, 1)
			go func() {
				errCh <- promServer.ListenAndServe()
			}()

			select {
			case err := <-errCh:
				if err != nil && err != http.ErrServerClosed {
					return fmt.Errorf("prometheus server failed: %w", err)
				}
				return nil
			case <-gCtx.Done():
				// AIDEV-NOTE: Immediately shutdown Prometheus server when context is cancelled
				shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := promServer.Shutdown(shutdownCtx); err != nil {
					logger.Warn("prometheus server shutdown during startup failed", slog.String("error", err.Error()))
				}
				return gCtx.Err()
			}
		})
	}

	// Implement proper signal handling with buffered channel
	sigChan := make(chan os.Signal, 2) // Buffer for multiple signals
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// AIDEV-NOTE: Signal handling continues during graceful shutdown to prevent SIGABRT panics
	shutdownSignalReceived := make(chan struct{})

	// Handle shutdown coordination
	g.Go(func() error {
		select {
		case sig := <-sigChan:
			logger.Info("received shutdown signal",
				slog.String("signal", sig.String()),
			)
			close(shutdownSignalReceived)
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

	// Continue handling signals during graceful shutdown to prevent SIGABRT panics
	go func() {
		for {
			select {
			case <-shutdownSignalReceived:
				// Already shutting down, ignore
				return
			case sig := <-sigChan:
				logger.Warn("received additional signal during shutdown, ignoring",
					slog.String("signal", sig.String()),
				)
				// Continue listening for more signals
			}
		}
	}()

	// Coordinated shutdown with proper ordering
	performGracefulShutdown(logger, server, promServer, providers, builderService, &shutdownStarted, &shutdownMutex, cfg.Server.ShutdownTimeout)
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
	fmt.Printf("  UNKEY_BUILDERD_OTEL_SERVICE_VERSION         Service version (default: 0.1.0)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_SAMPLING_RATE           Trace sampling rate 0.0-1.0 (default: 1.0)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_ENDPOINT                OTLP endpoint (default: localhost:4318)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_PROMETHEUS_ENABLED      Enable Prometheus metrics (default: true)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_PROMETHEUS_PORT         Prometheus metrics port (default: 9466)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_PROMETHEUS_INTERFACE    Prometheus binding interface (default: 127.0.0.1)\n")
	fmt.Printf("  UNKEY_BUILDERD_OTEL_HIGH_CARDINALITY_ENABLED  Enable high-cardinality labels (default: false)\n")
	fmt.Printf("\nTLS Configuration:\n")
	fmt.Printf("  UNKEY_BUILDERD_TLS_MODE                     TLS mode: disabled, file, spiffe (default: disabled)\n")
	fmt.Printf("  UNKEY_BUILDERD_TLS_CERT_FILE                Path to certificate file (file mode)\n")
	fmt.Printf("  UNKEY_BUILDERD_TLS_KEY_FILE                 Path to private key file (file mode)\n")
	fmt.Printf("  UNKEY_BUILDERD_TLS_CA_FILE                  Path to CA bundle file (file mode)\n")
	fmt.Printf("  UNKEY_BUILDERD_SPIFFE_SOCKET                SPIFFE workload API socket (default: /run/spire/sockets/agent.sock)\n")
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

// Rate-limited handler using token bucket algorithm for better efficiency
type rateLimitedHandler struct {
	handler http.Handler
	limiter *rate.Limiter
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
		_, _ = w.Write([]byte(`{"error":"rate limit exceeded","status":429}`))
		return
	}
	rl.handler.ServeHTTP(w, r)
}

// AIDEV-NOTE: Health handler removed - using unified health package instead

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
		if err := os.MkdirAll(dir, 0o755); err != nil {
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
func performGracefulShutdown(logger *slog.Logger, server *http.Server, promServer *http.Server, providers *observability.Providers, builderService *service.BuilderService, shutdownStarted *int64, shutdownMutex *sync.Mutex, shutdownTimeout time.Duration) {
	// Ensure shutdown only happens once
	if !atomic.CompareAndSwapInt64(shutdownStarted, 0, 1) {
		logger.Warn("shutdown already in progress")
		return
	}

	logger.Info("attempting to acquire shutdown mutex")
	shutdownMutex.Lock()
	defer shutdownMutex.Unlock()

	logger.Info("acquired shutdown mutex, performing graceful shutdown")

	// Create shutdown context with configurable timeout
	// AIDEV-NOTE: Use a shorter timeout to avoid systemd SIGABRT
	actualTimeout := shutdownTimeout
	if actualTimeout > 12*time.Second {
		actualTimeout = 12 * time.Second // Leave 3s buffer before systemd timeout
	}
	shutdownCtx, cancel := context.WithTimeout(context.Background(), actualTimeout)
	defer cancel()

	logger.Info("starting graceful shutdown with timeout",
		slog.Duration("timeout", actualTimeout),
	)

	// Use errgroup for coordinated shutdown
	g, gCtx := errgroup.WithContext(shutdownCtx)

	// AIDEV-NOTE: Shutdown BuilderService first to stop new builds and wait for running ones
	if builderService != nil {
		g.Go(func() error {
			logger.Info("starting BuilderService shutdown")
			if err := builderService.Shutdown(gCtx); err != nil {
				logger.Error("BuilderService shutdown failed", slog.String("error", err.Error()))
				return fmt.Errorf("BuilderService shutdown failed: %w", err)
			}
			logger.Info("BuilderService shutdown complete")
			return nil
		})
	}

	// Shutdown HTTP server
	if server != nil {
		g.Go(func() error {
			logger.Info("starting HTTP server shutdown")
			if err := server.Shutdown(gCtx); err != nil {
				logger.Error("HTTP server shutdown failed", slog.String("error", err.Error()))
				return fmt.Errorf("HTTP server shutdown failed: %w", err)
			}
			logger.Info("HTTP server shutdown complete")
			return nil
		})
	}

	// Shutdown Prometheus server if running
	if promServer != nil {
		g.Go(func() error {
			logger.Info("starting Prometheus server shutdown")
			if err := promServer.Shutdown(gCtx); err != nil {
				logger.Error("Prometheus server shutdown failed", slog.String("error", err.Error()))
				return fmt.Errorf("prometheus server shutdown failed: %w", err)
			}
			logger.Info("Prometheus server shutdown complete")
			return nil
		})
	}

	// Shutdown OpenTelemetry providers
	if providers != nil {
		g.Go(func() error {
			logger.Info("starting OpenTelemetry providers shutdown")
			if err := providers.Shutdown(gCtx); err != nil {
				logger.Error("OpenTelemetry shutdown failed", slog.String("error", err.Error()))
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
