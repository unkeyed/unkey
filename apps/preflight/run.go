package preflight

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/apps/preflight/internal/services/cleanup"
	"github.com/unkeyed/unkey/apps/preflight/internal/services/mutator"
	"github.com/unkeyed/unkey/apps/preflight/internal/services/registry"
	"github.com/unkeyed/unkey/apps/preflight/internal/services/registry/credentials"
	"github.com/unkeyed/unkey/apps/preflight/routes/healthz"
	"github.com/unkeyed/unkey/apps/preflight/routes/mutate"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/pkg/zen"
)

func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New().With(slog.String("service", "preflight"))

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Set up registry credentials manager
	var registries []credentials.Registry
	if cfg.DepotToken != "" {
		registries = append(registries, credentials.NewDepot(&credentials.DepotConfig{
			Logger: logger,
			Token:  cfg.DepotToken,
		}))
		logger.Info("depot registry configured for on-demand pull tokens")
	}
	credentialsManager := credentials.NewManager(registries...)

	reg := registry.New(registry.Config{
		Logger:      logger,
		Clientset:   clientset,
		Credentials: credentialsManager,
	})

	tlsConfig, err := tls.NewFromFiles(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	//nolint:exhaustruct // zen.Config has many optional fields with sensible defaults
	server, err := zen.New(zen.Config{
		Logger: logger,
		TLS:    tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	m := mutator.New(mutator.Config{
		Logger:                  logger,
		Registry:                reg,
		Clientset:               clientset,
		Credentials:             credentialsManager,
		UnkeyEnvImage:           cfg.UnkeyEnvImage,
		UnkeyEnvImagePullPolicy: cfg.UnkeyEnvImagePullPolicy,
		AnnotationPrefix:        cfg.AnnotationPrefix,
		DefaultProviderEndpoint: cfg.KraneEndpoint,
	})

	middlewares := []zen.Middleware{
		zen.WithPanicRecovery(logger),
		zen.WithLogging(logger),
	}

	server.RegisterRoute(middlewares, &healthz.Handler{})
	server.RegisterRoute(middlewares, &mutate.Handler{
		Logger:  logger,
		Mutator: m,
	})

	// Start background cleanup of expired pull secrets
	cleanupService := cleanup.New(&cleanup.Config{
		Logger:    logger,
		Clientset: clientset,
	})
	go cleanupService.Start(ctx)

	addr := fmt.Sprintf(":%d", cfg.HttpPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	logger.Info("starting preflight server", "addr", addr)
	return server.Serve(ctx, ln)
}
