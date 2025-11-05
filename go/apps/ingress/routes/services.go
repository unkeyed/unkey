package routes

import (
	"github.com/unkeyed/unkey/go/apps/ingress/services/deployments"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

type Services struct {
	Logger            logging.Logger
	DeploymentService *deployments.Service
	CurrentRegion     string
	BaseDomain        string
}
