package deployments

import (
	"context"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// regionProximity maps AWS regions to their closest regions in order of proximity.
// This is used to route traffic to the nearest available region when the deployment
// is not in the current region.
var regionProximity = map[string][]string{
	// US East
	"us-east-1": {"us-east-2", "us-west-2", "us-west-1", "ca-central-1", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2"},
	"us-east-2": {"us-east-1", "us-west-2", "us-west-1", "ca-central-1", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2"},

	// US West
	"us-west-1": {"us-west-2", "us-east-2", "us-east-1", "ca-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2", "eu-west-1", "eu-central-1"},
	"us-west-2": {"us-west-1", "us-east-2", "us-east-1", "ca-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2", "eu-west-1", "eu-central-1"},

	// Canada
	"ca-central-1": {"us-east-2", "us-east-1", "us-west-2", "us-west-1", "eu-west-1", "eu-central-1", "ap-northeast-1", "ap-southeast-1", "ap-southeast-2"},

	// Europe
	"eu-west-1":    {"eu-west-2", "eu-central-1", "eu-north-1", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},
	"eu-west-2":    {"eu-west-1", "eu-central-1", "eu-north-1", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},
	"eu-central-1": {"eu-west-1", "eu-west-2", "eu-north-1", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},
	"eu-north-1":   {"eu-central-1", "eu-west-1", "eu-west-2", "us-east-1", "us-east-2", "us-west-2", "us-west-1", "ap-south-1", "ap-southeast-1"},

	// Asia Pacific
	"ap-south-1":     {"ap-southeast-1", "ap-southeast-2", "ap-northeast-1", "eu-west-1", "eu-central-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2"},
	"ap-northeast-1": {"ap-southeast-1", "ap-southeast-2", "ap-south-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2", "eu-west-1", "eu-central-1"},
	"ap-southeast-1": {"ap-southeast-2", "ap-northeast-1", "ap-south-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2", "eu-west-1", "eu-central-1"},
	"ap-southeast-2": {"ap-southeast-1", "ap-northeast-1", "ap-south-1", "us-west-2", "us-west-1", "us-east-1", "us-east-2", "eu-west-1", "eu-central-1"},
}

// Config holds configuration for the deployment service.
type Config struct {
	Logger logging.Logger
	Region string
}

// service implements the Service interface
type service struct {
	logger logging.Logger
	region string
}

var _ Service = (*service)(nil)

// New creates a new deployment service instance.
func New(cfg Config) (*service, error) {
	return &service{
		logger: cfg.Logger,
		region: cfg.Region,
	}, nil
}

// LookupByHostname finds where to route a request based on hostname
// Returns:
//   - deployment, true, nil if found
//   - nil, false, nil if not found
//   - nil, false, error if lookup failed
func (s *service) LookupByHostname(ctx context.Context, hostname string) (*partitionv1.Deployment, bool, error) {
	s.logger.Info("looking up deployment", "hostname", hostname)

	// Mock: Map certain hostnames to deployments
	// In reality this would query the partition database
	mockDeployments := map[string]*partitionv1.Deployment{
		"api.example.com": {
			Id:             "deployment-prod-api",
			IsEnabled:      true,
			K8SServiceName: "gateway-production.default.svc.cluster.local:8080",
			Region:         "us-east-1",
			Image:          "gateway:latest",
		},
		"preview.example.com": {
			Id:             "deployment-preview-api",
			IsEnabled:      true,
			K8SServiceName: "gateway-preview.default.svc.cluster.local:8080",
			Region:         "us-east-1",
			Image:          "gateway:preview",
		},
		"test.unkey.local": {
			Id:             "deployment-test-local",
			IsEnabled:      true,
			K8SServiceName: "gateway-preview.default.svc.cluster.local:8080",
			Region:         "us-east-1",
			Image:          "gateway:preview",
		},
		// Example: deployment in different region
		"eu.example.com": {
			Id:             "deployment-prod-eu",
			IsEnabled:      true,
			K8SServiceName: "gateway-production.default.svc.cluster.local:8080",
			Region:         "eu-west-1",
			Image:          "gateway:latest",
		},
	}

	deployment, found := mockDeployments[hostname]
	if !found {
		s.logger.Info("deployment not found", "hostname", hostname)
		return nil, false, nil
	}

	s.logger.Info("deployment found",
		"hostname", hostname,
		"deploymentId", deployment.Id,
		"region", deployment.Region,
		"k8sService", deployment.K8SServiceName,
	)

	return deployment, true, nil
}

// IsLocal returns true if the deployment is in the current region
func (s *service) IsLocal(deployment *partitionv1.Deployment) bool {
	return deployment.Region == s.region
}
