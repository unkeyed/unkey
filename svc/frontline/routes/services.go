package routes

import (
	"github.com/unkeyed/unkey/gen/rpc/ctrl"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/svc/frontline/services/proxy"
	"github.com/unkeyed/unkey/svc/frontline/services/router"
)

type Services struct {
	Region        string
	RouterService router.Service
	ProxyService  proxy.Service
	Clock         clock.Clock
	AcmeClient    ctrl.AcmeServiceClient
}
