package routing

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"net/url"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	pdb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"google.golang.org/protobuf/encoding/protojson"
)

// service implements the RoutingService interface with database backend.
type service struct {
	db     db.Database
	logger logging.Logger

	gatewayConfigCache cache.Cache[string, ConfigWithWorkspace]
	instanceCache      cache.Cache[string, pdb.Instance]
}

var _ Service = (*service)(nil)

// New creates a new routing service instance.
func New(config Config) (*service, error) {
	if err := assert.All(
		assert.NotNilAndNotZero(config.Logger, "Logger is required"),
		assert.NotNilAndNotZero(config.DB, "Database is required"),
		assert.NotNilAndNotZero(config.GatewayConfigCache, "Gateway config cache is required"),
		assert.NotNilAndNotZero(&config.InstanceCache, "Instance cache is required"),
	); err != nil {
		return nil, err
	}

	return &service{
		db:                 config.DB,
		logger:             config.Logger,
		gatewayConfigCache: config.GatewayConfigCache,
		instanceCache:      config.InstanceCache,
	}, nil
}

// GetConfig retrieves gateway configuration and workspace ID by hostname.
func (s *service) GetConfig(ctx context.Context, host string) (*ConfigWithWorkspace, error) {
	config, hit, err := s.gatewayConfigCache.SWR(ctx, host, func(ctx context.Context) (ConfigWithWorkspace, error) {
		gatewayRow, err := pdb.Query.FindGatewayByHostname(ctx, s.db.RO(), host)
		if err != nil {
			return ConfigWithWorkspace{}, err
		}

		// Unmarshal the protobuf blob from the database
		var gatewayConfig partitionv1.GatewayConfig
		if err := protojson.Unmarshal(gatewayRow.Config, &gatewayConfig); err != nil {
			return ConfigWithWorkspace{}, fmt.Errorf("failed to unmarshal gateway config: %w", err)
		}

		return ConfigWithWorkspace{
			Config:      &gatewayConfig,
			WorkspaceID: gatewayRow.WorkspaceID,
		}, nil
	}, caches.DefaultFindFirstOp)

	if err != nil && !db.IsNotFound(err) {
		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("error loading gateway configuration"),
			fault.Public("Failed to load gateway configuration"),
		)
	}

	if db.IsNotFound(err) {
		return nil, fault.Wrap(err,
			fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
			fault.Internal("no gateway configuration found for hostname"),
			fault.Public("No configuration found for this domain"),
		)
	}

	if hit == cache.Null {
		return nil, fault.New(
			"gateway not found, null hit",
			fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
			fault.Public("No configuration found for this domain"),
		)
	}

	return &config, nil
}

// SelectVM picks an available VM from the gateway's VM list using random selection.
func (s *service) SelectVM(ctx context.Context, config *partitionv1.GatewayConfig) (*url.URL, error) {
	if len(config.GetDeployments()) == 0 {
		return nil, fmt.Errorf("no deployments configured")
	}

	deployment := config.GetDeployments()[0]
	if !deployment.GetIsEnabled() {
		return nil, fmt.Errorf("gateway %s is disabled", deployment.GetId())
	}

	if len(config.GetInstances()) == 0 {
		return nil, fmt.Errorf("no VMs available for gateway %s", deployment.GetId())
	}

	availableInstances := make([]pdb.Instance, 0)
	for _, instance := range config.GetInstances() {
		dbVm, hit, err := s.instanceCache.SWR(ctx, instance.GetId(), func(ctx context.Context) (pdb.Instance, error) {
			// refactor: this is bad BAD, we should really add a getMany method to the cache
			return pdb.Query.FindInstanceById(ctx, s.db.RO(), instance.GetId())
		}, caches.DefaultFindFirstOp)
		if err != nil {
			if db.IsNotFound(err) {
				s.logger.Warn("failed to load VM from cache", slog.String("vm_id", instance.GetId()), slog.String("error", err.Error()))
				continue
			}

			return nil, err
		}

		if hit == cache.Null {
			continue
		}

		if dbVm.Status != pdb.InstanceStatusRunning {
			continue
		}

		availableInstances = append(availableInstances, dbVm)
	}

	if len(availableInstances) == 0 {
		return nil, fmt.Errorf("no available instances for gateway %s", deployment.GetId())
	}

	// select random instance
	//nolint:gosec // G404: Non-cryptographic random selection for load balancing
	selectedInstance := availableInstances[rand.Intn(len(availableInstances))]

	// Unmarshal the instance config to get the address
	var instanceConfig partitionv1.InstanceConfig
	if err := protojson.Unmarshal(selectedInstance.Config, &instanceConfig); err != nil {
		return nil, fmt.Errorf("failed to unmarshal instance config: %w", err)
	}

	fullUrl := fmt.Sprintf("http://%s", instanceConfig.Address)

	targetURL, err := url.Parse(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid instance URL %s: %w", fullUrl, err)
	}

	return targetURL, nil
}
