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
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/go/apps/billaged/internal/aggregator"
	"github.com/unkeyed/unkey/go/apps/billaged/internal/config"
	"github.com/unkeyed/unkey/go/apps/billaged/internal/observability"
	"github.com/unkeyed/unkey/go/apps/billaged/internal/service"
	healthpkg "github.com/unkeyed/unkey/go/deploy/pkg/health"
	"github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
	"github.com/unkeyed/unkey/go/gen/proto/billaged/v1/billagedv1connect"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
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

	// Parse command-line flags with environment variable fallbacks
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

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{ //nolint:exhaustruct // AddSource and ReplaceAttr use appropriate default values
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load configuration
	cfg, err := config.LoadConfigWithLogger(logger)
	if err != nil {
		logger.Error("failed to load configuration",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Parse aggregation interval from config
	aggregationInterval, err := time.ParseDuration(cfg.Aggregation.Interval)
	if err != nil {
		logger.Error("invalid aggregation interval",
			slog.String("interval", cfg.Aggregation.Interval),
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	// Log startup
	logger.Info("starting billaged service",
		slog.String("version", getVersion()),
		slog.String("go_version", runtime.Version()),
		slog.String("port", cfg.Server.Port),
		slog.String("address", cfg.Server.Address),
		slog.String("aggregation_interval", aggregationInterval.String()),
		slog.Bool("otel_enabled", cfg.OpenTelemetry.Enabled),
	)

	// Initialize TLS provider (defaults to disabled)
	ctx := context.Background()
	tlsConfig := tlspkg.Config{ //nolint:exhaustruct // Optional TLS fields use appropriate default values
		Mode:             tlspkg.Mode(cfg.TLS.Mode),
		CertFile:         cfg.TLS.CertFile,
		KeyFile:          cfg.TLS.KeyFile,
		CAFile:           cfg.TLS.CAFile,
		SPIFFESocketPath: cfg.TLS.SPIFFESocketPath,
	}
	tlsProvider, err := tlspkg.NewProvider(ctx, tlsConfig)
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

	// Initialize OpenTelemetry
	otelProviders, err := observability.InitProviders(ctx, cfg, getVersion())
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := otelProviders.Shutdown(shutdownCtx); err != nil {
			logger.Error("failed to shutdown OpenTelemetry",
				slog.String("error", err.Error()),
			)
		}
	}()

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

	// Initialize billing metrics if OpenTelemetry is enabled
	var billingMetrics *observability.BillingMetrics
	if cfg.OpenTelemetry.Enabled {
		var err error
		billingMetrics, err = observability.NewBillingMetrics(logger, cfg.OpenTelemetry.HighCardinalityLabelsEnabled)
		if err != nil {
			logger.Error("failed to initialize billing metrics",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		logger.Info("billing metrics initialized",
			slog.Bool("high_cardinality_enabled", cfg.OpenTelemetry.HighCardinalityLabelsEnabled),
		)
	}

	// Create aggregator with usage summary callback
	agg := aggregator.NewAggregator(logger, aggregationInterval)

	// Set up usage summary callback to print results
	agg.SetUsageSummaryCallback(func(summary *aggregator.UsageSummary) {
		printUsageSummary(logger, summary)
	})

	// Create billing service
	billingService := service.NewBillingService(logger, agg, billingMetrics)

	// Configure shared interceptor options
	interceptorOpts := []interceptors.Option{
		interceptors.WithServiceName("billaged"),
		interceptors.WithLogger(logger),
		interceptors.WithActiveRequestsMetric(true),
		interceptors.WithRequestDurationMetric(false), // Match existing behavior
		interceptors.WithErrorResampling(true),
		interceptors.WithPanicStackTrace(true),
	}

	// Add meter if OpenTelemetry is enabled
	if cfg.OpenTelemetry.Enabled {
		interceptorOpts = append(interceptorOpts, interceptors.WithMeter(otel.Meter("billaged")))
	}

	// Get default interceptors (tenant auth, metrics, logging)
	sharedInterceptors := interceptors.NewDefaultInterceptors("billaged", interceptorOpts...)

	// Convert UnaryInterceptorFunc to Interceptor
	var interceptorList []connect.Interceptor
	for _, interceptor := range sharedInterceptors {
		interceptorList = append(interceptorList, connect.Interceptor(interceptor))
	}

	mux := http.NewServeMux()
	path, handler := billagedv1connect.NewBillingServiceHandler(billingService,
		connect.WithInterceptors(interceptorList...),
	)
	mux.Handle(path, handler)

	// Add stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		activeVMs := agg.GetActiveVMCount()

		response := fmt.Sprintf(`{
			"active_vms": %d,
			"aggregation_interval": "%s"
		}`, activeVMs, aggregationInterval.String())

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(response))
	})

	// Add Prometheus metrics endpoint if enabled
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		mux.Handle("/metrics", otelProviders.PrometheusHTTP)
		logger.Info("Prometheus metrics endpoint enabled",
			slog.String("path", "/metrics"),
		)
	}

	// Create HTTP server with H2C support
	serverAddr := fmt.Sprintf("%s:%s", cfg.Server.Address, cfg.Server.Port)

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
		Addr:    serverAddr,
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}), //nolint:exhaustruct // Using default HTTP/2 server configuration
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

	// Start periodic aggregation
	aggCtx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go agg.StartPeriodicAggregation(aggCtx)

	// Start Prometheus server on separate port if enabled
	var promServer *http.Server
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		// AIDEV-NOTE: Use configured interface, defaulting to localhost for security
		promAddr := fmt.Sprintf("%s:%s", cfg.OpenTelemetry.PrometheusInterface, cfg.OpenTelemetry.PrometheusPort)
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())
		promMux.HandleFunc("/health", healthpkg.Handler("billaged", getVersion(), startTime))

		promServer = &http.Server{
			Addr:         promAddr,
			Handler:      promMux,
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
		}

		go func() {
			localhostOnly := cfg.OpenTelemetry.PrometheusInterface == "127.0.0.1" || cfg.OpenTelemetry.PrometheusInterface == "localhost"
			logger.Info("starting prometheus metrics server",
				slog.String("address", promAddr),
				slog.Bool("localhost_only", localhostOnly),
			)
			if err := promServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("prometheus server failed",
					slog.String("error", err.Error()),
				)
			}
		}()
	}

	// Start main server in goroutine
	go func() {
		if serverTLSConfig != nil {
			logger.Info("starting HTTPS server with TLS",
				slog.String("address", serverAddr),
				slog.String("tls_mode", cfg.TLS.Mode),
			)
			// Empty strings for cert/key paths - SPIFFE provides them in memory
			if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logger.Error("server failed",
					slog.String("error", err.Error()),
				)
				os.Exit(1)
			}
		} else {
			logger.Info("starting HTTP server without TLS",
				slog.String("address", serverAddr),
			)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("server failed",
					slog.String("error", err.Error()),
				)
				os.Exit(1)
			}
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down billaged service")

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	cancel() // Stop aggregation

	// Shutdown all servers
	var shutdownErrors []error

	if err := server.Shutdown(shutdownCtx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("main server: %w", err))
	}

	if promServer != nil {
		if err := promServer.Shutdown(shutdownCtx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("prometheus server: %w", err))
		}
	}

	if len(shutdownErrors) > 0 {
		logger.Error("failed to shutdown servers gracefully",
			slog.String("error", fmt.Sprintf("%v", shutdownErrors)),
		)
		os.Exit(1)
	}

	logger.Info("billaged service shutdown complete")
}

