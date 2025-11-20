package routes

import (
	"time"

	"github.com/unkeyed/unkey/go/apps/ingress/middleware"
	acme "github.com/unkeyed/unkey/go/apps/ingress/routes/acme"
	internalHealth "github.com/unkeyed/unkey/go/apps/ingress/routes/internal_health"
	proxy "github.com/unkeyed/unkey/go/apps/ingress/routes/proxy"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Register registers all ingress routes for the HTTPS server
func Register(srv *zen.Server, svc *Services) {
	// Setup middlewares
	withObservability := middleware.WithObservability(svc.Logger)
	withLogging := zen.WithLogging(svc.Logger)
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withErrorHandling := middleware.WithErrorHandling(svc.Logger)
	withTimeout := zen.WithTimeout(5 * time.Minute)
	withMetrics := middleware.WithMetrics(svc.Logger, svc.Region)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withObservability,
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

	// Catch-all proxy route
	srv.RegisterRoute(
		defaultMiddlewares,
		&proxy.Handler{
			Logger:        svc.Logger,
			Region:        svc.Region,
			RouterService: svc.RouterService,
			ProxyService:  svc.ProxyService,
			Clock:         svc.Clock,
		},
	)
}

// RegisterChallengeServer registers routes for the HTTP challenge server (Let's Encrypt ACME)
func RegisterChallengeServer(srv *zen.Server, svc *Services) {
	withLogging := zen.WithLogging(svc.Logger)
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withErrorHandling := middleware.WithErrorHandling(svc.Logger)

	challengeMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withLogging,
		withErrorHandling,
	}

	// Health check endpoint
	srv.RegisterRoute(
		[]zen.Middleware{withLogging},
		&internalHealth.Handler{
			Logger: svc.Logger,
		},
	)

	// ACME challenge endpoint for Let's Encrypt (/.well-known/acme-challenge/*)
	srv.RegisterRoute(
		challengeMiddlewares,
		&acme.Handler{
			Logger:        svc.Logger,
			RouterService: svc.RouterService,
			AcmeClient:    svc.AcmeClient,
		},
	)
}
