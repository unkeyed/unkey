package routes

import (
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	acme "github.com/unkeyed/unkey/svc/frontline/routes/acme"
	proxy "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
	redirect "github.com/unkeyed/unkey/svc/frontline/routes/redirect"
)

// Register registers all frontline routes for the HTTPS server
func Register(srv *zen.Server, svc *Services) {
	withSanitizeHeaders := middleware.WithReservedHeaderStrip()
	withLogging := zen.WithLogging(zen.SkipPaths("/_unkey/internal/"))
	withPanicRecovery := zen.WithPanicRecovery()
	withObservability := middleware.WithObservability(svc.Region, svc.ErrorPageRenderer)
	withClickHouseLogging := middleware.WithClickHouseLogging(svc.FrontlineRequests, svc.Clock, svc.FrontlineID, svc.Region, svc.Platform)
	withTimeout := zen.WithTimeout(svc.RequestTimeout)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withSanitizeHeaders,
		withLogging,
		withObservability,
		withClickHouseLogging,
		withTimeout,
	}

	// Catches all requests and routes them to the local instance or to a peer frontline.
	srv.RegisterRoute(
		defaultMiddlewares,
		&proxy.Handler{
			RouterService: svc.RouterService,
			ProxyService:  svc.ProxyService,
			Engine:        svc.Engine,
			Clock:         svc.Clock,
		},
	)
}

// RegisterHTTPServer registers routes for the plain-HTTP listener: ACME
// HTTP-01 challenges and a catchall HTTP→HTTPS redirect. ACME's path is more
// specific, so Go's ServeMux dispatches challenges first.
//
// The redirect intentionally runs without logging/observability middleware:
// it is on the hot path for any human-typed http:// URL, must stay cheap,
// and exposes its own Prometheus counter for volume tracking.
func RegisterHTTPServer(srv *zen.Server, svc *Services) {
	srv.RegisterRoute(
		[]zen.Middleware{
			zen.WithPanicRecovery(),
			middleware.WithReservedHeaderStrip(),
			zen.WithLogging(zen.SkipPaths("/_unkey/internal/")),
			middleware.WithObservability(svc.Region, svc.ErrorPageRenderer),
		},
		&acme.Handler{
			RouterService: svc.RouterService,
			AcmeClient:    svc.AcmeClient,
		},
	)

	srv.RegisterRoute(nil, &redirect.Handler{})
}
