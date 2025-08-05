package router

import (
	"github.com/unkeyed/unkey/go/apps/gw/router/gateway_proxy"
	"github.com/unkeyed/unkey/go/apps/gw/server"
)

// Register registers all routes with the gateway server.
// This function runs during startup and sets up the middleware chain and routes.
func Register(srv *server.Server, svc *Services) {
	// Define default middleware stack
	defaultMiddlewares := []server.Middleware{
		server.WithPanicRecovery(svc.Logger),
		server.WithErrorHandling(svc.Logger),
		server.WithTracing(),
		server.WithMetrics(svc.ClickHouse), // Pass ClickHouse for event buffering
		server.WithLogging(svc.Logger),
	}

	// Create the main proxy handler that handles all requests
	proxyHandler := &gateway_proxy.Handler{
		Logger:         svc.Logger,
		RoutingService: svc.RoutingService,
	}

	// Wrap the handler with middleware and assign it to the server
	// Since this is a gateway, we handle all routes with a single handler
	srv.SetHandler(srv.WrapHandler(proxyHandler.Handle, defaultMiddlewares))
}