// printUsageSummary prints the aggregated usage summary every 60 seconds
func printUsageSummary(logger *slog.Logger, summary *aggregator.UsageSummary) {
	logger.Info("=== BILLAGED USAGE SUMMARY ===",
		"vm_id", summary.VMID,
		"customer_id", summary.CustomerID,
		"period", summary.Period.String(),
		"start_time", summary.StartTime.Format("15:04:05"),
		"end_time", summary.EndTime.Format("15:04:05"),
	)

	logger.Info("CPU USAGE",
		"vm_id", summary.VMID,
		"cpu_time_used_ms", summary.CPUTimeUsedMs,
		"cpu_time_used_seconds", float64(summary.CPUTimeUsedMs)/1000.0,
	)

	logger.Info("MEMORY USAGE",
		"vm_id", summary.VMID,
		"avg_memory_usage_mb", summary.AvgMemoryUsageBytes/(1024*1024),
		"max_memory_usage_mb", summary.MaxMemoryUsageBytes/(1024*1024),
	)

	logger.Info("DISK I/O",
		"vm_id", summary.VMID,
		"disk_read_mb", summary.DiskReadBytes/(1024*1024),
		"disk_write_mb", summary.DiskWriteBytes/(1024*1024),
		"total_disk_io_mb", summary.TotalDiskIO/(1024*1024),
	)

	logger.Info("NETWORK I/O",
		"vm_id", summary.VMID,
		"network_rx_mb", summary.NetworkRxBytes/(1024*1024),
		"network_tx_mb", summary.NetworkTxBytes/(1024*1024),
		"total_network_io_mb", summary.TotalNetworkIO/(1024*1024),
	)

	logger.Info("BILLING METRICS",
		"vm_id", summary.VMID,
		"resource_score", fmt.Sprintf("%.2f", summary.ResourceScore),
		"sample_count", summary.SampleCount,
	)

	logger.Info("=== END USAGE SUMMARY ===",
		"vm_id", summary.VMID,
	)
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("Billaged - VM Usage Billing Aggregation Service\n")
	fmt.Printf("Version: %s\n", getVersion())
	fmt.Printf("Built with: %s\n", runtime.Version())
}

