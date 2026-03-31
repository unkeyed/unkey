package routes

import (
	"net"
	"net/http"
	"time"

	pprofRoute "github.com/unkeyed/unkey/pkg/pprof"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/sentinel/middleware"
	proxy "github.com/unkeyed/unkey/svc/sentinel/routes/proxy"
)

func Register(srv *zen.Server, svc *Services) {
	withPanicRecovery := zen.WithPanicRecovery()
	withObservability := middleware.WithObservability(svc.EnvironmentID, svc.Region)
	withSentinelLogging := middleware.WithSentinelLogging(svc.SentinelRequests, svc.Clock, svc.SentinelID, svc.Region)
	withProxyErrorHandling := middleware.WithProxyErrorHandling()
	withTimeout := zen.WithTimeout(svc.RequestTimeout)
	withLogging := zen.WithLogging(zen.SkipPaths("/_unkey/internal/", "/health/"))
	defaultMiddlewares := []zen.Middleware{
		withPanicRecovery,
		withLogging,
		withObservability,
		withSentinelLogging,
		withTimeout,
		withProxyErrorHandling,
	}

	//nolint:exhaustruct
	transport := &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		MaxIdleConns:        200,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
	}

	if svc.Pprof != nil {
		srv.RegisterRoute(
			[]zen.Middleware{withLogging},
			&pprofRoute.Handler{
				Username: svc.Pprof.Username,
				Password: svc.Pprof.Password,
				Prefix:   "/_unkey/internal",
			},
		)
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
			Engine:             svc.Engine,
		},
	)
}
