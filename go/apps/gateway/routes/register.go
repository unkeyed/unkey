package routes

import (
	"net"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/go/apps/gateway/middleware"
	internalHealth "github.com/unkeyed/unkey/go/apps/gateway/routes/internal_health"
	proxy "github.com/unkeyed/unkey/go/apps/gateway/routes/proxy"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Register registers all gateway routes
func Register(srv *zen.Server, svc *Services) {
	// Setup middlewares
	withLogging := zen.WithLogging(svc.Logger)
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withErrorHandling := middleware.WithErrorHandling(svc.Logger)
	withTimeout := zen.WithTimeout(5 * time.Minute)
	withMetrics := middleware.WithMetrics(svc.EnvironmentID, svc.Region)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withLogging,
		withMetrics,       // Record metrics before error handling to capture all requests
		withErrorHandling,
		withTimeout,
	}

	// Health check endpoint (minimal middlewares)
	srv.RegisterRoute(
		[]zen.Middleware{withLogging},
		&internalHealth.Handler{
			Logger: svc.Logger,
		},
	)

	// Create shared transport for connection pooling
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second, // Timeout for instance responses
	}

	// Catch-all proxy route
	srv.RegisterRoute(
		defaultMiddlewares,
		&proxy.Handler{
			Logger:        svc.Logger,
			RouterService: svc.RouterService,
			Clock:         svc.Clock,
			Transport:     transport,
		},
	)
}
