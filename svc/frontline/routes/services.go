package routes

import (
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/clock"
	panicmetrics "github.com/unkeyed/unkey/pkg/prometheus/metrics"
	"github.com/unkeyed/unkey/svc/frontline/internal/errorpage"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
)

type Services struct {
	Region            string
	RouterService     router.Service
	ProxyService      proxy.Service
	Clock             clock.Clock
	AcmeClient        ctrl.AcmeServiceClient
	ErrorPageRenderer errorpage.Renderer
	MiddlewareMetrics *middleware.Metrics
	PanicMetrics      *panicmetrics.Metrics
}
