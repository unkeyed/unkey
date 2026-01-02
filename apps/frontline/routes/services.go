package routes

import (
	"github.com/unkeyed/unkey/apps/frontline/services/proxy"
	"github.com/unkeyed/unkey/apps/frontline/services/router"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

type Services struct {
	Logger        logging.Logger
	Region        string
	RouterService router.Service
	ProxyService  proxy.Service
	Clock         clock.Clock
	AcmeClient    ctrlv1connect.AcmeServiceClient
}
