package outpost

import (
	"context"
	"crypto/ecdsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/outpost/proxy"
)

func Run(ctx context.Context, cfg Config) error {
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("bad config: %w", err)
	}

	r := runner.New()
	defer r.Recover()

	outpostRequests := batch.NewNoop[schema.OutpostRequest]()

	if cfg.ClickHouse.URL != "" {
		chClient, err := clickhouse.New(clickhouse.Config{
			URL: cfg.ClickHouse.URL,
		})
		if err != nil {
			return fmt.Errorf("unable to create clickhouse: %w", err)
		}

		outpostRequests = clickhouse.NewBuffer[schema.OutpostRequest](chClient, "default.outpost_requests_raw_v1", clickhouse.BufferConfig{
			Name:          "outpost_requests",
			BatchSize:     cfg.ClickHouse.BatchSize,
			BufferSize:    cfg.ClickHouse.BufferSize,
			FlushInterval: 5 * time.Second,
			Consumers:     cfg.ClickHouse.Consumers,
			Drop:          true,
			OnFlushError:  nil,
		})

		r.Defer(func() error { outpostRequests.Close(); return nil })
		r.Defer(chClient.Close)
	}

	caCert, caKey, err := loadCA(cfg.CA)
	if err != nil {
		return fmt.Errorf("unable to load CA: %w", err)
	}

	certCache := proxy.NewCertCache(caCert, caKey)
	transport := proxy.NewOutboundTransport()
	proxyHandler := proxy.NewHandler(certCache, transport, outpostRequests, cfg.InstanceID, cfg.Region)

	healthSrv, err := zen.New(zen.Config{
		TLS:                nil,
		Flags:              nil,
		EnableH2C:          false,
		MaxRequestBodySize: 0,
		ReadTimeout:        -1,
		WriteTimeout:       -1,
	})
	if err != nil {
		return fmt.Errorf("unable to create health server: %w", err)
	}
	r.RegisterHealth(healthSrv.Mux(), "/_unkey/internal/health")
	r.DeferCtx(healthSrv.Shutdown)

	healthListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HttpPort))
	if err != nil {
		return fmt.Errorf("unable to create health listener: %w", err)
	}
	r.Go(func(ctx context.Context) error {
		logger.Info("outpost health server started", "addr", healthListener.Addr().String())
		serveErr := healthSrv.Serve(ctx, healthListener)
		if serveErr != nil && !errors.Is(serveErr, context.Canceled) {
			return fmt.Errorf("health server error: %w", serveErr)
		}
		return nil
	})

	proxySrv := &http.Server{
		Handler:           proxyHandler,
		ReadHeaderTimeout: 10 * time.Second,
	}
	r.DeferCtx(proxySrv.Shutdown)

	proxyListener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.ProxyPort))
	if err != nil {
		return fmt.Errorf("unable to create proxy listener: %w", err)
	}
	r.Go(func(ctx context.Context) error {
		logger.Info("outpost proxy server started", "addr", proxyListener.Addr().String())
		serveErr := proxySrv.Serve(proxyListener)
		if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			return fmt.Errorf("proxy server error: %w", serveErr)
		}
		return nil
	})

	logger.Info("outpost initialized", "region", cfg.Region, "platform", cfg.Platform)

	if err := r.Wait(ctx); err != nil {
		return fmt.Errorf("shutdown failed: %w", err)
	}

	logger.Info("outpost shut down successfully")
	return nil
}

func loadCA(cfg CAConfig) (*x509.Certificate, *ecdsa.PrivateKey, error) {
	certPEM, err := os.ReadFile(cfg.CertFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read CA cert: %w", err)
	}

	block, _ := pem.Decode(certPEM)
	if block == nil {
		return nil, nil, fmt.Errorf("no PEM block found in CA cert file")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA cert: %w", err)
	}

	keyPEM, err := os.ReadFile(cfg.KeyFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read CA key: %w", err)
	}

	var keyBlock *pem.Block
	rest := keyPEM
	for {
		keyBlock, rest = pem.Decode(rest)
		if keyBlock == nil {
			return nil, nil, fmt.Errorf("no EC PRIVATE KEY block found in CA key file")
		}
		if keyBlock.Type == "EC PRIVATE KEY" {
			break
		}
	}

	key, err := x509.ParseECPrivateKey(keyBlock.Bytes)
	if err != nil {
		return nil, nil, fmt.Errorf("parse CA key: %w", err)
	}

	return cert, key, nil
}
