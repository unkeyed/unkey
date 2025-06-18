package main

import (
	"context"
	"errors"
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

	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/assetmanager"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/firecracker"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/billing"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/config"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/database"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/network"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/service"
	healthpkg "github.com/unkeyed/unkey/go/deploy/pkg/health"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// version is set at build time via ldflags
var version = "0.2.0" // AIDEV-NOTE: Bumped minor version for integrated jailer feature

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
	//exhaustruct:ignore
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Log startup
	logger.Info("starting vmm control plane",
		slog.String("version", getVersion()),
		slog.String("go_version", runtime.Version()),
	)

	// Load configuration
	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Error("failed to load configuration",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}

	logger.Info("configuration loaded",
		slog.String("backend", string(cfg.Backend.Type)),
		slog.String("address", cfg.Server.Address),
		slog.String("port", cfg.Server.Port),
		slog.Bool("otel_enabled", cfg.OpenTelemetry.Enabled),
	)

	// Initialize OpenTelemetry
	ctx := context.Background()
	otelProviders, err := observability.InitProviders(ctx, cfg, getVersion(), logger)
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry",
			slog.String("error", err.Error()),
		)
		os.Exit(1)
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if shutdownErr := otelProviders.Shutdown(shutdownCtx); shutdownErr != nil {
			logger.Error("failed to shutdown OpenTelemetry",
				slog.String("error", shutdownErr.Error()),
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

	// Initialize TLS provider (defaults to disabled)
	//exhaustruct:ignore
	tlsConfig := tlspkg.Config{
		Mode:              tlspkg.Mode(cfg.TLS.Mode),
		CertFile:          cfg.TLS.CertFile,
		KeyFile:           cfg.TLS.KeyFile,
		CAFile:            cfg.TLS.CAFile,
		SPIFFESocketPath:  cfg.TLS.SPIFFESocketPath,
		EnableCertCaching: cfg.TLS.EnableCertCaching,
	}
	// Parse certificate cache TTL
	if cfg.TLS.CertCacheTTL != "" {
		if duration, parseErr := time.ParseDuration(cfg.TLS.CertCacheTTL); parseErr == nil {
			tlsConfig.CertCacheTTL = duration
		} else {
			logger.Warn("invalid TLS certificate cache TTL, using default 5s",
				"value", cfg.TLS.CertCacheTTL,
				"error", parseErr)
		}
	}
	tlsProvider, err := tlspkg.NewProvider(ctx, tlsConfig)
	if err != nil {
		// AIDEV-BUSINESS_RULE: TLS/SPIFFE is required - fatal error if it fails
		logger.Error("TLS initialization failed",
			"error", err,
			"mode", cfg.TLS.Mode)
		os.Exit(1)
	}
	defer tlsProvider.Close()

	logger.Info("TLS provider initialized",
		"mode", cfg.TLS.Mode,
		"spiffe_enabled", cfg.TLS.Mode == "spiffe")

	// Initialize database
	db, err := database.NewWithLogger(cfg.Database.DataDir, logger)
	if err != nil {
		logger.Error("failed to initialize database",
			slog.String("error", err.Error()),
			slog.String("data_dir", cfg.Database.DataDir),
		)
		os.Exit(1)
	}
	defer db.Close()

	// Create VM repository
	vmRepo := database.NewVMRepository(db)

	logger.Info("database initialized",
		slog.String("data_dir", cfg.Database.DataDir),
	)

	// Initialize backend based on configuration
	var backend types.Backend
	switch cfg.Backend.Type {
	case types.BackendTypeFirecracker:
		// Use SDK client v4 with integrated jailer - let SDK handle complete lifecycle
		// AIDEV-NOTE: SDK manages firecracker process, integrated jailer, and networking
		networkManager, err := network.NewManager(logger, network.DefaultConfig())
		if err != nil {
			logger.Error("failed to create network manager",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}

		// Base directory for VM data
		baseDir := "/opt/metald/vms"

		// Create AssetManager client for asset preparation
		var assetClient assetmanager.Client
		if cfg.AssetManager.Enabled {
			// Use TLS-enabled HTTP client
			httpClient := tlsProvider.HTTPClient()
			assetClient, err = assetmanager.NewClientWithHTTP(&cfg.AssetManager, logger, httpClient)
			if err != nil {
				logger.Error("failed to create assetmanager client",
					slog.String("error", err.Error()),
				)
				os.Exit(1)
			}
			logger.Info("initialized assetmanager client",
				slog.String("endpoint", cfg.AssetManager.Endpoint),
			)
		} else {
			// Use noop client if assetmanager is disabled
			assetClient, _ = assetmanager.NewClient(&cfg.AssetManager, logger)
			logger.Info("assetmanager disabled, using noop client")
		}

		// Use SDK v4 with integrated jailer - the only supported backend
		sdkClient, err := firecracker.NewSDKClientV4(logger, networkManager, assetClient, &cfg.Backend.Jailer, baseDir)
		if err != nil {
			logger.Error("failed to create SDK client v4 with integrated jailer",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}

		logger.Info("initialized firecracker SDK v4 backend with integrated jailer",
			slog.String("firecracker_binary", "/usr/local/bin/firecracker"),
			slog.Uint64("uid", uint64(cfg.Backend.Jailer.UID)),
			slog.Uint64("gid", uint64(cfg.Backend.Jailer.GID)),
			slog.String("chroot_base", cfg.Backend.Jailer.ChrootBaseDir),
		)

		if err := sdkClient.Initialize(); err != nil {
			logger.Error("failed to initialize SDK client v4",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		backend = sdkClient

		// Note: Network manager is initialized and managed by SDK v4
	case types.BackendTypeCloudHypervisor:
		logger.Error("CloudHypervisor backend not implemented",
			slog.String("backend", string(cfg.Backend.Type)),
		)
		os.Exit(1)
	default:
		logger.Error("unsupported backend type",
			slog.String("backend", string(cfg.Backend.Type)),
		)
		os.Exit(1)
	}

	// Create billing client based on configuration
	var billingClient billing.BillingClient
	if cfg.Billing.Enabled {
		if cfg.Billing.MockMode {
			billingClient = billing.NewMockBillingClient(logger)
			logger.Info("initialized mock billing client")
		} else {
			// Use TLS-enabled HTTP client
			httpClient := tlsProvider.HTTPClient()
			// AIDEV-NOTE: Enhanced debug logging for service connection initialization
			logger.Debug("attempting to initialize billing client",
				slog.String("endpoint", cfg.Billing.Endpoint),
				slog.String("tls_mode", cfg.TLS.Mode),
				slog.Bool("mock_mode", cfg.Billing.MockMode),
			)
			billingClient = billing.NewConnectRPCBillingClientWithHTTP(cfg.Billing.Endpoint, logger, httpClient)
			logger.Info("initialized ConnectRPC billing client",
				"endpoint", cfg.Billing.Endpoint,
				"tls_enabled", cfg.TLS.Mode != "disabled",
			)
		}
	} else {
		billingClient = billing.NewMockBillingClient(logger)
		logger.Info("billing disabled, using mock client")
	}

	// Create VM metrics (only if OpenTelemetry is enabled)
	var vmMetrics *observability.VMMetrics
	var billingMetrics *observability.BillingMetrics
	if cfg.OpenTelemetry.Enabled {
		var err error
		vmMetrics, err = observability.NewVMMetrics(logger, cfg.OpenTelemetry.HighCardinalityLabelsEnabled)
		if err != nil {
			logger.Error("failed to initialize VM metrics",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}

		billingMetrics, err = observability.NewBillingMetrics(logger, cfg.OpenTelemetry.HighCardinalityLabelsEnabled)
		if err != nil {
			logger.Error("failed to initialize billing metrics",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
		logger.Info("VM and billing metrics initialized",
			slog.Bool("high_cardinality_enabled", cfg.OpenTelemetry.HighCardinalityLabelsEnabled),
		)
	}

	// Create metrics collector
	instanceID := fmt.Sprintf("metald-%d", time.Now().Unix())
	metricsCollector := billing.NewMetricsCollector(backend, billingClient, logger, instanceID, billingMetrics)

	// Start heartbeat service
	metricsCollector.StartHeartbeat()

	// Create VM service
	vmService := service.NewVMService(backend, logger, metricsCollector, vmMetrics, vmRepo)

	// Create unified health handler
	healthHandler := healthpkg.Handler("metald", getVersion(), startTime)

	// Create ConnectRPC handler with interceptors
	interceptors := []connect.Interceptor{
		service.AuthenticationInterceptor(logger), // Authentication must come first
		loggingInterceptor(logger),
	}

	// Add OTEL interceptor if enabled
	if cfg.OpenTelemetry.Enabled {
		interceptors = append(interceptors, observability.NewOTELInterceptor())
	}

	mux := http.NewServeMux()
	path, handler := vmprovisionerv1connect.NewVmServiceHandler(vmService,
		connect.WithInterceptors(interceptors...),
	)
	mux.Handle(path, handler)

	// Add Prometheus metrics endpoint if enabled
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		mux.Handle("/metrics", otelProviders.PrometheusHTTP)
		logger.Info("Prometheus metrics endpoint enabled",
			slog.String("path", "/metrics"),
		)
	}

	// Create HTTP server with H2C support for gRPC
	addr := fmt.Sprintf("%s:%s", cfg.Server.Address, cfg.Server.Port)

	// AIDEV-NOTE: Removed otelhttp.NewHandler to prevent double-span issues
	// The OTEL interceptor in the ConnectRPC handler already handles tracing
	var httpHandler http.Handler = mux

	// Configure server with optional TLS and security timeouts
	server := &http.Server{
		Addr:    addr,
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}), //nolint:exhaustruct
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

	// Start main server in goroutine
	go func() {
		if serverTLSConfig != nil {
			logger.Info("starting HTTPS server with TLS",
				slog.String("address", addr),
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
				slog.String("address", addr),
			)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("server failed",
					slog.String("error", err.Error()),
				)
				os.Exit(1)
			}
		}
	}()

	// Start Prometheus server on separate port if enabled
	var promServer *http.Server
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		// AIDEV-NOTE: Use configured interface, defaulting to localhost for security
		promAddr := fmt.Sprintf("%s:%s", cfg.OpenTelemetry.PrometheusInterface, cfg.OpenTelemetry.PrometheusPort)
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())
		promMux.HandleFunc("/health", healthHandler)

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

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	<-sigChan

	logger.Info("shutting down server")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Shutdown all servers
	var shutdownErrors []error

	if err := server.Shutdown(ctx); err != nil {
		shutdownErrors = append(shutdownErrors, fmt.Errorf("main server: %w", err))
	}

	if promServer != nil {
		if err := promServer.Shutdown(ctx); err != nil {
			shutdownErrors = append(shutdownErrors, fmt.Errorf("prometheus server: %w", err))
		}
	}

	if len(shutdownErrors) > 0 {
		logger.Error("failed to shutdown servers gracefully",
			slog.String("error", errors.Join(shutdownErrors...).Error()),
		)
		os.Exit(1)
	}

	logger.Info("server shutdown complete")
}

// loggingInterceptor logs all RPC calls
func loggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (resp connect.AnyResponse, err error) {
			// Add panic recovery
			defer func() {
				if r := recover(); r != nil {
					logger.Error("panic in logging interceptor",
						slog.String("procedure", req.Spec().Procedure),
						slog.Any("panic", r),
					)
					// Only override err if it's not already set
					if err == nil {
						err = connect.NewError(connect.CodeInternal, fmt.Errorf("internal server error: %v", r))
					}
				}
			}()

			start := time.Now()

			// Log request
			logger.LogAttrs(ctx, slog.LevelInfo, "rpc request",
				slog.String("procedure", req.Spec().Procedure),
				slog.String("protocol", req.Peer().Protocol),
			)

			// Execute request
			resp, err = next(ctx, req)

			// Log response
			duration := time.Since(start)
			if err != nil {
				logger.LogAttrs(ctx, slog.LevelError, "rpc error",
					slog.String("procedure", req.Spec().Procedure),
					slog.Duration("duration", duration),
					slog.String("error", err.Error()),
				)
			} else {
				logger.LogAttrs(ctx, slog.LevelInfo, "rpc success",
					slog.String("procedure", req.Spec().Procedure),
					slog.Duration("duration", duration),
				)
			}

			return resp, err
		}
	}
}

// printUsage displays help information
func printUsage() {
	fmt.Printf("Metald API Server\n\n")
	fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nEnvironment Variables:\n")
	fmt.Printf("  UNKEY_METALD_PORT                Server port (default: 8080)\n")
	fmt.Printf("  UNKEY_METALD_ADDRESS             Bind address (default: 0.0.0.0)\n")
	fmt.Printf("  UNKEY_METALD_BACKEND             Backend type (default: firecracker)\n")
	fmt.Printf("\nOpenTelemetry Configuration:\n")
	fmt.Printf("  UNKEY_METALD_OTEL_ENABLED              Enable OpenTelemetry (default: false)\n")
	fmt.Printf("  UNKEY_METALD_OTEL_SERVICE_NAME         Service name (default: vmm-controlplane)\n")
	fmt.Printf("  UNKEY_METALD_OTEL_SERVICE_VERSION      Service version (default: 0.1.0)\n")
	fmt.Printf("  UNKEY_METALD_OTEL_SAMPLING_RATE        Trace sampling rate 0.0-1.0 (default: 1.0)\n")
	fmt.Printf("  UNKEY_METALD_OTEL_ENDPOINT             OTLP endpoint (default: localhost:4318)\n")
	fmt.Printf("  UNKEY_METALD_OTEL_PROMETHEUS_ENABLED   Enable Prometheus metrics (default: true)\n")
	fmt.Printf("  UNKEY_METALD_OTEL_PROMETHEUS_PORT      Prometheus metrics port on 0.0.0.0 (default: 9464)\n")
	fmt.Printf("  UNKEY_METALD_OTEL_HIGH_CARDINALITY_ENABLED  Enable high-cardinality labels (default: false)\n")
	fmt.Printf("\nJailer Configuration (Integrated):\n")
	fmt.Printf("  UNKEY_METALD_JAILER_UID                User ID for jailer process (default: 1000)\n")
	fmt.Printf("  UNKEY_METALD_JAILER_GID                Group ID for jailer process (default: 1000)\n")
	fmt.Printf("  UNKEY_METALD_JAILER_CHROOT_DIR         Chroot base directory (default: /srv/jailer)\n")
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  %s                                    # Start metald with default configuration\n", os.Args[0])
	fmt.Printf("  sudo %s                               # Start metald as root (required for networking)\n", os.Args[0])
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("Metald API Server\n")
	fmt.Printf("Version: %s\n", getVersion())
	fmt.Printf("Built with: %s\n", runtime.Version())
}
