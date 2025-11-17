package routes

import (
	"time"

	"github.com/unkeyed/unkey/go/apps/ingress/services/deployments"
	"github.com/unkeyed/unkey/go/apps/ingress/services/proxy"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Services struct {
	Logger            logging.Logger
	Region            string
	DeploymentService deployments.Service
	ProxyService      proxy.Service
	Clock             interface{ Now() time.Time }
}
