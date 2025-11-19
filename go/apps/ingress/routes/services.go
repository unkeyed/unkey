package routes

import (
	"github.com/unkeyed/unkey/go/apps/ingress/services/deployments"
	"github.com/unkeyed/unkey/go/apps/ingress/services/proxy"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/clock"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Services struct {
	Logger            logging.Logger
	Region            string
	DeploymentService deployments.Service
	ProxyService      proxy.Service
	Clock             clock.Clock
	AcmeClient        ctrlv1connect.AcmeServiceClient
}
