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
	withPanicRecovery := zen.WithPanicRecovery()
	withObservability := middleware.WithObservability(svc.EnvironmentID, svc.Region)
	withSentinelLogging := middleware.WithSentinelLogging(svc.ClickHouse, svc.Clock, svc.SentinelID, svc.Region)
	withProxyErrorHandling := middleware.WithProxyErrorHandling()
	withLogging := zen.WithLogging()
	withTimeout := zen.WithTimeout(5 * time.Minute)

	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withObservability,
		withSentinelLogging,
		withProxyErrorHandling,
		withLogging,
		withTimeout,
	}

	srv.RegisterRoute(
		[]zen.Middleware{withLogging},
		&internalHealth.Handler{},
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
			RouterService:      svc.RouterService,
			Clock:              svc.Clock,
			Transport:          transport,
			SentinelID:         svc.SentinelID,
			Region:             svc.Region,
			MaxRequestBodySize: svc.MaxRequestBodySize,
		},
	)
}
