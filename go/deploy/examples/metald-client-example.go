// Package main demonstrates how to create a client for metald with SPIFFE/mTLS support
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

/*
USAGE EXAMPLES:

1. Without TLS (development mode):
   go run metald-client-example.go -endpoint http://localhost:8080 -tls-mode disabled

2. With file-based TLS:
   go run metald-client-example.go -endpoint https://localhost:8080 \
     -tls-mode file \
     -tls-cert /path/to/cert.pem \
     -tls-key /path/to/key.pem \
     -tls-ca /path/to/ca.pem

3. With SPIFFE (production):
   go run metald-client-example.go -endpoint https://localhost:8080 \
     -tls-mode spiffe \
     -spiffe-socket unix:///run/spire/sockets/agent.sock

ENVIRONMENT VARIABLES:
   Instead of flags, you can also use environment variables:
   - METALD_ENDPOINT
   - METALD_TLS_MODE (disabled, file, spiffe)
   - METALD_TLS_CERT_FILE
   - METALD_TLS_KEY_FILE
   - METALD_TLS_CA_FILE
   - METALD_SPIFFE_SOCKET_PATH
   - METALD_CUSTOMER_ID
*/

func main() {
	// Parse command-line flags
	var (
		endpoint         = flag.String("endpoint", getEnvOrDefault("METALD_ENDPOINT", "http://localhost:8080"), "Metald API endpoint")
		tlsMode          = flag.String("tls-mode", getEnvOrDefault("METALD_TLS_MODE", "disabled"), "TLS mode: disabled, file, or spiffe")
		tlsCertFile      = flag.String("tls-cert", getEnvOrDefault("METALD_TLS_CERT_FILE", ""), "TLS certificate file (for file mode)")
		tlsKeyFile       = flag.String("tls-key", getEnvOrDefault("METALD_TLS_KEY_FILE", ""), "TLS key file (for file mode)")
		tlsCAFile        = flag.String("tls-ca", getEnvOrDefault("METALD_TLS_CA_FILE", ""), "TLS CA file (for file mode)")
		spiffeSocketPath = flag.String("spiffe-socket", getEnvOrDefault("METALD_SPIFFE_SOCKET_PATH", "unix:///run/spire/sockets/agent.sock"), "SPIFFE agent socket path")
		customerID       = flag.String("customer-id", getEnvOrDefault("METALD_CUSTOMER_ID", "customer-123"), "Customer ID for VM isolation")
		action           = flag.String("action", "create-and-boot", "Action to perform: create-and-boot, list, info, shutdown, delete")
		vmID             = flag.String("vm-id", "", "VM ID (for info, shutdown, delete actions)")
	)
	flag.Parse()

	// Initialize logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Create TLS configuration
	tlsConfig := tlspkg.Config{
		Mode:             tlspkg.Mode(*tlsMode),
		CertFile:         *tlsCertFile,
		KeyFile:          *tlsKeyFile,
		CAFile:           *tlsCAFile,
		SPIFFESocketPath: *spiffeSocketPath,
		// Enable certificate caching for better performance
		EnableCertCaching: true,
		CertCacheTTL:      5 * time.Second,
	}

	// Create TLS provider
	ctx := context.Background()
	tlsProvider, err := tlspkg.NewProvider(ctx, tlsConfig)
	if err != nil {
		logger.Error("failed to create TLS provider", "error", err)
		os.Exit(1)
	}
	defer tlsProvider.Close()

	// Create HTTP client with TLS configuration
	httpClient := tlsProvider.HTTPClient()

	// Create metald client
	client := vmprovisionerv1connect.NewVmServiceClient(
		httpClient,
		*endpoint,
		connect.WithInterceptors(
			authInterceptor(*customerID),
			loggingInterceptor(logger),
		),
	)

	// Perform the requested action
	switch *action {
	case "create-and-boot":
		createAndBootVM(ctx, client, *customerID, logger)
	case "list":
		listVMs(ctx, client, logger)
	case "info":
		if *vmID == "" {
			log.Fatal("VM ID required for info action")
		}
		getVMInfo(ctx, client, *vmID, logger)
	case "shutdown":
		if *vmID == "" {
			log.Fatal("VM ID required for shutdown action")
		}
		shutdownVM(ctx, client, *vmID, logger)
	case "delete":
		if *vmID == "" {
			log.Fatal("VM ID required for delete action")
		}
		deleteVM(ctx, client, *vmID, logger)
	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

// createAndBootVM demonstrates creating and booting a VM
func createAndBootVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, customerID string, logger *slog.Logger) {
	// Create VM configuration
	vmConfig := &vmprovisionerv1.VmConfig{
		Cpu: &vmprovisionerv1.CpuConfig{
			VcpuCount:    2,
			MaxVcpuCount: 4,
		},
		Memory: &vmprovisionerv1.MemoryConfig{
			SizeBytes:       1024 * 1024 * 1024, // 1GB
			HotplugEnabled:  true,
			MaxSizeBytes:    2 * 1024 * 1024 * 1024, // 2GB max
		},
		Boot: &vmprovisionerv1.BootConfig{
			KernelPath: "/assets/vmlinux",
			InitrdPath: "/assets/initrd.img",
			KernelArgs: "console=ttyS0 reboot=k panic=1 pci=off",
		},
		Storage: []*vmprovisionerv1.StorageDevice{
			{
				Id:            "rootfs",
				Path:          "/assets/rootfs.ext4",
				ReadOnly:      false,
				IsRootDevice:  true,
				InterfaceType: "virtio-blk",
			},
		},
		Network: []*vmprovisionerv1.NetworkInterface{
			{
				Id:            "eth0",
				InterfaceType: "virtio-net",
				Mode:          vmprovisionerv1.NetworkMode_NETWORK_MODE_IPV4_ONLY,
				Ipv4Config: &vmprovisionerv1.IPv4Config{
					Dhcp: true,
				},
			},
		},
		Console: &vmprovisionerv1.ConsoleConfig{
			Enabled:     true,
			Output:      "/tmp/vm-console.log",
			ConsoleType: "serial",
		},
		Metadata: map[string]string{
			"purpose":     "example",
			"environment": "development",
		},
	}

	// Create VM
	createReq := &vmprovisionerv1.CreateVmRequest{
		Config:     vmConfig,
		CustomerId: customerID,
	}

	logger.Info("creating VM", "customer_id", customerID)
	createResp, err := client.CreateVm(ctx, connect.NewRequest(createReq))
	if err != nil {
		logger.Error("failed to create VM", "error", err)
		return
	}

	vmID := createResp.Msg.VmId
	logger.Info("VM created", "vm_id", vmID, "state", createResp.Msg.State.String())

	// Boot the VM
	bootReq := &vmprovisionerv1.BootVmRequest{
		VmId: vmID,
	}

	logger.Info("booting VM", "vm_id", vmID)
	bootResp, err := client.BootVm(ctx, connect.NewRequest(bootReq))
	if err != nil {
		logger.Error("failed to boot VM", "error", err)
		return
	}

	logger.Info("VM booted", "vm_id", vmID, "state", bootResp.Msg.State.String())

	// Get VM info to show network details
	infoReq := &vmprovisionerv1.GetVmInfoRequest{
		VmId: vmID,
	}

	infoResp, err := client.GetVmInfo(ctx, connect.NewRequest(infoReq))
	if err != nil {
		logger.Error("failed to get VM info", "error", err)
		return
	}

	if infoResp.Msg.NetworkInfo != nil {
		logger.Info("VM network info",
			"vm_id", vmID,
			"ip_address", infoResp.Msg.NetworkInfo.IpAddress,
			"mac_address", infoResp.Msg.NetworkInfo.MacAddress,
			"tap_device", infoResp.Msg.NetworkInfo.TapDevice,
		)
	}
}

// listVMs demonstrates listing all VMs
func listVMs(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, logger *slog.Logger) {
	req := &vmprovisionerv1.ListVmsRequest{
		PageSize: 100,
	}

	logger.Info("listing VMs")
	resp, err := client.ListVms(ctx, connect.NewRequest(req))
	if err != nil {
		logger.Error("failed to list VMs", "error", err)
		return
	}

	logger.Info("VMs found", "count", resp.Msg.TotalCount)
	for _, vm := range resp.Msg.Vms {
		logger.Info("VM",
			"vm_id", vm.VmId,
			"state", vm.State.String(),
			"vcpus", vm.VcpuCount,
			"memory_mb", vm.MemorySizeBytes/(1024*1024),
			"customer_id", vm.CustomerId,
			"created", time.Unix(vm.CreatedTimestamp, 0).Format(time.RFC3339),
		)
	}
}

// getVMInfo demonstrates getting detailed VM information
func getVMInfo(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, vmID string, logger *slog.Logger) {
	req := &vmprovisionerv1.GetVmInfoRequest{
		VmId: vmID,
	}

	logger.Info("getting VM info", "vm_id", vmID)
	resp, err := client.GetVmInfo(ctx, connect.NewRequest(req))
	if err != nil {
		logger.Error("failed to get VM info", "error", err)
		return
	}

	logger.Info("VM info",
		"vm_id", resp.Msg.VmId,
		"state", resp.Msg.State.String(),
		"vcpus", resp.Msg.Config.Cpu.VcpuCount,
		"memory_mb", resp.Msg.Config.Memory.SizeBytes/(1024*1024),
	)

	if resp.Msg.Metrics != nil {
		logger.Info("VM metrics",
			"cpu_usage_percent", resp.Msg.Metrics.CpuUsagePercent,
			"memory_usage_mb", resp.Msg.Metrics.MemoryUsageBytes/(1024*1024),
			"uptime_seconds", resp.Msg.Metrics.UptimeSeconds,
		)
	}

	if resp.Msg.NetworkInfo != nil {
		logger.Info("VM network",
			"ip_address", resp.Msg.NetworkInfo.IpAddress,
			"mac_address", resp.Msg.NetworkInfo.MacAddress,
			"tap_device", resp.Msg.NetworkInfo.TapDevice,
		)
	}
}

// shutdownVM demonstrates shutting down a VM
func shutdownVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, vmID string, logger *slog.Logger) {
	req := &vmprovisionerv1.ShutdownVmRequest{
		VmId:           vmID,
		Force:          false, // Try graceful shutdown first
		TimeoutSeconds: 30,
	}

	logger.Info("shutting down VM", "vm_id", vmID)
	resp, err := client.ShutdownVm(ctx, connect.NewRequest(req))
	if err != nil {
		logger.Error("failed to shutdown VM", "error", err)
		return
	}

	logger.Info("VM shutdown", "vm_id", vmID, "state", resp.Msg.State.String())
}

