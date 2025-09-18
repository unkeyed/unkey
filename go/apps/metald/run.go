package metald

import (
	"context"
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/billing"
	"github.com/unkeyed/unkey/go/apps/metald/internal/database"
	"github.com/unkeyed/unkey/go/apps/metald/internal/observability"
	"github.com/unkeyed/unkey/go/apps/metald/internal/service"

	"github.com/pressly/goose/v3"
	_ "github.com/unkeyed/unkey/go/apps/metald/migrations" // Import Go migrations
	healthpkg "github.com/unkeyed/unkey/go/deploy/pkg/health"
	"github.com/unkeyed/unkey/go/deploy/pkg/observability/interceptors"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
	"github.com/unkeyed/unkey/go/gen/proto/metald/v1/metaldv1connect"

	"connectrpc.com/connect"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

// version is set at build time via ldflags
var version = ""

// Config represents the complete configuration for metald
type Config struct {
	Server        ServerConfig
	Backend       BackendConfig
	Database      DatabaseConfig
	AssetManager  AssetManagerConfig
	Billing       BillingConfig
	TLS           TLSConfig
	OpenTelemetry OpenTelemetryConfig
	InstanceID    string
}

// ServerConfig contains server settings
type ServerConfig struct {
	Address string
	Port    string
}

// BackendConfig contains backend settings
type BackendConfig struct {
	Type   string // firecracker, docker, or k8s
	Jailer JailerConfig
}

// JailerConfig contains jailer settings (firecracker only)
type JailerConfig struct {
	UID           uint32
	GID           uint32
	ChrootBaseDir string
}

// DatabaseConfig contains database settings
type DatabaseConfig struct {
	DataDir string
}

// AssetManagerConfig contains asset manager settings
type AssetManagerConfig struct {
	Enabled  bool
	Endpoint string
}

// BillingConfig contains billing settings
type BillingConfig struct {
	Enabled  bool
	Endpoint string
	MockMode bool
}

// TLSConfig contains TLS settings
type TLSConfig struct {
	Mode              string // disabled, file, or spiffe
	CertFile          string
	KeyFile           string
	CAFile            string
	SPIFFESocketPath  string
	EnableCertCaching bool
	CertCacheTTL      string
}

// OpenTelemetryConfig contains OpenTelemetry settings
type OpenTelemetryConfig struct {
	Enabled                      bool
	ServiceName                  string
	ServiceVersion               string
	TracingSamplingRate          float64
	OTLPEndpoint                 string
	PrometheusEnabled            bool
	PrometheusPort               string
	PrometheusInterface          string
	HighCardinalityLabelsEnabled bool
}

// Validate validates the configuration
func (c *Config) Validate() error {
	// Validate backend type
	switch c.Backend.Type {
	case "firecracker", "docker", "k8s":
		// Valid backend types
	default:
		return fmt.Errorf("invalid backend type: %s (must be firecracker, docker, or k8s)", c.Backend.Type)
	}

	// Validate TLS mode
	switch c.TLS.Mode {
	case "disabled", "file", "spiffe":
		// Valid TLS modes
	default:
		return fmt.Errorf("invalid TLS mode: %s (must be disabled, file, or spiffe)", c.TLS.Mode)
	}

	// If TLS mode is file, ensure cert and key files are provided
	if c.TLS.Mode == "file" {
		if c.TLS.CertFile == "" || c.TLS.KeyFile == "" {
			return fmt.Errorf("TLS mode is 'file' but cert or key file not provided")
		}
	}

	// Validate sampling rate
	if c.OpenTelemetry.TracingSamplingRate < 0 || c.OpenTelemetry.TracingSamplingRate > 1 {
		return fmt.Errorf("invalid tracing sampling rate: %f (must be between 0 and 1)", c.OpenTelemetry.TracingSamplingRate)
	}

	return nil
}

// Run starts the metald service with the given configuration
func Run(ctx context.Context, cfg Config) error {
	startTime := time.Now()

	// Initialize structured logger with JSON output
	//exhaustruct:ignore
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))
	slog.SetDefault(logger)

	// Log startup
	logger.Info("starting metald",
		slog.String("version", getVersion()),
		slog.String("go_version", runtime.Version()),
		slog.String("backend", cfg.Backend.Type),
		slog.String("instance_id", cfg.InstanceID),
	)

	// Initialize OpenTelemetry
	otelProviders, err := observability.InitProviders(ctx, convertToInternalConfig(&cfg), getVersion(), logger)
	if err != nil {
		logger.Error("failed to initialize OpenTelemetry",
			slog.String("error", err.Error()),
		)

		return err
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

	// Initialize TLS provider
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
		logger.Error("TLS initialization failed",
			"error", err,
			"mode", cfg.TLS.Mode)
		return err
	}
	defer tlsProvider.Close()

	logger.Info("TLS provider initialized",
		"mode", cfg.TLS.Mode,
		"spiffe_enabled", cfg.TLS.Mode == "spiffe")

	// Initialize database
	db, dbErr := database.NewDatabaseWithLogger(cfg.Database.DataDir, slog.Default())
	if dbErr != nil {
		logger.Error("failed to get DB",
			slog.String("error", dbErr.Error()),
		)
		return dbErr
	}
	defer db.Close()

	// Run database migrations
	if err := runMigrations(cfg.Database.DataDir, logger, db.DB()); err != nil {
		logger.Error("failed to run migrations", slog.String("error", err.Error()))
		return err
	}

	// Initialize backend based on configuration
	backend, err := initializeBackend(ctx, &cfg, logger, tlsProvider)
	if err != nil {
		logger.Error("failed to initialize backend",
			slog.String("error", err.Error()),
			slog.String("backend", cfg.Backend.Type),
		)
		return err
	}

	// Create billing client
	var billingClient billing.BillingClient
	if cfg.Billing.Enabled {
		if cfg.Billing.MockMode {
			billingClient = billing.NewMockBillingClient(logger)
			logger.Info("initialized mock billing client")
		} else {
			httpClient := tlsProvider.HTTPClient()
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
		vmMetrics, err = observability.NewVMMetrics(logger, cfg.OpenTelemetry.HighCardinalityLabelsEnabled)
		if err != nil {
			logger.Error("failed to initialize VM metrics",
				slog.String("error", err.Error()),
			)
			return err
		}

		billingMetrics, err = observability.NewBillingMetrics(logger, cfg.OpenTelemetry.HighCardinalityLabelsEnabled)
		if err != nil {
			logger.Error("failed to initialize billing metrics",
				slog.String("error", err.Error()),
			)
			return err
		}
		logger.Info("VM and billing metrics initialized",
			slog.Bool("high_cardinality_enabled", cfg.OpenTelemetry.HighCardinalityLabelsEnabled),
		)
	}

	// Create metrics collector
	metricsCollector := billing.NewMetricsCollector(backend, billingClient, logger, cfg.InstanceID, billingMetrics)

	// Start heartbeat service
	metricsCollector.StartHeartbeat()

	// Create VM service
	vmService := service.NewVMService(backend, logger, metricsCollector, vmMetrics, db.Queries)

	// Create unified health handler
	healthHandler := healthpkg.Handler("metald", getVersion(), startTime)

	// Create ConnectRPC handler with shared interceptors
	var interceptorList []connect.Interceptor

	// Configure shared interceptor options
	interceptorOpts := []interceptors.Option{
		interceptors.WithServiceName("metald"),
		interceptors.WithLogger(logger),
		interceptors.WithActiveRequestsMetric(true),
		interceptors.WithRequestDurationMetric(true),
		interceptors.WithErrorResampling(true),
		interceptors.WithPanicStackTrace(true),
	}

	// Add meter if OpenTelemetry is enabled
	if cfg.OpenTelemetry.Enabled {
		interceptorOpts = append(interceptorOpts, interceptors.WithMeter(otel.Meter("metald")))
	}

	// Get default interceptors
	sharedInterceptors := interceptors.NewDefaultInterceptors("metald", interceptorOpts...)

	// Add shared interceptors
	for _, interceptor := range sharedInterceptors {
		interceptorList = append(interceptorList, connect.Interceptor(interceptor))
	}

	mux := http.NewServeMux()
	path, handler := metaldv1connect.NewVmServiceHandler(vmService,
		connect.WithInterceptors(interceptorList...),
	)
	mux.Handle(path, handler)

	// Add health endpoint
	mux.HandleFunc("/health", healthHandler)

	// Add Prometheus metrics endpoint if enabled
	if cfg.OpenTelemetry.Enabled && cfg.OpenTelemetry.PrometheusEnabled {
		mux.Handle("/metrics", otelProviders.PrometheusHTTP)
		logger.Info("Prometheus metrics endpoint enabled",
			slog.String("path", "/metrics"),
		)
	}

	// Create HTTP server with H2C support for gRPC
	addr := fmt.Sprintf("%s:%s", cfg.Server.Address, cfg.Server.Port)

	var httpHandler http.Handler = mux

	// Configure server with optional TLS and security timeouts
	server := &http.Server{
		Addr:           addr,
		Handler:        h2c.NewHandler(httpHandler, &http2.Server{}), //nolint:exhaustruct
		ReadTimeout:    30 * time.Second,
		WriteTimeout:   30 * time.Second,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20, // 1MB max header size
	}

	// Apply TLS configuration if enabled
	serverTLSConfig, _ := tlsProvider.ServerTLSConfig()
	if serverTLSConfig != nil {
		server.TLSConfig = serverTLSConfig
		server.Handler = httpHandler // For TLS, use regular handler, not h2c
	}

	// Start main server in goroutine
	go func() {
		if serverTLSConfig != nil {
			logger.Info("starting HTTPS server with TLS",
				slog.String("address", addr),
				slog.String("tls_mode", cfg.TLS.Mode),
			)
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

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
			slog.String("error", errors.Join(shutdownErrors...).Error()),
		)
		return errors.Join(shutdownErrors...)
	}

	logger.Info("server shutdown complete")
	return nil
}

// initializeBackend creates the appropriate backend based on configuration
func initializeBackend(ctx context.Context, cfg *Config, logger *slog.Logger, tlsProvider tlspkg.Provider) (types.Backend, error) {
	switch cfg.Backend.Type {
	case "firecracker":
		return initializeFirecrackerBackend(ctx, cfg, logger, tlsProvider)
	case "docker":
		return initializeDockerBackend(ctx, cfg, logger)
	case "k8s":
		return initializeK8sBackend(ctx, cfg, logger)
	default:
		return nil, fmt.Errorf("unsupported backend type: %s", cfg.Backend.Type)
	}
}

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
				return "dev-" + setting.Value[:7] // First 8 chars of commit hash
			}
		}

		return "dev"
	}

	return version
}

// runMigrations runs database migrations using goose with embedded migrations
func runMigrations(dataDir string, logger *slog.Logger, db *sql.DB) error {
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	// Set embedded filesystem as the migration source
	goose.SetBaseFS(migrationFS)

	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	logger.Info("migrations completed successfully")

	return nil
}
