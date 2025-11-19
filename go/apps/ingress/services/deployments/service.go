package deployments

import (
	"context"
	"fmt"

	"github.com/unkeyed/unkey/go/apps/ingress/services/caches"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	internalCaches "github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"google.golang.org/protobuf/encoding/protojson"
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
	Logger                logging.Logger
	Region                string
	DB                    db.Database
	GatewayConfigCache    cache.Cache[string, caches.GatewayConfigData]
	InstancesByDeployment cache.Cache[string, []db.Vm]
}

// service implements the Service interface
type service struct {
	logger                logging.Logger
	region                string
	db                    db.Database
	gatewayConfigCache    cache.Cache[string, caches.GatewayConfigData]
	instancesByDeployment cache.Cache[string, []db.Vm]
}

var _ Service = (*service)(nil)

// New creates a new deployment service instance.
func New(cfg Config) (*service, error) {
	return &service{
		logger:                cfg.Logger,
		region:                cfg.Region,
		db:                    cfg.DB,
		gatewayConfigCache:    cfg.GatewayConfigCache,
		instancesByDeployment: cfg.InstancesByDeployment,
	}, nil
}

// LookupByHostname finds where to route a request based on hostname.
// For deployments in multiple regions, it selects the closest available one.
// Returns:
//   - deployment, true, nil if found
//   - nil, false, nil if not found
//   - nil, false, error if lookup failed
func (s *service) LookupByHostname(ctx context.Context, hostname string) (*partitionv1.Deployment, bool, error) {
	s.logger.Info("looking up deployment", "hostname", hostname)

	// Lookup gateway config from database with SWR cache
	configData, hit, err := s.gatewayConfigCache.SWR(ctx, hostname, func(ctx context.Context) (caches.GatewayConfigData, error) {
		gatewayRow, err := db.Query.FindGatewayByHostname(ctx, s.db.RO(), hostname)
		if err != nil {
			return caches.GatewayConfigData{}, err
		}

		// Unmarshal the protobuf blob from the database
		var gatewayConfig partitionv1.GatewayConfig
		if err := protojson.Unmarshal(gatewayRow.Config, &gatewayConfig); err != nil {
			return caches.GatewayConfigData{}, fmt.Errorf("failed to unmarshal gateway config: %w", err)
		}

		return caches.GatewayConfigData{
			Config:      &gatewayConfig,
			WorkspaceID: gatewayRow.WorkspaceID,
		}, nil
	}, internalCaches.DefaultFindFirstOp)

	if err != nil && !db.IsNotFound(err) {
		return nil, false, fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.ConfigLoadFailed.URN()),
			fault.Internal("error loading gateway configuration"),
			fault.Public("Failed to load gateway configuration"),
		)
	}

	if db.IsNotFound(err) {
		s.logger.Info("deployment not found", "hostname", hostname)
		return nil, false, nil
	}

	if hit == cache.Null {
		s.logger.Info("deployment not found (null cache)", "hostname", hostname)
		return nil, false, nil
	}

	deployments := []*partitionv1.Deployment{configData.Config.Deployment}
	// if len(deployments) == 0 {
	// 	return nil, false, fault.New("gateway config missing deployments",
	// 		fault.Code(codes.Ingress.Internal.InternalServerError.URN()),
	// 		fault.Internal("gateway config has no deployments"),
	// 		fault.Public("Invalid gateway configuration"),
	// 	)
	// }

	// Collect all enabled deployments
	enabledDeployments := make(map[string]*partitionv1.Deployment) // deploymentID -> deployment
	for _, deployment := range deployments {
		if !deployment.GetIsEnabled() {
			continue
		}
		enabledDeployments[deployment.Id] = deployment
	}

	if len(enabledDeployments) == 0 {
		return nil, false, fault.New("no enabled deployments",
			fault.Code(codes.Ingress.Routing.DeploymentDisabled.URN()),
			fault.Internal("all deployments are disabled"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Batch load ALL instances for enabled deployments using SWRMany with single query
	enabledDeploymentIDs := make([]string, 0, len(enabledDeployments))
	for deploymentID := range enabledDeployments {
		enabledDeploymentIDs = append(enabledDeploymentIDs, deploymentID)
	}

	instancesByDeploymentMap, _, err := s.instancesByDeployment.SWRMany(ctx, enabledDeploymentIDs, func(ctx context.Context, deploymentIDs []string) (map[string][]db.Vm, error) {
		// Single query to get ALL instances for ALL deployment IDs
		instances, err := db.Query.FindVMsByIds(ctx, s.db.RO(), deploymentIDs)
		if err != nil {
			return nil, err
		}

		// Group instances by deployment ID
		result := make(map[string][]db.Vm)
		for _, instance := range instances {
			result[instance.DeploymentID] = append(result[instance.DeploymentID], instance)
		}

		return result, nil
	}, internalCaches.DefaultFindFirstOp)

	if err != nil {
		return nil, false, fault.Wrap(err,
			fault.Code(codes.Ingress.Internal.InstanceLoadFailed.URN()),
			fault.Internal("failed to load instances"),
			fault.Public("Unable to process request"),
		)
	}

	// Check which deployments have running instances
	deploymentHasRunningInstances := make(map[string]bool)
	for deploymentID, instances := range instancesByDeploymentMap {
		for _, instance := range instances {
			if instance.Status == db.VmsStatusRunning {
				deploymentHasRunningInstances[deploymentID] = true
				break
			}
		}
	}

	// Filter deployments that are enabled AND have running instances by region
	availableDeploymentsByRegion := make(map[string]*partitionv1.Deployment)
	// for deploymentID, deployment := range enabledDeployments {
	// 	if deploymentHasRunningInstances[deploymentID] {
	// 		availableDeploymentsByRegion[deployment.Region] = deployment
	// 	}
	// }

	if len(availableDeploymentsByRegion) == 0 {
		return nil, false, fault.New("no available deployments",
			fault.Code(codes.Ingress.Routing.NoRunningInstances.URN()),
			fault.Internal("no deployments have running instances"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Select the closest deployment: prefer local region, then closest, then any
	selectedDeployment := s.selectClosestDeployment(availableDeploymentsByRegion)

	s.logger.Info("deployment found",
		"hostname", hostname,
		"deploymentId", selectedDeployment.Id,
		// "region", selectedDeployment.Region,
		"totalAvailableRegions", len(availableDeploymentsByRegion),
	)

	return selectedDeployment, true, nil
}

// selectClosestDeployment selects the closest available deployment based on region proximity.
// It prefers the local region first, then uses the proximity map to find the nearest region,
// and finally falls back to any available deployment.
func (s *service) selectClosestDeployment(availableDeploymentsByRegion map[string]*partitionv1.Deployment) *partitionv1.Deployment {
	// First check if our local region is available
	if availableDeploymentsByRegion[s.region] != nil {
		return availableDeploymentsByRegion[s.region]
	}

	// Find closest region from proximity map
	proximityList, exists := regionProximity[s.region]
	if exists {
		for _, region := range proximityList {
			if availableDeploymentsByRegion[region] != nil {
				return availableDeploymentsByRegion[region]
			}
		}
	}

	// If still not found, use any available deployment
	for _, deployment := range availableDeploymentsByRegion {
		return deployment
	}

	return nil
}
