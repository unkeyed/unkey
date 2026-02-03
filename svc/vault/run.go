package vault

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/gen/proto/vault/v1/vaultv1connect"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/svc/vault/internal/storage"
	storagemiddleware "github.com/unkeyed/unkey/svc/vault/internal/storage/middleware"
	"github.com/unkeyed/unkey/svc/vault/internal/vault"
)

func Run(ctx context.Context, cfg Config) error {
	err := cfg.Validate()
	if err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	logger := logging.New()
	if cfg.InstanceID != "" {
		logger = logger.With(slog.String("instanceID", cfg.InstanceID))
	}

	r := runner.New(logger)
	defer r.Recover()

	// Create the connect handler
	mux := http.NewServeMux()
	r.RegisterHealth(mux)

	s3, err := storage.NewS3(storage.S3Config{
		S3URL:             cfg.S3URL,
		S3Bucket:          cfg.S3Bucket,
		S3AccessKeyID:     cfg.S3AccessKeyID,
		S3AccessKeySecret: cfg.S3AccessKeySecret,
		Logger:            logger,
	})
	if err != nil {
		return fmt.Errorf("failed to create s3 storage: %w", err)
	}

	s3 = storagemiddleware.WithTracing("s3", s3)
	v, err := vault.New(vault.Config{
		Logger:      logger,
		Storage:     s3,
		MasterKeys:  cfg.MasterKeys,
		BearerToken: cfg.BearerToken,
	})
	if err != nil {
		return fmt.Errorf("unable to create vault service: %w", err)
	}

	mux.Handle(vaultv1connect.NewVaultServiceHandler(v))

	addr := fmt.Sprintf(":%d", cfg.HttpPort)
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
		logger.Info("Starting vault server", "addr", addr)
		err := server.ListenAndServe()
		if err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("server failed: %w", err)
		}
		return nil
	})

	if err := r.Wait(ctx); err != nil {
		logger.Error("Shutdown failed", "error", err)
		return err
	}

	return nil
}
