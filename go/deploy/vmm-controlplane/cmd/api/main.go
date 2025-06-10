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
	"syscall"
	"time"

	"vmm-controlplane/gen/vmm/v1/vmmv1connect"
	"vmm-controlplane/internal/backend/cloudhypervisor"
	"vmm-controlplane/internal/backend/firecracker"
	"vmm-controlplane/internal/backend/types"
	"vmm-controlplane/internal/config"
	"vmm-controlplane/internal/health"
	"vmm-controlplane/internal/observability"
	"vmm-controlplane/internal/service"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	// Track application start time for uptime calculations
	startTime := time.Now()

	// Parse command-line flags
	var (
		socketPath  = flag.String("socket", "/tmp/ch.sock", "Path to Cloud Hypervisor Unix socket")
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
	logger.Info("starting vmm control plane",
		slog.String("socket_path", *socketPath),
	)

	// Load configuration with socket path override
	cfg, err := config.LoadConfigWithSocketPath(*socketPath)
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
	otelProviders, err := observability.InitProviders(ctx, cfg)
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
		)
	}

	// Initialize backend based on configuration
	var backend types.Backend
	switch cfg.Backend.Type {
	case types.BackendTypeCloudHypervisor:
		backend = cloudhypervisor.NewClient(cfg.Backend.CloudHypervisor.Endpoint, logger)
		logger.Info("initialized cloud hypervisor backend",
			slog.String("endpoint", cfg.Backend.CloudHypervisor.Endpoint),
		)
	case types.BackendTypeFirecracker:
		backend = firecracker.NewClient(cfg.Backend.Firecracker.Endpoint, logger)
		logger.Info("initialized firecracker backend",
			slog.String("endpoint", cfg.Backend.Firecracker.Endpoint),
		)
	default:
		logger.Error("unsupported backend type",
			slog.String("backend", string(cfg.Backend.Type)),
		)
		os.Exit(1)
	}

	// Create VM service
	vmService := service.NewVMService(backend, logger)

	// Create health handler
	healthHandler := health.NewHandler(backend, logger, startTime)

	// Create ConnectRPC handler with interceptors
	interceptors := []connect.Interceptor{
		loggingInterceptor(logger),
	}

	// Add OTEL interceptor if enabled
	if cfg.OpenTelemetry.Enabled {
		interceptors = append(interceptors, observability.NewOTELInterceptor())
	}

	mux := http.NewServeMux()
	path, handler := vmmv1connect.NewVmServiceHandler(vmService,
		connect.WithInterceptors(interceptors...),
	)
	mux.Handle(path, handler)

	// Add comprehensive health check endpoint
	mux.Handle("/_/health", healthHandler)

	// Add simple health check endpoint for backwards compatibility
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	})

	// Add Prometheus metrics endpoint if enabled
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		mux.Handle("/metrics", otelProviders.PrometheusHTTP)
		logger.Info("Prometheus metrics endpoint enabled",
			slog.String("path", "/metrics"),
		)
	}

	// Create HTTP server with H2C support for gRPC
	addr := fmt.Sprintf("%s:%s", cfg.Server.Address, cfg.Server.Port)

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

	// Start main server in goroutine
	go func() {
		logger.Info("starting http server",
			slog.String("address", addr),
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}
	}()

	// Start Prometheus server on separate port if enabled
	var promServer *http.Server
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		promAddr := fmt.Sprintf("0.0.0.0:%s", cfg.OpenTelemetry.PrometheusPort)
		promMux := http.NewServeMux()
		promMux.Handle("/metrics", promhttp.Handler())

		promServer = &http.Server{
			Addr:    promAddr,
			Handler: promMux,
		}

		go func() {
			logger.Info("starting prometheus metrics server",
				slog.String("address", promAddr),
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
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()

			// Log request
			logger.LogAttrs(ctx, slog.LevelInfo, "rpc request",
				slog.String("procedure", req.Spec().Procedure),
				slog.String("protocol", req.Peer().Protocol),
			)

			// Execute request
			resp, err := next(ctx, req)

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
	fmt.Printf("VMM Control Plane (VMCP) API Server\n\n")
	fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nEnvironment Variables:\n")
	fmt.Printf("  UNKEY_VMCP_PORT                Server port (default: 8080)\n")
	fmt.Printf("  UNKEY_VMCP_ADDRESS             Bind address (default: 0.0.0.0)\n")
	fmt.Printf("  UNKEY_VMCP_BACKEND             Backend type (cloudhypervisor, firecracker)\n")
	fmt.Printf("  UNKEY_VMCP_CH_ENDPOINT         Cloud Hypervisor endpoint (overridden by -socket flag)\n")
	fmt.Printf("  UNKEY_VMCP_FC_ENDPOINT         Firecracker endpoint\n")
	fmt.Printf("\nOpenTelemetry Configuration:\n")
	fmt.Printf("  UNKEY_VMCP_OTEL_ENABLED              Enable OpenTelemetry (default: false)\n")
	fmt.Printf("  UNKEY_VMCP_OTEL_SERVICE_NAME         Service name (default: vmm-controlplane)\n")
	fmt.Printf("  UNKEY_VMCP_OTEL_SERVICE_VERSION      Service version (default: 0.0.1)\n")
	fmt.Printf("  UNKEY_VMCP_OTEL_SAMPLING_RATE        Trace sampling rate 0.0-1.0 (default: 1.0)\n")
	fmt.Printf("  UNKEY_VMCP_OTEL_ENDPOINT             OTLP endpoint (default: localhost:4318)\n")
	fmt.Printf("  UNKEY_VMCP_OTEL_PROMETHEUS_ENABLED   Enable Prometheus metrics (default: true)\n")
	fmt.Printf("  UNKEY_VMCP_OTEL_PROMETHEUS_PORT      Prometheus metrics port on 0.0.0.0 (default: 9464)\n")
	fmt.Printf("\nExamples:\n")
	fmt.Printf("  %s                                    # Use default socket /tmp/ch.sock\n", os.Args[0])
	fmt.Printf("  %s -socket /var/run/ch.sock          # Use custom Unix socket\n", os.Args[0])
	fmt.Printf("  %s -socket ./ch.sock                 # Use relative path socket\n", os.Args[0])
}

// printVersion displays version information
func printVersion() {
	fmt.Printf("VMM Control Plane\n")
	fmt.Printf("Version: dev\n")
	fmt.Printf("Built with: %s\n", runtime.Version())
}
