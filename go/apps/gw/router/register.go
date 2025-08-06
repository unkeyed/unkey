package router

import (
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/gw/router/gateway_proxy"
	"github.com/unkeyed/unkey/go/apps/gw/server"
	"github.com/unkeyed/unkey/go/apps/gw/services/proxy"
)

// Register registers all routes with the gateway server.
// This function runs during startup and sets up the middleware chain and routes.
func Register(srv *server.Server, svc *Services, region string) {
	// Define default middleware stack
	defaultMiddlewares := []server.Middleware{
		server.WithTracing(),
		server.WithMetrics(svc.ClickHouse, region), // Pass ClickHouse for event buffering
		server.WithLogging(svc.Logger),
		server.WithPanicRecovery(svc.Logger),
		server.WithErrorHandling(svc.Logger),
	}

	// Create shared transport for connection pooling and HTTP/2 support
	transport := &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
		// Enable HTTP/2
		ForceAttemptHTTP2: true,
	}

	// Create proxy service with shared transport
	proxyService, err := proxy.New(proxy.Config{
		Logger:              svc.Logger,
		MaxIdleConns:        transport.MaxIdleConns,
		IdleConnTimeout:     "90s",
		TLSHandshakeTimeout: "10s",
		// Pass the shared transport
		Transport: transport,
	})
	if err != nil {
		// This shouldn't happen with valid config, but handle it gracefully
		svc.Logger.Error("failed to create proxy service", "error", err.Error())
		return
	}

	// Create the main proxy handler that handles all requests
	proxyHandler := &gateway_proxy.Handler{
		Logger:         svc.Logger,
		RoutingService: svc.RoutingService,
		Proxy:          proxyService,
	}

	// Wrap the handler with middleware and assign it to the server
	// Since this is a gateway, we handle all routes with a single handler
	srv.SetHandler(
		srv.WrapHandler(proxyHandler.Handle, defaultMiddlewares),
	)
}
