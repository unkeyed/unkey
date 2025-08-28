package routing

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
	"google.golang.org/protobuf/proto"
)

// service implements the RoutingService interface with database backend.
type service struct {
	db     db.Database
	logger logging.Logger

	gatewayConfigCache cache.Cache[string, *partitionv1.GatewayConfig]
	vmCache            cache.Cache[string, db.Vm]
}

var _ Service = (*service)(nil)

// New creates a new routing service instance.
func New(config Config) (*service, error) {
	if err := assert.All(
		assert.NotNilAndNotZero(config.Logger, "Logger is required"),
		assert.NotNilAndNotZero(config.DB, "Database is required"),
		assert.NotNilAndNotZero(config.GatewayConfigCache, "Gateway config cache is required"),
		assert.NotNilAndNotZero(config.VMCache, "VM cache is required"),
	); err != nil {
		return nil, err
	}

	return &service{
		db:                 config.DB,
		logger:             config.Logger,
		gatewayConfigCache: config.GatewayConfigCache,
		vmCache:            config.VMCache,
	}, nil
}

// GetTarget retrieves target configuration by ID.
func (s *service) GetConfig(ctx context.Context, host string) (*partitionv1.GatewayConfig, error) {
	config, hit, err := s.gatewayConfigCache.SWR(ctx, host, func(ctx context.Context) (*partitionv1.GatewayConfig, error) {
		gatewayRow, err := db.Query.FindGatewayByHostname(ctx, s.db.RO(), host)
		if err != nil {
			return nil, err
		}

		// Unmarshal the protobuf blob from the database
		var gatewayConfig partitionv1.GatewayConfig
		if err := proto.Unmarshal(gatewayRow.Config, &gatewayConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal gateway config: %w", err)
		}

		return &gatewayConfig, nil
	}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.Wrap(err,
				fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
				fault.Internal("no gateway configuration found for hostname"),
				fault.Public("No configuration found for this domain"),
			)
		}

		return nil, fault.Wrap(err,
			fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
			fault.Internal("error loading gateway configuration"),
			fault.Public("Failed to load gateway configuration"),
		)
	}

	if hit == cache.Null {
		return nil, fault.New("gateway config null",
			fault.Code(codes.Gateway.Routing.ConfigNotFound.URN()),
			fault.Public("No configuration found for this domain"),
		)
	}

	return config, nil
}

// SelectVM picks an available VM from the gateway's VM list using simple round-robin.
func (s *service) SelectVM(ctx context.Context, config *partitionv1.GatewayConfig) (*url.URL, error) {
	if !config.IsEnabled {
		return nil, fmt.Errorf("gateway %s is disabled", config.DeploymentId)
	}

	if len(config.Vms) == 0 {
		return nil, fmt.Errorf("no VMs available for gateway %s", config.DeploymentId)
	}

	availableVms := make([]db.Vm, 0)
	for _, vm := range config.Vms {
		vm, hit, err := s.vmCache.SWR(ctx, vm.Id, func(ctx context.Context) (db.Vm, error) {
			// refactor: this is bad BAD, we should really add a getMany method to the cache
			return db.Query.FindVMById(ctx, s.db.RO(), vm.Id)
		}, caches.DefaultFindFirstOp)

		if err != nil {
			if db.IsNotFound(err) {
				continue
			}

			return nil, err
		}

		if hit == cache.Null {
			continue
		}

		if vm.Status != db.VmsStatusRunning {
			continue
		}

		availableVms = append(availableVms, vm)
	}

	if len(availableVms) == 0 {
		return nil, fmt.Errorf("no available VMs for gateway %s", config.DeploymentId)
	}

	// select random VM
	selectedVM := availableVms[rand.Intn(len(availableVms))]

	fullUrl := fmt.Sprintf("http://%s:%d", selectedVM.PrivateIp.String, selectedVM.Port.Int32)

	targetURL, err := url.Parse(fullUrl)
	if err != nil {
		return nil, fmt.Errorf("invalid VM URL %s: %w", fullUrl, err)
	}

	return targetURL, nil
}
