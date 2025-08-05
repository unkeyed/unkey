package routing

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
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
	return &service{
		db:                 config.DB,
		logger:             config.Logger,
		gatewayConfigCache: config.GatewayConfigCache,
		vmCache:            config.VMCache,
	}, nil
}

// GetTarget retrieves target configuration by ID.
func (s *service) GetConfig(ctx context.Context, host string) (*partitionv1.GatewayConfig, error) {
	// Use SWR (Stale-While-Revalidate) cache pattern
	config, hit, err := s.gatewayConfigCache.SWR(ctx, host, func(ctx context.Context) (*partitionv1.GatewayConfig, error) {
		s.logger.Debug("fetching target from database", "host", host)

		// For now, return a mock gateway since we don't have the actual DB schema
		// In production, this would query the database for gateway configuration
		target := &partitionv1.GatewayConfig{
			// GatewayID:    targetID,
			// WorkspaceID:  "default-workspace",
			// Name:         fmt.Sprintf("Gateway %s", targetID),
			// Enabled:      true,
			// AvailableVMs: []string{"http://localhost:33667"}, // Mock VMs
			// Metadata: map[string]string{
			// 	"version": "1.0",
			// },
		}

		return target, nil
	}, func(err error) cache.Op {
		if err != nil {
			if db.IsNotFound(err) {
				return cache.WriteNull
			}

			return cache.Noop
		}

		return cache.WriteValue
	})

	if hit == cache.Null {
		return nil, fault.New("no target found")
	}

	return config, err
}

// SelectVM picks an available VM from the gateway's VM list using simple round-robin.
func (s *service) SelectVM(ctx context.Context, config *partitionv1.GatewayConfig) (*url.URL, error) {
	s.logger.Debug("selecting VM for gateway") // "gatewayID", targetInfo.GatewayID,
	// "availableVMs", len(targetInfo.AvailableVMs),

	if !config.IsEnabled {
		return nil, fmt.Errorf("gateway %s is disabled", config.DeploymentId)
	}

	if len(config.Vms) == 0 {
		return nil, fmt.Errorf("no VMs available for gateway %s", config.DeploymentId)
	}

	availableVms := make([]db.Vm, 0)
	for _, vm := range config.Vms {
		vm, hit, err := s.vmCache.SWR(ctx, vm.Id, func(ctx context.Context) (db.Vm, error) {
			// todo:
			return db.Vm{}, nil
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

	targetURL, err := url.Parse("http://" + selectedVM.PrivateIp.String)
	if err != nil {
		return nil, fmt.Errorf("invalid VM URL %s: %w", selectedVM.PrivateIp.String, err)
	}

	s.logger.Debug("selected VM",
		"gatewayID", config.DeploymentId,
		"selectedVM", targetURL.String(),
	)

	return targetURL, nil
}
