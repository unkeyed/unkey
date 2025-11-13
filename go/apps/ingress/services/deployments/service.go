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
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
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
	InstancesByDeployment cache.Cache[string, []pdb.Instance]
}

// service implements the Service interface
type service struct {
	logger                logging.Logger
	region                string
	db                    db.Database
	gatewayConfigCache    cache.Cache[string, caches.GatewayConfigData]
	instancesByDeployment cache.Cache[string, []pdb.Instance]
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
		gatewayRow, err := pdb.Query.FindGatewayByHostname(ctx, s.db.RO(), hostname)
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
			fault.Code(codes.Gateway.Internal.InternalServerError.URN()),
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

	deployments := configData.Config.GetDeployments()
	if len(deployments) == 0 {
		return nil, false, fault.New("gateway config missing deployments",
			fault.Code(codes.Gateway.Internal.InternalServerError.URN()),
			fault.Internal("gateway config has no deployments"),
			fault.Public("Invalid gateway configuration"),
		)
	}

	// Filter enabled deployments by region availability and find the closest one
	var selectedDeployment *partitionv1.Deployment
	availableDeploymentsByRegion := make(map[string]*partitionv1.Deployment)

	for _, deployment := range deployments {
		if !deployment.GetIsEnabled() {
			continue
		}

		// Load instances for this deployment
		instancesList, hit, err := s.instancesByDeployment.SWR(ctx, deployment.Id, func(ctx context.Context) ([]pdb.Instance, error) {
			return pdb.Query.FindInstancesByDeploymentId(ctx, s.db.RO(), deployment.Id)
		}, internalCaches.DefaultFindFirstOp)

		if err != nil && !db.IsNotFound(err) {
			s.logger.Warn("failed to load instances for deployment", "deploymentId", deployment.Id, "error", err)
			continue
		}

		if db.IsNotFound(err) || hit == cache.Null || len(instancesList) == 0 {
			s.logger.Debug("no instances for deployment", "deploymentId", deployment.Id, "region", deployment.Region)
			continue
		}

		// Check if any instances are running
		hasRunningInstances := false
		for _, instance := range instancesList {
			if instance.Status == pdb.InstanceStatusRunning {
				hasRunningInstances = true
				break
			}
		}

		if hasRunningInstances {
			availableDeploymentsByRegion[deployment.Region] = deployment
		}
	}

	if len(availableDeploymentsByRegion) == 0 {
		return nil, false, fault.New("no available deployments",
			fault.Code(codes.Gateway.Routing.VMSelectionFailed.URN()),
			fault.Internal("no deployments have running instances"),
			fault.Public("Service temporarily unavailable"),
		)
	}

	// Select the closest deployment based on region proximity
	if availableDeploymentsByRegion[s.region] != nil {
		// Current region is available - use it
		selectedDeployment = availableDeploymentsByRegion[s.region]
	} else {
		// Find closest region from proximity map
		proximityList, exists := regionProximity[s.region]
		if exists {
			for _, region := range proximityList {
				if availableDeploymentsByRegion[region] != nil {
					selectedDeployment = availableDeploymentsByRegion[region]
					break
				}
			}
		}
		// If still not found, use any available deployment
		if selectedDeployment == nil {
			for _, deployment := range availableDeploymentsByRegion {
				selectedDeployment = deployment
				break
			}
		}
	}

	s.logger.Info("deployment found",
		"hostname", hostname,
		"deploymentId", selectedDeployment.Id,
		"region", selectedDeployment.Region,
		"totalAvailableRegions", len(availableDeploymentsByRegion),
	)

	return selectedDeployment, true, nil
}

// IsLocal returns true if the deployment is in the current region
func (s *service) IsLocal(deployment *partitionv1.Deployment) bool {
	return deployment.Region == s.region
}
