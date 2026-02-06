package preflight

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/tls"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/cleanup"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/mutator"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/registry"
	"github.com/unkeyed/unkey/svc/preflight/internal/services/registry/credentials"
	"github.com/unkeyed/unkey/svc/preflight/routes/mutate"
)

func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New().With(slog.String("service", "preflight"))
	r := runner.New(logger)
	defer r.Recover()

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
		Logger:             logger,
		Clientset:          clientset,
		Credentials:        credentialsManager,
		InsecureRegistries: cfg.InsecureRegistries,
		RegistryAliases:    cfg.RegistryAliases,
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

	r.RegisterHealth(server.Mux())

	m := mutator.New(mutator.Config{
		Logger:                  logger,
		Registry:                reg,
		Clientset:               clientset,
		Credentials:             credentialsManager,
		InjectImage:             cfg.InjectImage,
		InjectImagePullPolicy:   cfg.InjectImagePullPolicy,
		DefaultProviderEndpoint: cfg.KraneEndpoint,
	})

	middlewares := []zen.Middleware{
		zen.WithPanicRecovery(logger),
		zen.WithLogging(logger),
	}

	server.RegisterRoute(middlewares, &mutate.Handler{
		Logger:  logger,
		Mutator: m,
	})

	// Start background cleanup of expired pull secrets
	cleanupService := cleanup.New(&cleanup.Config{
		Logger:    logger,
		Clientset: clientset,
	})

	r.Go(func(ctx context.Context) error {
		cleanupService.Start(ctx)
		return nil
	})

	addr := fmt.Sprintf(":%d", cfg.HttpPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	r.DeferCtx(server.Shutdown)
	r.Go(func(ctx context.Context) error {
		logger.Info("starting preflight server", "addr", addr)
		return server.Serve(ctx, ln)
	})

	return r.Wait(ctx)
}
