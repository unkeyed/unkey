package routes

import (
	"net"
	"net/http"
	"time"

	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/middleware"
	internalHealth "github.com/unkeyed/unkey/svc/sentinel/routes/internal_health"
	proxy "github.com/unkeyed/unkey/svc/sentinel/routes/proxy"
)

func Register(srv *zen.Server, svc *Services) {
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withSentinelLogging := middleware.WithSentinelLogging(svc.ClickHouse, svc.Clock, svc.SentinelID, svc.Region)
	withProxyErrorHandling := middleware.WithProxyErrorHandling()
	withTimeout := zen.WithTimeout(5 * time.Minute)

	// Create wide-enabled observability middleware
	withWideObservability := middleware.WithWideObservability(middleware.WideObservabilityConfig{
		Logger:         svc.Logger,
		EnvironmentID:  svc.EnvironmentID,
		Region:         svc.Region,
		ServiceVersion: svc.Image,
		Sampler:        zen.NewTailSamplerFromConfig(svc.WideSuccessSampleRate, svc.WideSlowThresholdMs),
	})

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withWideObservability,
		withSentinelLogging,
		withProxyErrorHandling,
		withTimeout,
	}

	// Health check uses simple wide middleware
	srv.RegisterRoute(
		[]zen.Middleware{
			withPanicRecovery,
			zen.WithWide(zen.NewWideConfig(svc.Logger, "sentinel", svc.Image, svc.Region)),
		},
		&internalHealth.Handler{
			Logger: svc.Logger,
		},
	)

	//nolint:exhaustruct
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:          200,
		MaxIdleConnsPerHost:   50,
		IdleConnTimeout:       90 * time.Second,
		ResponseHeaderTimeout: 30 * time.Second,
	}

	srv.RegisterRoute(
		defaultMiddlewares,
		&proxy.Handler{
			Logger:             svc.Logger,
			RouterService:      svc.RouterService,
			Clock:              svc.Clock,
			Transport:          transport,
			SentinelID:         svc.SentinelID,
			Region:             svc.Region,
			MaxRequestBodySize: svc.MaxRequestBodySize,
		},
	)
}
