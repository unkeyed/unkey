package routes

import (
	"time"

	"github.com/unkeyed/unkey/go/apps/ingress/middleware"
	acme "github.com/unkeyed/unkey/go/apps/ingress/routes/acme"
	internalHealth "github.com/unkeyed/unkey/go/apps/ingress/routes/internal_health"
	proxy "github.com/unkeyed/unkey/go/apps/ingress/routes/proxy"
	"github.com/unkeyed/unkey/go/pkg/zen"
)

// Register registers all ingress routes
func Register(srv *zen.Server, svc *Services) {
	// Setup middlewares
	withTracing := middleware.WithTracing(svc.Logger)
	withLogging := zen.WithLogging(svc.Logger)
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withErrorHandling := middleware.WithErrorHandling(svc.Logger)
	withTimeout := zen.WithTimeout(5 * time.Minute)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withTracing,
		withLogging,
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

	// ACME challenge endpoint for Let's Encrypt (/.well-known/acme-challenge/*)
	srv.RegisterCatchAll(
		defaultMiddlewares,
		"/.well-known/acme-challenge/{token}",
		(&acme.Handler{
			Logger: svc.Logger,
		}).Handle,
	)

	// Catch-all proxy route (must be registered last)
	srv.RegisterCatchAll(
		defaultMiddlewares,
		"/{path...}",
		(&proxy.Handler{
			Logger:            svc.Logger,
			DeploymentService: svc.DeploymentService,
			CurrentRegion:     svc.CurrentRegion,
			BaseDomain:        svc.BaseDomain,
		}).Handle,
	)
}
