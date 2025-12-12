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
	withLogging := zen.WithLogging(svc.Logger)
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withObservability := middleware.WithObservability(svc.Logger, svc.Region)
	withTimeout := zen.WithTimeout(5 * time.Minute)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withLogging,
		withObservability,
		withTimeout,
	}

	srv.RegisterRoute(
		[]zen.Middleware{withLogging},
		&internalHealth.Handler{
			Logger: svc.Logger,
		},
	)

	// Catches all requests and routes them to the sentinel or some other region.
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

	// Health check endpoint
	srv.RegisterRoute(
		[]zen.Middleware{withLogging},
		&internalHealth.Handler{
			Logger: svc.Logger,
		},
	)

	// Catches /.well-known/acme-challenge/{token} so we can forward to ctrl plane.
	srv.RegisterRoute(
		[]zen.Middleware{
			zen.WithPanicRecovery(svc.Logger),
			withLogging,
			middleware.WithObservability(svc.Logger, svc.Region),
		},
		&acme.Handler{
			Logger:        svc.Logger,
			RouterService: svc.RouterService,
			AcmeClient:    svc.AcmeClient,
		},
	)
}
