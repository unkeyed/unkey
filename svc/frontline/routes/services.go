package routes

import (
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/zen"
	"github.com/unkeyed/unkey/svc/frontline/middleware"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
)

type Services struct {
	Region        string
	RouterService router.Service
	ProxyService  proxy.Service
	Clock         clock.Clock
	AcmeClient    ctrlv1connect.AcmeServiceClient
	ZenMetrics zen.Metrics
	ObsMetrics middleware.Metrics
}
