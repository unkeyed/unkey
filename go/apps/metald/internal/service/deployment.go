package service

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/apps/metald/internal/backend/types"
	"github.com/unkeyed/unkey/go/apps/metald/internal/database"
	"github.com/unkeyed/unkey/go/apps/metald/internal/network"
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

	br := netConfig.BridgeName
	// Ignore for now
	// br, brErr := network.CreateBridge(logger, netConfig)
	// if brErr != nil {
	// 	logger.Info("failed to create bridge",
	// 		slog.String("error", brErr.Error()),
	// 	)
	// 	if err := s.queries.ReleaseNetwork(ctx, n.ID); err != nil { // Delete entry from DB
	// 		return nil, connect.NewError(connect.CodeInternal, err)
	// 	}

	// 	logger.Debug("cleaned up bridge")

	// 	return nil, connect.NewError(connect.CodeInternal, brErr)
	// }

	// Generate available IPs for this network
	availableIPs, ipErr := network.GenerateAvailableIPs(n.BaseNetwork)
	if ipErr != nil {
		logger.ErrorContext(ctx, "failed to generate available IPs",
			slog.String("error", ipErr.Error()),
		)
		return nil, connect.NewError(connect.CodeInternal, ipErr)
	}

	logger.Info("BridgeName is %s", br)
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

		// This returns vmID and err
		_, err := s.backend.CreateVM(ctx, &metaldv1.VmConfig{
			VcpuCount:     req.Msg.Deployment.GetCpu(),
			MemorySizeMib: req.Msg.Deployment.GetMemorySizeMib(),
			Boot:          req.Msg.Deployment.GetImage(),
			NetworkConfig: "",
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

		_, err = s.queries.CreateVM(ctx, database.CreateVMParams{
			VmID:          vmID,
			DeploymentID:  req.Msg.GetDeployment().DeploymentId,
			VcpuCount:     int64(req.Msg.Deployment.GetCpu()),
			MemorySizeMib: int64(req.Msg.Deployment.GetMemorySizeMib()),
			Boot:          req.Msg.Deployment.GetImage(),
			NetworkConfig: sql.NullString{String: "", Valid: false},
			ConsoleConfig: sql.NullString{String: "", Valid: false},
			StorageConfig: sql.NullString{String: "", Valid: false},
			Metadata:      sql.NullString{String: "", Valid: false},
			IpAddress:     sql.NullString{String: ipRow.Column1, Valid: true},
			BridgeName:    sql.NullString{String: na.BridgeName, Valid: true},
			Status:        int64(types.VMStatusCreated),
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to create VM",
				slog.String("error", err.Error()),
			)
		}

		// Not sure if we should boot it as well??
		err = s.backend.BootVM(ctx, vmID)
		if err != nil {
			logger.ErrorContext(ctx, "failed to create VM",
				slog.String("error", err.Error()),
			)

			return nil, connect.NewError(connect.CodeInternal, err)
		}

		now := time.Now().UnixMilli()
		err = s.queries.UpdateVMStatus(ctx, database.UpdateVMStatusParams{
			VmID:         vmID,
			Status:       int64(types.VMStatusRunning),
			ErrorMessage: sql.NullString{String: "", Valid: false},
			UpdatedAt:    now,
			StartedAt:    sql.NullInt64{Valid: true, Int64: now},
			StoppedAt:    sql.NullInt64{Valid: false, Int64: 0},
		})
		if err != nil {
			logger.ErrorContext(ctx, "failed to update VM status",
				slog.String("error", err.Error()),
			)
		}

		vmIds = append(vmIds, vmID)
	}

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

	vms, err := s.queries.GetVMsByDeployment(ctx, req.Msg.GetDeploymentId())
	if err != nil {
		logger.ErrorContext(ctx, "failed to get VMs",
			slog.String("error", err.Error()),
		)

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	vmResponse := make([]*metaldv1.GetDeploymentResponse_Vm, len(vms))
	for i, vm := range vms {
		vmResponse[i] = &metaldv1.GetDeploymentResponse_Vm{
			Id:    vm.VmID,
			Host:  vm.IpAddress.String,
			Port:  8080,
			State: types.VMStatus(vm.Status).ToProtoVmState(),
		}
	}

	return connect.NewResponse(&metaldv1.GetDeploymentResponse{
		DeploymentId: req.Msg.GetDeploymentId(),
		Vms:          vmResponse,
	}), nil
}
