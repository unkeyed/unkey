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

func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New().With(slog.String("service", "secrets-webhook"))

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes clientset: %w", err)
	}

	reg := registry.New(logger, clientset)

	tlsConfig, err := tls.NewFromFiles(cfg.TLSCertFile, cfg.TLSKeyFile)
	if err != nil {
		return fmt.Errorf("failed to load TLS certificates: %w", err)
	}

	server, err := zen.New(zen.Config{
		Logger: logger,
		TLS:    tlsConfig,
	})
	if err != nil {
		return fmt.Errorf("failed to create server: %w", err)
	}

	m := mutator.New(&mutator.Config{
		UnkeyEnvImage:           cfg.UnkeyEnvImage,
		AnnotationPrefix:        cfg.AnnotationPrefix,
		DefaultProviderEndpoint: cfg.KraneEndpoint,
	}, logger, reg)

	middlewares := []zen.Middleware{
		zen.WithPanicRecovery(logger),
		zen.WithLogging(logger),
	}

	server.RegisterRoute(middlewares, &healthz.Handler{})
	server.RegisterRoute(middlewares, &mutate.Handler{
		Logger:  logger,
		Mutator: m,
	})

	addr := fmt.Sprintf(":%d", cfg.HttpPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	logger.Info("starting secrets webhook server", "addr", addr)
	return server.Serve(ctx, ln)
}
