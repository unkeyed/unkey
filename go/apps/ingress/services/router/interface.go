package router

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// RouteDecision contains the routing decision
type RouteDecision struct {
	// LocalGateway is set if there's a healthy gateway in the local region
	LocalGateway *db.Gateway

	// NearestNLBRegion is set if we need to forward to another region's NLB
	NearestNLBRegion string

	// DeploymentID to pass in X-Unkey-Deployment-Id header
	DeploymentID string
}

// Service handles route lookups and gateway selection
type Service interface {
	// LookupByHostname finds routing info for a hostname
	// Returns the ingress route and all gateways for that environment
	LookupByHostname(ctx context.Context, hostname string) (*db.IngressRoute, []db.Gateway, error)

	// SelectGateway picks the best gateway for the request
	// Returns a RouteDecision with either LocalGateway or NearestNLBRegion set
	SelectGateway(route *db.IngressRoute, gateways []db.Gateway) (*RouteDecision, error)
}

// Config holds configuration for the router service
type Config struct {
	Logger                logging.Logger
	Region                string
	DB                    db.Database
	IngressRouteCache     cache.Cache[string, db.IngressRoute]
	GatewaysByEnvironment cache.Cache[string, []db.Gateway]
}
