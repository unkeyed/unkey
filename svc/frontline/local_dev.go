package frontline

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/unkeyed/unkey/internal/services/ratelimit"
	"github.com/unkeyed/unkey/pkg/batch"
	"github.com/unkeyed/unkey/pkg/clickhouse/schema"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/internal/meta"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies"
	"github.com/unkeyed/unkey/svc/frontline/internal/policies/keyauth"
	"github.com/unkeyed/unkey/svc/frontline/internal/proxy"
	"github.com/unkeyed/unkey/svc/frontline/internal/router"
	"github.com/unkeyed/unkey/svc/frontline/routes"
)

// LocalDevConfig connects file-backed routes to the public key verification API.
type LocalDevConfig struct {
	HTTPPort   int                `toml:"http_port" config:"default=8080,min=1,max=65535"`
	APIBaseURL string             `toml:"api_url" config:"default=https://api.unkey.com"`
	RootKey    string             `toml:"root_key" config:"required,nonempty"`
	Routes     []router.FileRoute `toml:"routes" config:"nonempty"`

	ConfigDirectory string `toml:"-"`
}

// RunLocalDev starts the standalone gateway without production infrastructure.
func RunLocalDev(ctx context.Context, cfg LocalDevConfig) error {
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.HTTPPort))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	return ServeLocalDev(ctx, listener, cfg)
}

// ServeLocalDev runs a local-dev HTTP gateway until ctx is canceled.
// It owns listener and closes it on return, including on startup errors.
// Key verification uses the API; standalone rate limits are local to this server.
func ServeLocalDev(ctx context.Context, listener net.Listener, cfg LocalDevConfig) (err error) {
	defer func() {
		if closeErr := listener.Close(); closeErr != nil && !errors.Is(closeErr, net.ErrClosed) {
			err = errors.Join(err, closeErr)
		}
	}()
	routerService, err := router.NewFile(cfg.Routes, cfg.ConfigDirectory)
	if err != nil {
		return fmt.Errorf("load local-dev routes: %w", err)
	}
	clk := clock.New()
	authenticator, err := keyauth.NewAPI(keyauth.APIConfig{
		BaseURL: cfg.APIBaseURL,
		RootKey: cfg.RootKey,
		Clock:   clk,
	})
	if err != nil {
		return fmt.Errorf("configure key verification: %w", err)
	}

	var seed [ed25519.SeedSize]byte
	if _, err := rand.Read(seed[:]); err != nil {
		return fmt.Errorf("generate local-dev metadata key: %w", err)
	}
	metadata, err := meta.New(hex.EncodeToString(seed[:]))
	if err != nil {
		return err
	}
	limiter := ratelimit.NewLocal(clk)
	defer func() { err = errors.Join(err, limiter.Close()) }()
	engine, err := policies.New(policies.Config{
		KeyAuth:     authenticator,
		RateLimiter: limiter,
		Clock:       clk,
	})
	if err != nil {
		return err
	}
	defer engine.Close()
	transports := proxy.NewTransportRegistry()
	defer transports.CloseIdleConnections()
	renderer := errorpage.NewRenderer()
	proxyService, err := proxy.New(proxy.Config{
		InstanceID:          "local-dev",
		Platform:            "local",
		Region:              "local-dev",
		ApexDomain:          "",
		Clock:               clk,
		MaxHops:             0,
		Metadata:            metadata,
		MaxIdleConns:        0,
		IdleConnTimeout:     0,
		TLSHandshakeTimeout: 0,
		Transport:           nil,
		UpstreamTransports:  transports,
		ErrorPageRenderer:   renderer,
	})
	if err != nil {
		return err
	}
	events := batch.NewNoop[schema.FrontlineRequest]()
	defer events.Close()
	server, err := zen.New(zen.Config{
		TLS:                nil,
		Flags:              nil,
		EnableH2C:          true,
		MaxRequestBodySize: 0,
		StreamRequestBody:  true,
		TrustedProxyCIDRs:  nil,
		ReadTimeout:        -1,
		WriteTimeout:       -1,
	})
	if err != nil {
		return err
	}
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		err = errors.Join(err, server.Shutdown(shutdownCtx))
	}()
	routes.Register(server, &routes.Services{
		Region:            "local-dev",
		Platform:          "local",
		FrontlineID:       "local-dev",
		Metadata:          metadata,
		RouterService:     routerService,
		ProxyService:      proxyService,
		Engine:            engine,
		Clock:             clk,
		AcmeClient:        nil,
		DB:                nil,
		ErrorPageRenderer: renderer,
		RequestTimeout:    15 * time.Minute,
		FrontlineRequests: events,
	})
	logger.Info("starting local-dev gateway; do not use in production", "address", listener.Addr().String())
	return server.Serve(ctx, listener)
}
