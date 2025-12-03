package webhook

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	"github.com/unkeyed/unkey/go/apps/secrets-webhook/internal/services/mutator"
	"github.com/unkeyed/unkey/go/apps/secrets-webhook/internal/services/registry"
	"github.com/unkeyed/unkey/go/apps/secrets-webhook/routes/healthz"
	"github.com/unkeyed/unkey/go/apps/secrets-webhook/routes/mutate"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/tls"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Run starts the secrets webhook server using zen.
func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New()
	logger = logger.With(slog.String("service", "secrets-webhook"))

	// Create K8s client for registry authentication
	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	// Create registry for fetching image configs
	reg := registry.New(logger, clientset)

	// Load TLS config
	tlsConfig, err := tls.NewFromFiles(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	// Create zen server with TLS
	server, err := zen.New(zen.Config{
		Logger: logger,
		TLS:    tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	// Create mutator service
	m := mutator.New(&mutator.Config{
		UnkeyEnvImage:           cfg.UnkeyEnvImage,
		AnnotationPrefix:        cfg.AnnotationPrefix,
		DefaultProviderEndpoint: cfg.KraneEndpoint,
	}, logger, reg)

	// Common middleware stack
	middlewares := []zen.Middleware{
		zen.WithPanicRecovery(logger),
		zen.WithLogging(logger),
	}

	// Register routes
	server.RegisterRoute(middlewares, &healthz.Handler{})
	server.RegisterRoute(middlewares, &mutate.Handler{
		Logger:  logger,
		Mutator: m,
	})

	// Start server
	addr := fmt.Sprintf(":%d", cfg.HttpPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	logger.Info("Starting secrets webhook server", "addr", addr)
	return server.Serve(ctx, ln)
}
