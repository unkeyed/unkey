// Package mcp runs mcp.unkey.com, which serves Unkey's public API as MCP
// tools. It holds no database, vault or signing secret: every tool call is
// forwarded to the API with the caller's own credential.
package mcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/buildinfo"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/logger"
	"github.com/unkeyed/unkey/pkg/runner"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"github.com/unkeyed/unkey/svc/mcp/internal/gateway"
	"github.com/unkeyed/unkey/svc/mcp/internal/tools"
)

// Run serves until ctx is cancelled or the process gets SIGINT or SIGTERM.
func Run(ctx context.Context, cfg Config) error {
	registry, err := tools.Load(openapi.Spec)
	if err != nil {
		return fmt.Errorf("load tools from the OpenAPI spec: %w", err)
	}

	handler, err := gateway.New(gateway.Config{
		PublicURL:  cfg.PublicURL,
		APIURL:     cfg.APIURL,
		Issuer:     cfg.OAuthIssuer,
		Version:    buildinfo.Version,
		HTTPClient: &http.Client{Timeout: 2 * time.Minute},
		Clock:      clock.New(),
	}, registry)
	if err != nil {
		return fmt.Errorf("build gateway: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	mux.Handle("/", handler)

	r := runner.New()
	defer r.Recover()

	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.Port))
	if err != nil {
		return fmt.Errorf("listen on port %d: %w", cfg.Port, err)
	}
	server := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	r.DeferCtx(server.Shutdown)
	r.Go(func(context.Context) error {
		logger.Info("mcp server started", "port", cfg.Port, "public_url", cfg.PublicURL, "api_url", cfg.APIURL)
		if err := server.Serve(listener); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("serve: %w", err)
		}
		return nil
	})
	return r.Wait(ctx)
}
