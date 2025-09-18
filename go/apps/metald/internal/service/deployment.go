package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/database"
	"github.com/unkeyed/unkey/go/apps/metald/internal/network"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// In-memory tracking of deployments to VM IDs and IPs
var (
	deploymentVMs = make(map[string][]string) // deployment_id -> vm_ids
	vmIPs         = make(map[string]string)   // vm_id -> ip_address
	deploymentMu  sync.RWMutex
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

	nwAlloc, err := s.queries.GetNetworkAllocation(ctx, req.Msg.GetDeployment().DeploymentId)
	if err != nil && !db.IsNotFound(err) {
		logger.Info("failed to get network allocation",
			slog.String("error", err.Error()),
		)

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	if db.IsNotFound(err) {
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

		var br string

		if s.backend.Type() == string(types.BackendTypeFirecracker) {
			fBr, brErr := network.CreateBridge(logger, netConfig)
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

			br = fBr
		} else {
			br = netConfig.BridgeName
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

		nwAlloc = database.GetNetworkAllocationRow{
			ID:           na.ID,
			DeploymentID: na.DeploymentID,
			NetworkID:    na.NetworkID,
			BridgeName:   na.BridgeName,
			AvailableIps: na.AvailableIps,
			AllocatedAt:  na.AllocatedAt,
			BaseNetwork:  n.BaseNetwork,
		}
	}

	logger.DebugContext(ctx, "creating vms", slog.Int64("count", int64(vmCount)))
	vmIds := []string{}
	for vm := range vmCount {
		vmID := fmt.Sprintf("ud-%s", s.generateVmID(ctx))

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
			VmID:                vmID,
			IpAddr:              ipRow.Column1,
			NetworkAllocationID: ipRow.ID,
		})
		if allocErr != nil {
			logger.ErrorContext(ctx, "failed to pop available IP",
				slog.String("error", allocErr.Error()),
			)

			return nil, connect.NewError(connect.CodeInternal, allocErr)
		}

		// Pass network info to backend via NetworkConfig
		networkConfigJSON, _ := json.Marshal(map[string]string{
			"deployment_id": req.Msg.Deployment.GetDeploymentId(),
			"subnet":        nwAlloc.BaseNetwork,
			"allocated_ip":  ipRow.Column1,
			"bridge_name":   nwAlloc.BridgeName,
		})

		// This returns vmID and err
		_, err := s.backend.CreateVM(ctx, &metaldv1.VmConfig{
			VcpuCount:     req.Msg.Deployment.GetCpu(),
			MemorySizeMib: req.Msg.Deployment.GetMemorySizeMib(),
			Boot:          req.Msg.Deployment.GetImage(),
			NetworkConfig: string(networkConfigJSON),
			Console:       nil,
			Storage:       nil,
			Id:            vmID,
			Metadata:      nil,
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to create VM",
				slog.String("error", err.Error()),
			)

			return nil, connect.NewError(connect.CodeInternal, err)
		}

		logger.Debug("created vm",
			slog.String("id", ipAlloc.VmID),
			slog.String("ip", ipAlloc.IpAddr),
			slog.Any("requested", vmCount),
			slog.Any("fulfilled", vm),
		)

		// Store VM IP for tracking
		deploymentMu.Lock()
		vmIPs[vmID] = ipAlloc.IpAddr
		deploymentMu.Unlock()

		// For now boot the VM directly once it's been created
		err = s.backend.BootVM(ctx, vmID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to boot VM",
				slog.String("error", err.Error()),
			)

			return nil, connect.NewError(connect.CodeInternal, err)
		}

		vmIds = append(vmIds, vmID)
	}

	// Track all VM IDs for this deployment
	deploymentMu.Lock()
	deploymentVMs[req.Msg.GetDeployment().DeploymentId] = vmIds
	deploymentMu.Unlock()

	return connect.NewResponse(&metaldv1.CreateDeploymentResponse{
		VmIds: vmIds,
	}), nil
}

// GetDeployment returns all of the VMs and their state for the passed deployment_id
func (s *VMService) GetDeployment(ctx context.Context, req *connect.Request[metaldv1.GetDeploymentRequest]) (*connect.Response[metaldv1.GetDeploymentResponse], error) {
	ctx, span := s.tracer.Start(ctx, "metald.deployment.get",
		trace.WithAttributes(
			attribute.String("service.name", "metald"),
			attribute.String("operation.name", "deployment.get"),
		),
	)
	defer span.End()

	logger := s.logger.With("deployment_id", req.Msg.GetDeploymentId())

	// Get VM IDs from in-memory tracking, this should later be replaced via a DB read.
	deploymentMu.RLock()
	vmIDs := deploymentVMs[req.Msg.GetDeploymentId()]
	deploymentMu.RUnlock()

	if len(vmIDs) == 0 {
		return connect.NewResponse(&metaldv1.GetDeploymentResponse{
			DeploymentId: req.Msg.GetDeploymentId(),
			Vms:          []*metaldv1.GetDeploymentResponse_Vm{},
		}), nil
	}

	// Get VM info from backend for each VM ID
	var vmResponse []*metaldv1.GetDeploymentResponse_Vm
	for _, vmID := range vmIDs {
		vmInfo, err := s.backend.GetVMInfo(ctx, vmID)
		if err != nil {
			logger.Warn("failed to get VM info", slog.String("vm_id", vmID), slog.String("error", err.Error()))
			continue // Skip missing VMs
		}

		// Get VM IP from in-memory tracking
		deploymentMu.RLock()
		vmIP := vmIPs[vmID]
		deploymentMu.RUnlock()

		if vmIP == "" {
			vmIP = "unknown" // Fallback if IP not found
		}

		vmResponse = append(vmResponse, &metaldv1.GetDeploymentResponse_Vm{
			Id:    vmID,
			Host:  vmIP,
			Port:  8080, // For now just force the port to 8080 as this is what k8s/docker uses
			State: vmInfo.State,
		})
	}

	return connect.NewResponse(&metaldv1.GetDeploymentResponse{
		DeploymentId: req.Msg.GetDeploymentId(),
		Vms:          vmResponse,
	}), nil
}
