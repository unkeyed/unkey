package preflight

import (
	"context"
	"fmt"
	"net"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/pkg/logger"
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

	logger.SetSampler(logger.TailSampler{
		SlowThreshold: cfg.Logging.SlowThreshold,
		SampleRate:    cfg.Logging.SampleRate,
	})

	r := runner.New()
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
			Token: cfg.DepotToken,
		}))
		logger.Info("depot registry configured for on-demand pull tokens")
	}
	credentialsManager := credentials.NewManager(registries...)

	reg := registry.New(registry.Config{
		Clientset:          clientset,
		Credentials:        credentialsManager,
		InsecureRegistries: cfg.Registry.InsecureRegistries,
		RegistryAliases:    cfg.Registry.Aliases,
	})

	tlsConfig, err := tls.NewFromFiles(cfg.TLS.CertFile, cfg.TLS.KeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	//nolint:exhaustruct // zen.Config has many optional fields with sensible defaults
	server, err := zen.New(zen.Config{
		TLS: tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	r.RegisterHealth(server.Mux())

	m := mutator.New(mutator.Config{
		Registry:                reg,
		Clientset:               clientset,
		Credentials:             credentialsManager,
		InjectImage:             cfg.Inject.Image,
		InjectImagePullPolicy:   cfg.Inject.ImagePullPolicy,
		DefaultProviderEndpoint: cfg.KraneEndpoint,
	})

	middlewares := []zen.Middleware{
		zen.WithPanicRecovery(),
		zen.WithLogging(),
	}

	server.RegisterRoute(middlewares, &mutate.Handler{
		Mutator: m,
	})

	// Start background cleanup of expired pull secrets
	cleanupService := cleanup.New(&cleanup.Config{
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
