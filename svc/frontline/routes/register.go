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
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withTimeout := zen.WithTimeout(5 * time.Minute)

	// Create wide-enabled observability middleware
	withWideObservability := middleware.WithWideObservability(middleware.WideObservabilityConfig{
		Logger:         svc.Logger,
		Region:         svc.Region,
		ServiceVersion: svc.Image,
		Sampler:        zen.NewTailSamplerFromConfig(svc.WideSuccessSampleRate, svc.WideSlowThresholdMs),
	})

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withWideObservability,
		withTimeout,
	}

	// Health check uses simple wide middleware (no error handling complexity)
	srv.RegisterRoute(
		[]zen.Middleware{
			withPanicRecovery,
			zen.WithWide(zen.NewWideConfig(svc.Logger, "frontline", svc.Image, svc.Region)),
		},
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
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)

	// Create wide-enabled observability middleware
	withWideObservability := middleware.WithWideObservability(middleware.WideObservabilityConfig{
		Logger:         svc.Logger,
		Region:         svc.Region,
		ServiceVersion: svc.Image,
		Sampler:        zen.NewTailSamplerFromConfig(svc.WideSuccessSampleRate, svc.WideSlowThresholdMs),
	})

	// Health check endpoint
	srv.RegisterRoute(
		[]zen.Middleware{
			withPanicRecovery,
			zen.WithWide(zen.NewWideConfig(svc.Logger, "frontline", svc.Image, svc.Region)),
		},
		&internalHealth.Handler{
			Logger: svc.Logger,
		},
	)

	// Catches /.well-known/acme-challenge/{token} so we can forward to ctrl plane.
	srv.RegisterRoute(
		[]zen.Middleware{
			withPanicRecovery,
			withWideObservability,
		},
		&acme.Handler{
			Logger:        svc.Logger,
			RouterService: svc.RouterService,
			AcmeClient:    svc.AcmeClient,
		},
	)
}
