package routes

import (
	"time"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	acme "github.com/unkeyed/unkey/svc/frontline/routes/acme"
	internalHealth "github.com/unkeyed/unkey/svc/frontline/routes/internal_health"
	proxy "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

// Register registers all frontline routes for the HTTPS server
func Register(srv *zen.Server, svc *Services) {
	withLogging := zen.WithLogging()
	withPanicRecovery := zen.WithPanicRecovery()
	withObservability := middleware.WithObservability(svc.Region)
	withTimeout := zen.WithTimeout(5 * time.Minute)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withLogging,
		withObservability,
		withTimeout,
	}

	srv.RegisterRoute(
		[]zen.Middleware{withLogging},
		&internalHealth.Handler{},
	)

	// Catches all requests and routes them to the sentinel or some other region.
	srv.RegisterRoute(
		defaultMiddlewares,
		&proxy.Handler{
			Region:        svc.Region,
			RouterService: svc.RouterService,
			ProxyService:  svc.ProxyService,
			Clock:         svc.Clock,
		},
	)
}

// RegisterChallengeServer registers routes for the HTTP challenge server (Let's Encrypt ACME)
func RegisterChallengeServer(srv *zen.Server, svc *Services) {
	withLogging := zen.WithLogging()

	// Health check endpoint
	srv.RegisterRoute(
		[]zen.Middleware{withLogging},
		&internalHealth.Handler{},
	)

	// Catches /.well-known/acme-challenge/{token} so we can forward to ctrl plane.
	srv.RegisterRoute(
		[]zen.Middleware{
			zen.WithPanicRecovery(),
			withLogging,
			middleware.WithObservability(svc.Region),
		},
		&acme.Handler{
			RouterService: svc.RouterService,
			AcmeClient:    svc.AcmeClient,
		},
	)
}
