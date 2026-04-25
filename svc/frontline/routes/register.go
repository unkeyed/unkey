package routes

import (
	"time"

	edgeredirectv1 "github.com/unkeyed/unkey/gen/proto/frontline/edgeredirect/v1"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	acme "github.com/unkeyed/unkey/svc/frontline/routes/acme"
	proxy "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
	redirect "github.com/unkeyed/unkey/svc/frontline/routes/redirect"
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

	// Catches all requests, evaluates per-route edge-redirect rules, then
	// either short-circuits with a 308 or forwards to the sentinel/region.
	srv.RegisterRoute(
		defaultMiddlewares,
		&proxy.Handler{
			RouterService: svc.RouterService,
			ProxyService:  svc.ProxyService,
			EdgeRedirect:  svc.EdgeRedirect,
			Clock:         svc.Clock,
		},
	)
}

// RegisterHTTPServer registers routes for the plain-HTTP listener: ACME
// HTTP-01 challenges and a catchall HTTP→HTTPS redirect via the
// edge-redirect engine. ACME's path is more specific, so Go's ServeMux
// dispatches challenges first.
//
// The redirect handler intentionally runs without logging/observability
// middleware: it is on the hot path for any human-typed http:// URL, must
// stay cheap, and exposes its own counter for volume tracking.
func RegisterHTTPServer(srv *zen.Server, svc *Services) {
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

	// Always-on HTTP→HTTPS upgrade. Per-customer rules live on the HTTPS
	// listener; this listener has no router lookup wired and no DB hit.
	srv.RegisterRoute(nil, &redirect.Handler{
		Engine: svc.EdgeRedirect,
		Rules: []*edgeredirectv1.Rule{{
			Id:      "default-https",
			Enabled: true,
			Kind:    &edgeredirectv1.Rule_RequireHttps{RequireHttps: &edgeredirectv1.RequireHTTPS{}},
		}},
	})
}
