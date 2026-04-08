package routes

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	panicmetrics "github.com/unkeyed/unkey/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	acme "github.com/unkeyed/unkey/svc/frontline/routes/acme"
	proxy "github.com/unkeyed/unkey/svc/frontline/routes/proxy"
)

// panicMetrics is shared across Register and RegisterChallengeServer to avoid
// double-registration on prometheus.DefaultRegisterer.
var panicMetrics = panicmetrics.NewMetrics(prometheus.DefaultRegisterer)

// Register registers all frontline routes for the HTTPS server
func Register(srv *zen.Server, svc *Services) {
	withLogging := zen.WithLogging(zen.SkipPaths("/_unkey/internal/"))
	withPanicRecovery := zen.WithPanicRecovery(panicMetrics)
	withObservability := middleware.WithObservability(svc.Region, svc.ErrorPageRenderer, svc.MiddlewareMetrics)
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
			zen.WithPanicRecovery(panicMetrics),
			zen.WithLogging(zen.SkipPaths("/_unkey/internal/")),
			middleware.WithObservability(svc.Region, svc.ErrorPageRenderer, svc.MiddlewareMetrics),
		},
		&acme.Handler{
			RouterService: svc.RouterService,
			AcmeClient:    svc.AcmeClient,
		},
	)
}
