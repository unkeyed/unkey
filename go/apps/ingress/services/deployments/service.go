package deployments

import (
	"context"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Deployment represents where to route a request based on hostname
type Deployment struct {
	// Environment is "preview" or "production"
	Environment string
	// Region where the portal is deployed (e.g., "us-east-1")
	Region string
	// PortalAddress is the Kubernetes service address (e.g., portal-preview.svc.cluster.local:8080)
	PortalAddress string
}

// Service handles deployment lookups
type Service struct {
	logger logging.Logger
}

type Config struct {
	Logger logging.Logger
}

func New(cfg Config) (*Service, error) {
	return &Service{
		logger: cfg.Logger,
	}, nil
}

// LookupByHostname finds where to route a request based on hostname
// Returns:
//   - deployment, true, nil if found
//   - nil, false, nil if not found
//   - nil, false, error if lookup failed
func (s *Service) LookupByHostname(ctx context.Context, hostname string) (*Deployment, bool, error) {
	s.logger.Info("looking up deployment", "hostname", hostname)

	// Mock: Map certain hostnames to deployments
	// In reality this would query the partition database
	mockDeployments := map[string]*Deployment{
		"api.example.com": {
			Environment:   "production",
			Region:        "us-east-1",
			PortalAddress: "portal-production.svc.cluster.local:8080",
		},
		"preview.example.com": {
			Environment:   "preview",
			Region:        "us-east-1",
			PortalAddress: "portal-preview.svc.cluster.local:8080",
		},
		"test.unkey.local": {
			Environment:   "preview",
			Region:        "us-east-1",
			PortalAddress: "portal-preview.svc.cluster.local:8080",
		},
		// Example: deployment in different region
		"eu.example.com": {
			Environment:   "production",
			Region:        "eu-west-1",
			PortalAddress: "portal-production.svc.cluster.local:8080",
		},
	}

	deployment, found := mockDeployments[hostname]
	if !found {
		s.logger.Info("deployment not found", "hostname", hostname)
		return nil, false, nil
	}

	s.logger.Info("deployment found",
		"hostname", hostname,
		"environment", deployment.Environment,
		"region", deployment.Region,
	)

	return deployment, true, nil
}
