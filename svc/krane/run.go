package krane

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/prometheus"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/vault"
	"github.com/unkeyed/unkey/pkg/vault/storage"
	pkgversion "github.com/unkeyed/unkey/pkg/version"
	"github.com/unkeyed/unkey/svc/krane/internal/deployment"
	"github.com/unkeyed/unkey/svc/krane/internal/sentinel"
	"github.com/unkeyed/unkey/svc/krane/pkg/controlplane"
	"github.com/unkeyed/unkey/svc/krane/secrets"
	"github.com/unkeyed/unkey/svc/krane/secrets/token"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Run starts the krane agent server with the provided configuration.
//
// This function initializes all required services including Kubernetes client,
// vault service for secrets management, gRPC servers for API endpoints, and
// Prometheus metrics server. It blocks until the context is cancelled or a
// fatal error occurs.
//
// The function performs these steps in order:
// 1. Validates the configuration
// 2. Creates structured logger with instance metadata
// 3. Initializes vault service if master keys and S3 config are provided
// 4. Creates Kubernetes client using in-cluster configuration
// 5. Sets up gRPC server with SchedulerService handler
// 6. Registers SecretsService handler if vault is configured
// 7. Starts Prometheus metrics server if port is configured
// 8. Blocks until context cancellation or signal
// 9. Performs graceful shutdown of all services
//
// Returns an error if configuration validation fails, service initialization
// fails, or during shutdown. Context cancellation results in clean shutdown
// with nil error.
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	r := runner.New()

	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}
	if cfg.Region != "" {
		logger = logger.With(slog.String("region", cfg.Region))
	}
	if pkgversion.Version != "" {
		logger = logger.With(slog.String("version", pkgversion.Version))
	}

	cluster := controlplane.NewClient(controlplane.ClientConfig{
		URL:         cfg.ControlPlaneURL,
		BearerToken: cfg.ControlPlaneBearer,
		Region:      cfg.Region,
	})

	inClusterConfig, err := rest.InClusterConfig()
	if err != nil {
		return fmt.Errorf("failed to create in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create k8s clientset: %w", err)
	}

	dynamicClient, err := dynamic.NewForConfig(inClusterConfig)
	if err != nil {
		return fmt.Errorf("failed to create k8s dynamic client: %w", err)
	}

	// Start the deployment controller (independent control loop)
	deploymentCtrl := deployment.New(deployment.Config{
		ClientSet:     clientset,
		DynamicClient: dynamicClient,
		Logger:        logger,
		Cluster:       cluster,
		Region:        cfg.Region,
	})
	if err := deploymentCtrl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start deployment controller: %w", err)
	}
	r.Defer(deploymentCtrl.Stop)

	// Start the sentinel controller (independent control loop)
	sentinelCtrl := sentinel.New(sentinel.Config{
		ClientSet: clientset,
		Logger:    logger,
		Cluster:   cluster,
		Region:    cfg.Region,
	})
	if err := sentinelCtrl.Start(ctx); err != nil {
		return fmt.Errorf("failed to start sentinel controller: %w", err)
	}
	r.Defer(sentinelCtrl.Stop)

	// Create vault service for secrets decryption

	var vaultSvc *vault.Service
	if len(cfg.VaultMasterKeys) > 0 && cfg.VaultS3.URL != "" {
		vaultStorage, vaultStorageErr := storage.NewS3(storage.S3Config{
			Logger:            logger,
			S3URL:             cfg.VaultS3.URL,
			S3Bucket:          cfg.VaultS3.Bucket,
			S3AccessKeyID:     cfg.VaultS3.AccessKeyID,
			S3AccessKeySecret: cfg.VaultS3.AccessKeySecret,
		})
		if vaultStorageErr != nil {
			return fmt.Errorf("unable to create vault storage: %w", vaultStorageErr)
		}

		vaultSvc, err = vault.New(vault.Config{
			Logger:     logger,
			Storage:    vaultStorage,
			MasterKeys: cfg.VaultMasterKeys,
		})
		if err != nil {
			return fmt.Errorf("unable to create vault service: %w", err)
		}
		logger.Info("Vault service initialized", "bucket", cfg.VaultS3.Bucket)
	}

	// Create the connect handler
	mux := http.NewServeMux()

	// Register health endpoints for Kubernetes probes
	r.RegisterHealth(mux)

	tokenValidator := token.NewK8sValidator(token.K8sValidatorConfig{
		Clientset: clientset,
	})

	// Register secrets service if vault is configured
	if vaultSvc != nil {
		secretsSvc := secrets.New(secrets.Config{
			Logger:         logger,
			Vault:          vaultSvc,
			TokenValidator: tokenValidator,
		})
		mux.Handle(kranev1connect.NewSecretsServiceHandler(secretsSvc))
		logger.Info("Secrets service registered")
	} else {
		logger.Info("Secrets service not enabled (missing vault or token validator configuration)")
	}

	addr := fmt.Sprintf(":%d", cfg.RPCPort)
	server := &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: 30 * time.Second,
		// Do not set timeouts here, our streaming rpcs will get canceled too frequently
	}

	// Register server shutdown
	r.DeferCtx(server.Shutdown)

	// Start server
	r.Go(func(ctx context.Context) error {
		logger.Info("Starting ctrl server", "addr", addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	if cfg.PrometheusPort > 0 {
		r.Go(func(ctx context.Context) error {
			logger.Info("prometheus started", "port", cfg.PrometheusPort)
			return prometheus.Serve(fmt.Sprintf(":%d", cfg.PrometheusPort))
		})
	}

	// Wait for signal and handle shutdown
	logger.Info("Krane server started successfully")
	if err := r.Wait(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("krane server shut down successfully")
	return nil
}
