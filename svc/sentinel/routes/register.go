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
	withLogging := zen.WithLogging(svc.Logger)
	withPanicRecovery := zen.WithPanicRecovery(svc.Logger)
	withObservability := middleware.WithObservability(svc.Logger, svc.EnvironmentID, svc.Region)
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
			Logger:        svc.Logger,
			RouterService: svc.RouterService,
			Clock:         svc.Clock,
			Transport:     transport,
		},
	)
}
