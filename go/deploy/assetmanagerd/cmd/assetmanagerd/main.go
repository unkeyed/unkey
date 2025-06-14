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
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/gen/proto/asset/v1/assetv1connect"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/config"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/registry"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/service"
	"github.com/unkeyed/unkey/go/deploy/assetmanagerd/internal/storage"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

var version = "dev"

// getVersion returns the version, with fallback logic for development builds
func getVersion() string {
	// AIDEV-NOTE: Unified version handling pattern across all services
	// Priority: ldflags > VCS revision > module version > "dev"
	if version != "dev" {
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
			if err := shutdown(shutdownCtx); err != nil {
				logger.Error("failed to shutdown observability", slog.String("error", err.Error()))
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

	// Create service
	assetService := service.New(cfg, logger, assetRegistry, storageBackend)

	// Start garbage collector if enabled
	if cfg.GCEnabled {
		go assetService.StartGarbageCollector(ctx)
	}

	// Create ConnectRPC handler
	path, handler := assetv1connect.NewAssetManagerServiceHandler(
		assetService,
		connect.WithInterceptors(
			observability.NewLoggingInterceptor(logger),
			observability.NewMetricsInterceptor(),
		),
	)

	// Create HTTP server with OTEL instrumentation
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	
	// Health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	var httpHandler http.Handler = mux
	if cfg.OTELEnabled {
		httpHandler = otelhttp.NewHandler(mux, "assetmanagerd")
	}

	addr := fmt.Sprintf("%s:%d", cfg.Address, cfg.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      httpHandler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Start Prometheus metrics server if enabled
	if cfg.OTELEnabled && cfg.OTELPrometheusEnabled {
		go func() {
			metricsAddr := fmt.Sprintf(":%d", cfg.OTELPrometheusPort)
			metricsServer := observability.NewMetricsServer(metricsAddr)
			logger.Info("starting Prometheus metrics server", slog.String("addr", metricsAddr))
			if err := metricsServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Error("metrics server error", slog.String("error", err.Error()))
			}
		}()
	}

	// Start server
	go func() {
		logger.Info("starting assetmanagerd server",
			slog.String("addr", addr),
			slog.String("storage_backend", cfg.StorageBackend),
			slog.String("database_path", cfg.DatabasePath),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			cancel()
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