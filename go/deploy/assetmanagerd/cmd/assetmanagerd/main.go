package main

import (
	"context"
	"crypto/sha256"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/builderd"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/registry"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/service"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/storage"
	healthpkg "github.com/unkeyed/unkey/go/deploy/pkg/health"
	"github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1"
	"github.com/unkeyed/unkey/go/gen/proto/deploy/assetmanagerd/v1/assetmanagerdv1connect"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

var version = ""

// getVersion returns the version, with fallback logic for development builds
func getVersion() string {
	// AIDEV-NOTE: Unified version handling pattern across all services
	// Priority: ldflags > VCS revision > module version > "dev"
	if version != "" {
		return version
	}

	if info, ok := debug.ReadBuildInfo(); ok {
		// Check for VCS revision (git commit)
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" {
				return setting.Value[:8] // First 8 chars of commit hash
			}
		}
		// Fall back to module version if available
		if info.Main.Version != "" && info.Main.Version != "(devel)" {
			return info.Main.Version
		}
	}

	return "dev"
}

func main() {
	// Track application start time for uptime calculations
	startTime := time.Now()

	var showVersion bool
	flag.BoolVar(&showVersion, "version", false, "Show version information")
	flag.Parse()

	version = getVersion()

	if showVersion {
		fmt.Printf("assetmanagerd version %s\n", version)
		fmt.Printf("Go version: %s\n", runtime.Version())
		os.Exit(0)
	}

	// Create root logger
	//nolint:exhaustruct // Only Level field is needed for handler options
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	logger.Info("starting assetmanagerd",
		slog.String("version", version),
		slog.String("go_version", runtime.Version()),
	)

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load configuration", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Create context that cancels on interrupt
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown gracefully
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Initialize TLS provider (defaults to disabled)
	//nolint:exhaustruct // Only specified TLS fields are needed for this configuration
	tlsConfig := tlspkg.Config{
		Mode:             tlspkg.Mode(cfg.TLSMode),
		CertFile:         cfg.TLSCertFile,
		KeyFile:          cfg.TLSKeyFile,
		CAFile:           cfg.TLSCAFile,
		SPIFFESocketPath: cfg.TLSSPIFFESocketPath,
	}
	tlsProvider, err := tlspkg.NewProvider(ctx, tlsConfig)
	if err != nil {
		// AIDEV-NOTE: TLS/SPIFFE is now required - no fallback to disabled mode
		logger.Error("TLS initialization failed",
			"error", err,
			"mode", cfg.TLSMode)
		os.Exit(1)
	}
	defer tlsProvider.Close()

	logger.Info("TLS provider initialized",
		"mode", cfg.TLSMode,
		"spiffe_enabled", cfg.TLSMode == "spiffe")

	// Initialize OpenTelemetry
	var shutdown func(context.Context) error
	if cfg.OTELEnabled {
		shutdown, err = observability.InitProviders(ctx, cfg, version)
		if err != nil {
			logger.Error("failed to initialize observability", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer func() {
			shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer shutdownCancel()
			if shutdownErr := shutdown(shutdownCtx); shutdownErr != nil {
				logger.Error("failed to shutdown observability", slog.String("error", shutdownErr.Error()))
			}
		}()
	}

	// Initialize storage backend
	storageBackend, err := storage.NewBackend(cfg, logger)
	if err != nil {
		logger.Error("failed to initialize storage backend", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Initialize asset registry (SQLite database)
	assetRegistry, err := registry.New(cfg.DatabasePath, logger)
	if err != nil {
		logger.Error("failed to initialize asset registry", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer assetRegistry.Close()

	// Seed initial kernel assets if they don't exist
	if err := seedKernelAssets(assetRegistry, logger); err != nil {
		logger.Warn("failed to seed kernel assets", slog.String("error", err.Error()))
		// Don't exit - continue without kernel assets, they can be added later
	}

	// Initialize builderd client if enabled
	var builderdClient *builderd.Client
	if cfg.BuilderdEnabled {
		builderdCfg := &builderd.Config{
			Endpoint:    cfg.BuilderdEndpoint,
			Timeout:     cfg.BuilderdTimeout,
			MaxRetries:  cfg.BuilderdMaxRetries,
			RetryDelay:  cfg.BuilderdRetryDelay,
			TLSProvider: tlsProvider,
		}

		var err error
		builderdClient, err = builderd.NewClient(builderdCfg, logger)
		if err != nil {
			logger.Error("failed to create builderd client", slog.String("error", err.Error()))
			os.Exit(1)
		}

		logger.Info("builderd integration enabled",
			slog.String("endpoint", cfg.BuilderdEndpoint),
			slog.Bool("auto_register", cfg.BuilderdAutoRegister),
		)
	} else {
		logger.Info("builderd integration disabled")
	}

	// Create service
	assetService := service.New(cfg, logger, assetRegistry, storageBackend, builderdClient)

	// Start garbage collector if enabled
	if cfg.GCEnabled {
		go assetService.StartGarbageCollector(ctx)
	}

	// Configure shared interceptor options
	interceptorOpts := []interceptors.Option{
		interceptors.WithServiceName("assetmanagerd"),
		interceptors.WithLogger(logger),
		interceptors.WithActiveRequestsMetric(false), // Match existing behavior (no active requests metric)
		interceptors.WithRequestDurationMetric(true), // Match existing behavior
		interceptors.WithErrorResampling(true),
		interceptors.WithPanicStackTrace(true),
	}

	// Add meter if OpenTelemetry is enabled
	if cfg.OTELEnabled {
		interceptorOpts = append(interceptorOpts, interceptors.WithMeter(observability.GetMeter("assetmanagerd")))
	}

	// Get default interceptors (metrics, logging)
	sharedInterceptors := interceptors.NewDefaultInterceptors("assetmanagerd", interceptorOpts...)

	// Convert UnaryInterceptorFunc to Interceptor
	var interceptorList []connect.Interceptor
	for _, interceptor := range sharedInterceptors {
		interceptorList = append(interceptorList, connect.Interceptor(interceptor))
	}

	// Create ConnectRPC handler with shared interceptors
	path, handler := assetmanagerdv1connect.NewAssetManagerServiceHandler(
		assetService,
		connect.WithInterceptors(interceptorList...),
	)

	// Create HTTP server with OTEL instrumentation
	mux := http.NewServeMux()
	mux.Handle(path, handler)

	var httpHandler http.Handler = mux
	if cfg.OTELEnabled {
		httpHandler = otelhttp.NewHandler(mux, "assetmanagerd")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	server := &http.Server{
		Addr: addr,
		//nolint:exhaustruct // Default http2.Server configuration is sufficient
		Handler:      h2c.NewHandler(httpHandler, &http2.Server{}),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Apply TLS configuration if enabled
	serverTLSConfig, _ := tlsProvider.ServerTLSConfig()
	if serverTLSConfig != nil {
		server.TLSConfig = serverTLSConfig
	}

	// Start Prometheus metrics server if enabled
	if cfg.OTELEnabled && cfg.OTELPrometheusEnabled {
		go func() {
			// AIDEV-NOTE: Use configured interface, defaulting to localhost for security
			metricsAddr := fmt.Sprintf("%s:%d", cfg.OTELPrometheusInterface, cfg.OTELPrometheusPort)
			healthHandler := healthpkg.Handler("assetmanagerd", getVersion(), startTime)
			metricsServer := observability.NewMetricsServer(metricsAddr, healthHandler)
			localhostOnly := cfg.OTELPrometheusInterface == "127.0.0.1" || cfg.OTELPrometheusInterface == "localhost"
			logger.Info("starting Prometheus metrics server",
				slog.String("addr", metricsAddr),
				slog.Bool("localhost_only", localhostOnly))
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("metrics server error", slog.String("error", err.Error()))
			}
		}()
	}

	// Start server
	go func() {
		if serverTLSConfig != nil {
			// For TLS, we need to use regular handler, not h2c
			server.Handler = httpHandler
			logger.Info("starting HTTPS server with TLS",
				slog.String("addr", addr),
				slog.String("tls_mode", cfg.TLSMode),
				slog.String("storage_backend", cfg.StorageBackend),
				slog.String("database_path", cfg.DatabasePath),
			)
			// Empty strings for cert/key paths - SPIFFE provides them in memory
			if err := server.ListenAndServeTLS("", ""); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", slog.String("error", err.Error()))
				cancel()
			}
		} else {
			logger.Info("starting HTTP server without TLS",
				slog.String("addr", addr),
				slog.String("storage_backend", cfg.StorageBackend),
				slog.String("database_path", cfg.DatabasePath),
			)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("server error", slog.String("error", err.Error()))
				cancel()
			}
		}
	}()

	// Wait for shutdown signal
	select {
	case <-sigChan:
		logger.Info("received shutdown signal")
	case <-ctx.Done():
		logger.Info("context cancelled")
	}

	// Graceful shutdown
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	logger.Info("shutting down server")
	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("assetmanagerd stopped")
}

// seedKernelAssets automatically registers default kernel assets on startup
func seedKernelAssets(assetRegistry *registry.Registry, logger *slog.Logger) error {
	kernelPath := "/opt/vm-assets/vmlinux"

	// Check if kernel file exists
	if _, err := os.Stat(kernelPath); os.IsNotExist(err) {
		logger.Info("kernel file not found, skipping kernel asset seeding",
			slog.String("path", kernelPath))
		return nil
	}

	// Check if kernel asset already exists
	filters := registry.ListFilters{
		Type: assetv1.AssetType_ASSET_TYPE_KERNEL,
	}
	existingAssets, err := assetRegistry.ListAssets(filters)
	if err != nil {
		return fmt.Errorf("failed to check existing kernel assets: %w", err)
	}

	if len(existingAssets) > 0 {
		logger.Info("kernel assets already exist, skipping seeding",
			slog.Int("existing_count", len(existingAssets)))
		return nil
	}

	// Calculate file info
	fileInfo, err := os.Stat(kernelPath)
	if err != nil {
		return fmt.Errorf("failed to stat kernel file: %w", err)
	}

	checksum, err := calculateChecksum(kernelPath)
	if err != nil {
		return fmt.Errorf("failed to calculate kernel checksum: %w", err)
	}

	// Create kernel asset
	asset := &assetv1.Asset{
		Name:      "vmlinux",
		Type:      assetv1.AssetType_ASSET_TYPE_KERNEL,
		Backend:   assetv1.StorageBackend_STORAGE_BACKEND_LOCAL,
		Location:  kernelPath,
		SizeBytes: fileInfo.Size(),
		Checksum:  checksum,
		Status:    assetv1.AssetStatus_ASSET_STATUS_AVAILABLE,
		Labels: map[string]string{
			"version": "5.10",
			"arch":    "x86_64",
			"default": "true",
		},
		CreatedBy: "assetmanagerd-startup",
	}

	// Register the asset
	err = assetRegistry.CreateAsset(asset)
	if err != nil {
		return fmt.Errorf("failed to register kernel asset: %w", err)
	}

	logger.Info("seeded default kernel asset",
		slog.String("asset_id", asset.Id),
		slog.String("path", kernelPath),
		slog.Int64("size_bytes", fileInfo.Size()),
		slog.String("checksum", checksum))

	return nil
}

// calculateChecksum calculates SHA256 checksum of a file
func calculateChecksum(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