// printUsage displays help information
func printUsage() {
	fmt.Printf("Billaged - VM Usage Billing Aggregation Service\n\n")
	fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nEnvironment Variables:\n")
	fmt.Printf("  UNKEY_BILLAGED_PORT                         Server port (default: 8081)\n")
	fmt.Printf("  UNKEY_BILLAGED_ADDRESS                      Bind address (default: 0.0.0.0)\n")
	fmt.Printf("  UNKEY_BILLAGED_AGGREGATION_INTERVAL         Aggregation interval (default: 60s)\n")
	fmt.Printf("\nOpenTelemetry Configuration:\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_ENABLED                 Enable OpenTelemetry (default: false)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_SERVICE_NAME            Service name (default: billaged)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_SERVICE_VERSION         Service version (default: 0.0.1)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_SAMPLING_RATE           Trace sampling rate 0.0-1.0 (default: 1.0)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_ENDPOINT                OTLP endpoint (default: localhost:4318)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_PROMETHEUS_ENABLED      Enable Prometheus metrics (default: true)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_PROMETHEUS_PORT         Prometheus metrics port (default: 9465)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_PROMETHEUS_INTERFACE    Prometheus binding interface (default: 127.0.0.1)\n")
	fmt.Printf("  UNKEY_BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED  Enable high-cardinality labels (default: false)\n")
	fmt.Printf("\nTLS Configuration:\n")
	fmt.Printf("  UNKEY_BILLAGED_TLS_MODE               TLS mode: disabled, file, spiffe (default: disabled)\n")
	fmt.Printf("  UNKEY_BILLAGED_TLS_CERT_FILE          Path to certificate file (file mode)\n")
	fmt.Printf("  UNKEY_BILLAGED_TLS_KEY_FILE           Path to private key file (file mode)\n")
	fmt.Printf("  UNKEY_BILLAGED_TLS_CA_FILE            Path to CA bundle file (file mode)\n")
	fmt.Printf("  UNKEY_BILLAGED_SPIFFE_SOCKET          SPIFFE workload API socket (default: /run/spire/sockets/agent.sock)\n")
	fmt.Printf("\nDescription:\n")
	fmt.Printf("  Billaged receives VM usage metrics from metald instances and aggregates\n")
	fmt.Printf("  them for billing purposes. It calculates usage summaries every 60 seconds\n")
	fmt.Printf("  (configurable) and can track multiple VMs across multiple customers.\n\n")
	fmt.Printf("Endpoints:\n")
	fmt.Printf("  /billing.v1.BillingService/*  - ConnectRPC billing service\n")
	fmt.Printf("  /health                       - Health check endpoint\n")
	fmt.Printf("  /stats                        - Current statistics\n")
	fmt.Printf("  /metrics                      - Prometheus metrics (if enabled)\n\n")
	fmt.Printf("Examples:\n")
	fmt.Printf("  %s                                    # Default settings (port 8081)\n", os.Args[0])
	fmt.Printf("  UNKEY_BILLAGED_OTEL_ENABLED=true %s        # Enable telemetry\n", os.Args[0])
	fmt.Printf("  UNKEY_BILLAGED_AGGREGATION_INTERVAL=30s %s # 30-second summaries\n", os.Args[0])
}
