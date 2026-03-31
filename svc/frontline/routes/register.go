package routes

import (
	"time"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	acme "github.com/unkeyed/unkey/svc/frontline/routes/acme"
	proxy "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

// Register registers all frontline routes for the HTTPS server
func Register(srv *zen.Server, svc *Services) {
	withLogging := zen.WithLogging(zen.SkipPaths("/_unkey/internal/"))
	withPanicRecovery := zen.WithPanicRecovery()
	withObservability := middleware.WithObservability(svc.Region, svc.ErrorPageRenderer)
	withTimeout := zen.WithTimeout(15 * time.Minute)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withLogging,
		withObservability,
		withTimeout,
	}

	// Catches all requests and routes them to the sentinel or some other region.
	srv.RegisterRoute(
		defaultMiddlewares,
		&proxy.Handler{
			RouterService: svc.RouterService,
			ProxyService:  svc.ProxyService,
			Clock:         svc.Clock,
		},
	)
}

// RegisterChallengeServer registers routes for the HTTP challenge server (Let's Encrypt ACME)
func RegisterChallengeServer(srv *zen.Server, svc *Services) {
	// Catches /.well-known/acme-challenge/{token} so we can forward to ctrl plane.
	srv.RegisterRoute(
		[]zen.Middleware{
			zen.WithPanicRecovery(),
			zen.WithLogging(zen.SkipPaths("/_unkey/internal/")),
			middleware.WithObservability(svc.Region, svc.ErrorPageRenderer),
		},
		&acme.Handler{
			RouterService: svc.RouterService,
			AcmeClient:    svc.AcmeClient,
		},
	)
}
