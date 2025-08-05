package routing

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"time"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/internal/services/caches"
	"github.com/unkeyed/unkey/go/pkg/cache"
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
	return &service{
		db:                 config.DB,
		logger:             config.Logger,
		gatewayConfigCache: config.GatewayConfigCache,
		vmCache:            config.VMCache,
	}, nil
}

// GetTarget retrieves target configuration by ID.
func (s *service) GetConfig(ctx context.Context, host string) (*partitionv1.GatewayConfig, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		s.logger.Debug("gateway config lookup completed",
			"host", host,
			"latency_ms", latency.Milliseconds(),
			"latency_us", latency.Microseconds(),
		)
	}()

	// Use SWR (Stale-While-Revalidate) cache pattern
	config, hit, err := s.gatewayConfigCache.SWR(ctx, host, func(ctx context.Context) (*partitionv1.GatewayConfig, error) {
		dbStart := time.Now()
		s.logger.Debug("fetching target from database", "host", host)

		// Query the database for the gateway config blob
		gatewayRow, err := db.Query.GetGatewayConfig(ctx, s.db.RO(), host)
		if err != nil {
			return nil, err
		}

		// Unmarshal the protobuf blob from the database
		var gatewayConfig partitionv1.GatewayConfig
		if err := proto.Unmarshal(gatewayRow.Config, &gatewayConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal gateway config: %w", err)
		}

		dbLatency := time.Since(dbStart)
		s.logger.Debug("database lookup completed",
			"host", host,
			"db_latency_ms", dbLatency.Milliseconds(),
			"db_latency_us", dbLatency.Microseconds(),
		)

		return &gatewayConfig, nil
	}, caches.DefaultFindFirstOp)

	if err != nil {
		if db.IsNotFound(err) {
			return nil, fault.Wrap(err)
		}

		return nil, fault.Wrap(err)
	}

	if hit == cache.Null {
		return nil, fault.New("no target found")
	}

	return config, err
}

// SelectVM picks an available VM from the gateway's VM list using simple round-robin.
func (s *service) SelectVM(ctx context.Context, config *partitionv1.GatewayConfig) (*url.URL, error) {
	start := time.Now()
	defer func() {
		latency := time.Since(start)
		s.logger.Debug("VM selection completed",
			"deploymentID", config.DeploymentId,
			"latency_ms", latency.Milliseconds(),
			"latency_us", latency.Microseconds(),
		)
	}()

	s.logger.Debug("selecting VM for gateway",
		"deploymentID", config.DeploymentId,
		"total_vms", len(config.Vms),
	)

	if !config.IsEnabled {
		return nil, fmt.Errorf("gateway %s is disabled", config.DeploymentId)
	}

	if len(config.Vms) == 0 {
		return nil, fmt.Errorf("no VMs available for gateway %s", config.DeploymentId)
	}

	availableVms := make([]db.Vm, 0)
	vmLookupStart := time.Now()
	for _, vm := range config.Vms {
		vm, hit, err := s.vmCache.SWR(ctx, vm.Id, func(ctx context.Context) (db.Vm, error) {
			// refactor: this is bad BAD, we should really add a getMany method to the cache
			return db.Query.GetVMByID(ctx, s.db.RO(), vm.Id)
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
	vmLookupLatency := time.Since(vmLookupStart)

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

	s.logger.Debug("selected VM",
		"deploymentID", config.DeploymentId,
		"selectedVM", targetURL.String(),
		"fullUrl", fullUrl,
		"available_vms", len(availableVms),
		"vm_lookup_latency_ms", vmLookupLatency.Milliseconds(),
		"vm_lookup_latency_us", vmLookupLatency.Microseconds(),
	)

	return targetURL, nil
}
