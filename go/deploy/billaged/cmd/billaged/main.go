package main

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/unkeyed/unkey/go/deploy/billaged/gen/billing/v1/billingv1connect"
	"github.com/unkeyed/unkey/go/deploy/billaged/internal/aggregator"
	"github.com/unkeyed/unkey/go/deploy/billaged/internal/config"
	"github.com/unkeyed/unkey/go/deploy/billaged/internal/observability"
	"github.com/unkeyed/unkey/go/deploy/billaged/internal/service"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func main() {
	// Parse command-line flags with environment variable fallbacks
	var (
		showHelp = flag.Bool("help", false, "Show help information")
	)
	flag.Parse()

	if *showHelp {
		printUsage()
		os.Exit(0)
	}

	// Initialize structured logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
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

	logger.Info("starting billaged service",
		"port", cfg.Server.Port,
		"address", cfg.Server.Address,
		"aggregation_interval", aggregationInterval.String(),
		"otel_enabled", cfg.OpenTelemetry.Enabled,
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

	// Create ConnectRPC handler with interceptors
	interceptors := []connect.Interceptor{
		loggingInterceptor(logger),
	}

	// Add OTEL interceptor if enabled
	if cfg.OpenTelemetry.Enabled {
		interceptors = append(interceptors, observability.NewOTELInterceptor())
	}

	mux := http.NewServeMux()
	path, handler := billingv1connect.NewBillingServiceHandler(billingService,
		connect.WithInterceptors(interceptors...),
	)
	mux.Handle(path, handler)

	// Add health check endpoint
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"billaged"}`))
	})

	// Add stats endpoint
	mux.HandleFunc("/stats", func(w http.ResponseWriter, r *http.Request) {
		activeVMs := agg.GetActiveVMCount()
		
		response := fmt.Sprintf(`{
			"active_vms": %d,
			"aggregation_interval": "%s"
		}`, activeVMs, aggregationInterval.String())
		
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(response))
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

	server := &http.Server{
		Addr:    serverAddr,
		Handler: h2c.NewHandler(httpHandler, &http2.Server{}),
	}

	// Start periodic aggregation
	aggCtx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go agg.StartPeriodicAggregation(aggCtx)

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

	// Start server in goroutine
	go func() {
		logger.Info("billaged service listening",
			"address", serverAddr,
		)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed",
				"error", err.Error(),
			)
			os.Exit(1)
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
				logger.LogAttrs(ctx, slog.LevelDebug, "rpc success",
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
	fmt.Printf("Billaged - VM Usage Billing Aggregation Service\n\n")
	fmt.Printf("Usage: %s [OPTIONS]\n\n", os.Args[0])
	fmt.Printf("Options:\n")
	flag.PrintDefaults()
	fmt.Printf("\nEnvironment Variables:\n")
	fmt.Printf("  BILLAGED_PORT                         Server port (default: 8081)\n")
	fmt.Printf("  BILLAGED_ADDRESS                      Bind address (default: 0.0.0.0)\n")
	fmt.Printf("  BILLAGED_AGGREGATION_INTERVAL         Aggregation interval (default: 60s)\n")
	fmt.Printf("\nOpenTelemetry Configuration:\n")
	fmt.Printf("  BILLAGED_OTEL_ENABLED                 Enable OpenTelemetry (default: false)\n")
	fmt.Printf("  BILLAGED_OTEL_SERVICE_NAME            Service name (default: billaged)\n")
	fmt.Printf("  BILLAGED_OTEL_SERVICE_VERSION         Service version (default: 0.0.1)\n")
	fmt.Printf("  BILLAGED_OTEL_SAMPLING_RATE           Trace sampling rate 0.0-1.0 (default: 1.0)\n")
	fmt.Printf("  BILLAGED_OTEL_ENDPOINT                OTLP endpoint (default: localhost:4318)\n")
	fmt.Printf("  BILLAGED_OTEL_PROMETHEUS_ENABLED      Enable Prometheus metrics (default: true)\n")
	fmt.Printf("  BILLAGED_OTEL_PROMETHEUS_PORT         Prometheus metrics port on 0.0.0.0 (default: 9465)\n")
	fmt.Printf("  BILLAGED_OTEL_HIGH_CARDINALITY_ENABLED  Enable high-cardinality labels (default: false)\n")
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
	fmt.Printf("  BILLAGED_OTEL_ENABLED=true %s        # Enable telemetry\n", os.Args[0])
	fmt.Printf("  BILLAGED_AGGREGATION_INTERVAL=30s %s # 30-second summaries\n", os.Args[0])
}

