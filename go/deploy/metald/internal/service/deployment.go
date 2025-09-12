package service

import (
	"context"
	"fmt"
	"log/slog"
	"net"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/database"
	"github.com/unkeyed/unkey/go/deploy/metald/internal/network"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// CreateDeployment allocates a network, generates IDs etc
func (s *VMService) CreateDeployment(ctx context.Context, req *connect.Request[metaldv1.CreateDeploymentRequest]) (*connect.Response[metaldv1.CreateDeploymentResponse], error) {
	ctx, span := s.tracer.Start(ctx, "metald.vm.create",
		trace.WithAttributes(
			attribute.String("service.name", "metald"),
			attribute.String("operation.name", "deployment.create"),
		),
	)
	defer span.End()

	logger := s.logger.With("deployment_id", req.Msg.GetDeployment().GetDeploymentId())

	vmCount := req.Msg.GetDeployment().GetVmCount()

	n, netErr := s.queries.AllocateNetwork(ctx)
	if netErr != nil {
		logger.Info("failed to allocate network",
			slog.String("error", netErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, netErr)
	}

	_, bn, bnErr := net.ParseCIDR(n.BaseNetwork)
	if bnErr != nil {
		logger.Info("failed to parse network",
			slog.String("error", bnErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, netErr)
	}

	netConfig := network.Config{
		BaseNetwork:     bn,
		BridgeName:      network.GenerateID(),
		DNSServers:      []string{"4.2.2.2", "4.2.2.1"},
		EnableIPv6:      false,
		EnableRateLimit: false,
		RateLimitMbps:   0,
	}

	br, brErr := network.CreateBridge(logger, netConfig)
	if brErr != nil {
		logger.Info("failed to create bridge",
			slog.String("error", brErr.Error()),
		)
		if err := s.queries.ReleaseNetwork(ctx, n.ID); err != nil { // Delete entry from DB
			return nil, connect.NewError(connect.CodeInternal, err)
		}

		logger.Debug("cleaned up bridge")

		return nil, connect.NewError(connect.CodeInternal, brErr)
	}

	// Generate available IPs for this network
	availableIPs, ipErr := network.GenerateAvailableIPs(n.BaseNetwork)
	if ipErr != nil {
		logger.ErrorContext(ctx, "failed to generate available IPs",
			slog.String("error", ipErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, ipErr)
	}

	na, naErr := s.queries.CreateNetworkAllocation(ctx, database.CreateNetworkAllocationParams{
		DeploymentID: req.Msg.Deployment.GetDeploymentId(),
		BridgeName:   br,
		NetworkID:    n.ID,
		AvailableIps: availableIPs,
	})
	if naErr != nil {
		logger.ErrorContext(ctx, "failed to save network allocation",
			slog.String("error", naErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, naErr)
	}

	logger.Debug("network allocated",
		slog.Any("network_cidr", netConfig.BaseNetwork.String()),
		slog.Any("network_id", na.ID),
	)

	logger.Debug("bridge allocated",
		slog.Any("network_cidr", netConfig.BaseNetwork.String()),
		slog.String("bridge_id", br),
	)

	logger.DebugContext(ctx, "creating vms", slog.Int64("count", int64(vmCount)))
	for vm := range vmCount {
		id := s.generateVMID(ctx)

		vmid := fmt.Sprintf("ud-%s", id)

		// Pop an available IP from the pool
		ipRow, popErr := s.queries.PopAvailableIPJSON(ctx, req.Msg.Deployment.GetDeploymentId())
		if popErr != nil {
			logger.ErrorContext(ctx, "failed to pop available IP",
				slog.String("error", popErr.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, popErr)
		}

		// Now allocate the IP with all required fields
		ipAlloc, allocErr := s.queries.AllocateIP(ctx, database.AllocateIPParams{
			VmID:                vmid,
			IpAddr:              ipRow.JsonExtract.(string),
			NetworkAllocationID: ipRow.ID,
		})
		if allocErr != nil {
			logger.ErrorContext(ctx, "failed to pop available IP",
				slog.String("error", allocErr.Error()),
			)
			return nil, connect.NewError(connect.CodeInternal, allocErr)
		}

		// CREATION call here

		logger.Debug("created vm",
			slog.String("id", ipAlloc.VmID),
			slog.String("ip", ipAlloc.IpAddr),
			slog.Any("requested", vmCount),
			slog.Any("fulfilled", vm),
		)
	}

	return connect.NewResponse(&metaldv1.CreateDeploymentResponse{
		VmIds: []string{"ud-001", "ud-002", "ud-003"},
	}), nil
}

// GetDeployment returns all of the VMs and their state for the passed deployment_id
func (s *VMService) GetDeployment(ctx context.Context, req *connect.Request[metaldv1.GetDeploymentRequest]) (*connect.Response[metaldv1.GetDeploymentResponse], error) {

	// Sample VMs to "act" against
	vms := []*metaldv1.GetDeploymentResponse_Vm{
		{
			Id:    "ud-001",
			Host:  "host01.asldkfja.unkey.app",
			State: metaldv1.VmState_VM_STATE_RUNNING,
			Port:  8081,
		},
		{
			Id:    "ud-002",
			Host:  "host02.asldkfja.unkey.app",
			State: metaldv1.VmState_VM_STATE_CREATED,
			Port:  8082,
		},
		{
			Id:    "vm-003",
			Host:  "host03.asldkfja.unkey.app",
			State: metaldv1.VmState_VM_STATE_RUNNING,
			Port:  8083,
		},
	}

	return connect.NewResponse(&metaldv1.GetDeploymentResponse{
		DeploymentId: req.Msg.GetDeploymentId(),
		Vms:          vms,
	}), nil
}