// deleteVM demonstrates deleting a VM
func deleteVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, vmID string, logger *slog.Logger) {
	req := &vmprovisionerv1.DeleteVmRequest{
		VmId:  vmID,
		Force: true, // Force deletion even if running
	}

	logger.Info("deleting VM", "vm_id", vmID)
	resp, err := client.DeleteVm(ctx, connect.NewRequest(req))
	if err != nil {
		logger.Error("failed to delete VM", "error", err)
		return
	}

	logger.Info("VM deleted", "vm_id", vmID, "success", resp.Msg.Success)
}

// authInterceptor adds authentication headers
func authInterceptor(customerID string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// Add customer ID header for tenant isolation
			req.Header().Set("X-Customer-ID", customerID)
			
			// You can add other auth headers here, e.g.:
			// req.Header().Set("Authorization", "Bearer " + token)
			
			return next(ctx, req)
		}
	}
}

// loggingInterceptor logs all RPC calls
func loggingInterceptor(logger *slog.Logger) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			start := time.Now()
			
			// Log request
			logger.Debug("RPC request",
				"procedure", req.Spec().Procedure,
				"protocol", req.Peer().Protocol,
			)
			
			// Execute request
			resp, err := next(ctx, req)
			
			// Log response
			duration := time.Since(start)
			if err != nil {
				logger.Error("RPC error",
					"procedure", req.Spec().Procedure,
					"duration", duration,
					"error", err,
				)
			} else {
				logger.Info("RPC success",
					"procedure", req.Spec().Procedure,
					"duration", duration,
				)
			}
			
			return resp, err
		}
	}
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}