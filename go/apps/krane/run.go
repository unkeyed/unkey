package krane

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	k8s "k8s.io/client-go/kubernetes"

	"github.com/unkeyed/unkey/go/apps/krane/backend/docker"
	"github.com/unkeyed/unkey/go/apps/krane/backend/kubernetes"
	"github.com/unkeyed/unkey/go/apps/krane/secrets"
	"github.com/unkeyed/unkey/go/apps/krane/secrets/token"
	"github.com/unkeyed/unkey/go/gen/proto/krane/v1/kranev1connect"
	"github.com/unkeyed/unkey/go/pkg/otel"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/shutdown"
	"github.com/unkeyed/unkey/go/pkg/vault"
	"github.com/unkeyed/unkey/go/pkg/vault/storage"
	pkgversion "github.com/unkeyed/unkey/go/pkg/version"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Run starts the krane server with the provided configuration.
//
// Initializes the selected backend (Docker or Kubernetes), sets up HTTP/2
// server with Connect protocol, and handles graceful shutdown on context
// cancellation.
//
// When cfg.OtelEnabled is true, initializes OpenTelemetry tracing, metrics,
// and logging integration.
func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	shutdowns := shutdown.New()

	if cfg.OtelEnabled {
		grafanaErr := otel.InitGrafana(ctx, otel.Config{
			Application:     "krane",
			Version:         pkgversion.Version,
			InstanceID:      cfg.InstanceID,
			CloudRegion:     cfg.Region,
			TraceSampleRate: cfg.OtelTraceSamplingRate,
		},
			shutdowns,
		)
		if grafanaErr != nil {
			return fmt.Errorf("unable to init grafana: %w", grafanaErr)
		}
	}

	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}
	if cfg.Platform != "" {
		logger = logger.With(slog.String("platform", cfg.Platform))
	}
	if cfg.Region != "" {
		logger = logger.With(slog.String("region", cfg.Region))
	}
	if pkgversion.Version != "" {
		logger = logger.With(slog.String("version", pkgversion.Version))
	}

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

	var svc Svc
	var tokenValidator token.Validator
	var k8sClientset k8s.Interface
	switch cfg.Backend {
	case Kubernetes:
		{
			k8sBackend, k8sErr := kubernetes.New(kubernetes.Config{
				Logger:                logger,
				Region:                cfg.Region,
				DeploymentEvictionTTL: cfg.DeploymentEvictionTTL,
			})
			if k8sErr != nil {
				return fmt.Errorf("unable to init kubernetes backend: %w", k8sErr)
			}
			svc = k8sBackend
			k8sClientset = k8sBackend.GetClientset()

			// For K8s backend, use K8s service account token validator
			tokenValidator = token.NewK8sValidator(token.K8sValidatorConfig{
				Clientset: k8sClientset,
			})
			logger.Info("Using K8s service account token validator")
		}
	case Docker:
		{
			dockerBackend, dockerErr := docker.New(logger, docker.Config{
				SocketPath:       cfg.DockerSocketPath,
				RegistryURL:      cfg.RegistryURL,
				RegistryUsername: cfg.RegistryUsername,
				RegistryPassword: cfg.RegistryPassword,
				Region:           cfg.Region,
				Vault:            vaultSvc,
			})
			if dockerErr != nil {
				return fmt.Errorf("unable to init docker backend: %w", dockerErr)
			}
			svc = dockerBackend
		}
	default:
		return fmt.Errorf("unsupported backend: %s", cfg.Backend)
	}

	// Create the service handlers with interceptors
	mux.Handle(kranev1connect.NewDeploymentServiceHandler(svc))
	mux.Handle(kranev1connect.NewGatewayServiceHandler(svc))

	// Register secrets service if vault is configured
	if vaultSvc != nil && tokenValidator != nil {
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

	// Configure server
	addr := fmt.Sprintf(":%d", cfg.HttpPort)

	// Use h2c for HTTP/2 without TLS (for development)
	h2cHandler := h2c.NewHandler(mux, &http2.Server{
		MaxHandlers:                  0,
		MaxConcurrentStreams:         0,
		MaxDecoderHeaderTableSize:    0,
		MaxEncoderHeaderTableSize:    0,
		MaxReadFrameSize:             0,
		PermitProhibitedCipherSuites: false,
		IdleTimeout:                  0,
		ReadIdleTimeout:              0,
		PingTimeout:                  0,
		WriteByteTimeout:             0,
		MaxUploadBufferPerConnection: 0,
		MaxUploadBufferPerStream:     0,
		NewWriteScheduler:            nil,
		CountError:                   nil,
	})

	server := &http.Server{
		Addr:         addr,
		Handler:      h2cHandler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Register server shutdown
	shutdowns.RegisterCtx(server.Shutdown)

	// Start server
	go func() {
		logger.Info("Starting krane server", "addr", addr)

		listenErr := server.ListenAndServe()

		if listenErr != nil && listenErr != http.ErrServerClosed {
			logger.Error("Server failed", "error", listenErr.Error())
		}
	}()

	// Wait for signal and handle shutdown
	logger.Info("Krane server started successfully")
	if err := shutdowns.WaitForSignal(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	logger.Info("krane server shut down successfully")
	return nil
}

type Svc interface {
	kranev1connect.DeploymentServiceHandler
	kranev1connect.GatewayServiceHandler
}
