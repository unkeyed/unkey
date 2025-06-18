package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/tls"
)

// AIDEV-NOTE: Example client demonstrating how to communicate with metald using mTLS via SPIFFE/SPIRE
// This shows the full lifecycle of creating and booting a VM with proper authentication

func main() {
	var (
		metaldAddr      = flag.String("addr", "https://localhost:8080", "metald server address")
		customerID      = flag.String("customer", "example-customer", "customer ID for tenant isolation")
		vmID            = flag.String("vm-id", "", "VM ID (optional, will be generated if empty)")
		tlsMode         = flag.String("tls-mode", "disabled", "TLS mode: disabled, file, or spiffe")
		tlsCert         = flag.String("tls-cert", "", "TLS certificate file (for file mode)")
		tlsKey          = flag.String("tls-key", "", "TLS key file (for file mode)")
		tlsCA           = flag.String("tls-ca", "", "TLS CA file (for file mode)")
		spiffeSocket    = flag.String("spiffe-socket", "/run/spire/sockets/agent.sock", "SPIFFE agent socket path")
		enableCaching   = flag.Bool("enable-cert-caching", true, "Enable certificate caching for file mode")
		action          = flag.String("action", "create-and-boot", "Action: create, boot, list, info, create-and-boot")
	)
	flag.Parse()

	// Set up structured logging
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	ctx := context.Background()

	// Create TLS provider based on configuration
	tlsConfig := tls.Config{
		Mode:              tls.Mode(*tlsMode),
		CertFile:          *tlsCert,
		KeyFile:           *tlsKey,
		CAFile:            *tlsCA,
		SPIFFESocketPath:  *spiffeSocket,
		EnableCertCaching: *enableCaching,
		CertCacheTTL:      5 * time.Second,
	}

	tlsProvider, err := tls.NewProvider(ctx, tlsConfig)
	if err != nil {
		log.Fatalf("Failed to create TLS provider: %v", err)
	}
	defer tlsProvider.Close()

	// Get HTTP client with appropriate TLS configuration
	httpClient := tlsProvider.HTTPClient()

	// Wrap the transport to add customer ID header
	httpClient.Transport = &customerIDTransport{
		Base:       httpClient.Transport,
		CustomerID: *customerID,
	}

	// Create Connect client for metald
	client := vmprovisionerv1connect.NewVmServiceClient(
		httpClient,
		*metaldAddr,
		connect.WithInterceptors(&loggingInterceptor{logger: logger}),
	)

	// Execute the requested action
	switch *action {
	case "create":
		createVM(ctx, client, *vmID)
	case "boot":
		if *vmID == "" {
			log.Fatal("VM ID is required for boot action")
		}
		bootVM(ctx, client, *vmID)
	case "list":
		listVMs(ctx, client)
	case "info":
		if *vmID == "" {
			log.Fatal("VM ID is required for info action")
		}
		getVMInfo(ctx, client, *vmID)
	case "create-and-boot":
		vmID := createVM(ctx, client, *vmID)
		time.Sleep(2 * time.Second) // Give the VM time to be fully created
		bootVM(ctx, client, vmID)
		getVMInfo(ctx, client, vmID)
	default:
		log.Fatalf("Unknown action: %s", *action)
	}
}

// customerIDTransport adds the Authorization header to all requests
type customerIDTransport struct {
	Base       http.RoundTripper
	CustomerID string
}

func (t *customerIDTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid modifying the original
	req2 := req.Clone(req.Context())
	if req2.Header == nil {
		req2.Header = make(http.Header)
	}
	
	// Set Authorization header with development token format
	// In production, this would use a real JWT or API key
	req2.Header.Set("Authorization", fmt.Sprintf("Bearer dev_customer_%s", t.CustomerID))
	
	// Debug: Log the headers being sent
	if debug := os.Getenv("DEBUG"); debug != "" {
		fmt.Printf("DEBUG: Sending request to %s with headers:\n", req2.URL)
		for k, v := range req2.Header {
			fmt.Printf("  %s: %v\n", k, v)
		}
	}
	
	// Use the base transport, or default if nil
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	return base.RoundTrip(req2)
}

// loggingInterceptor logs all RPC calls
type loggingInterceptor struct {
	logger *slog.Logger
}

func (i *loggingInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		start := time.Now()
		i.logger.LogAttrs(ctx, slog.LevelInfo, "RPC call started",
			slog.String("procedure", req.Spec().Procedure),
		)

		resp, err := next(ctx, req)

		i.logger.LogAttrs(ctx, slog.LevelInfo, "RPC call completed",
			slog.String("procedure", req.Spec().Procedure),
			slog.Duration("duration", time.Since(start)),
			slog.Bool("error", err != nil),
		)

		return resp, err
	}
}

func (i *loggingInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *loggingInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func createVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, vmID string) string {
	// Create VM configuration
	config := &vmprovisionerv1.VmConfig{
		Cpu: &vmprovisionerv1.CpuConfig{
			VcpuCount:    2,
			MaxVcpuCount: 4,
		},
		Memory: &vmprovisionerv1.MemoryConfig{
			SizeBytes:       1 * 1024 * 1024 * 1024, // 1GB
			HotplugEnabled:  true,
			MaxSizeBytes:    4 * 1024 * 1024 * 1024, // 4GB max
		},
		Boot: &vmprovisionerv1.BootConfig{
			KernelPath: "/opt/vm-assets/vmlinux",
			// InitrdPath is optional - commenting out as we don't have an initrd
			// InitrdPath: "/opt/vm-assets/initrd.img",
			KernelArgs: "console=ttyS0 reboot=k panic=1 pci=off",
		},
		Storage: []*vmprovisionerv1.StorageDevice{
			{
				Id:            "rootfs",
				Path:          "/opt/vm-assets/rootfs.ext4",
				ReadOnly:      false,
				IsRootDevice:  true,
				InterfaceType: "virtio-blk",
			},
		},
		Network: []*vmprovisionerv1.NetworkInterface{
			{
				Id:            "eth0",
				InterfaceType: "virtio-net",
				Mode:          vmprovisionerv1.NetworkMode_NETWORK_MODE_DUAL_STACK,
				Ipv4Config: &vmprovisionerv1.IPv4Config{
					Dhcp: true,
				},
				Ipv6Config: &vmprovisionerv1.IPv6Config{
					Slaac:             true,
					PrivacyExtensions: true,
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

	req := &vmprovisionerv1.CreateVmRequest{
		VmId:   vmID,
		Config: config,
	}

	resp, err := client.CreateVm(ctx, connect.NewRequest(req))
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	fmt.Printf("VM created successfully:\n")
	fmt.Printf("  VM ID: %s\n", resp.Msg.VmId)
	fmt.Printf("  State: %s\n", resp.Msg.State.String())

	return resp.Msg.VmId
}

func bootVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, vmID string) {
	req := &vmprovisionerv1.BootVmRequest{
		VmId: vmID,
	}

	resp, err := client.BootVm(ctx, connect.NewRequest(req))
	if err != nil {
		log.Fatalf("Failed to boot VM: %v", err)
	}

	fmt.Printf("VM booted successfully:\n")
	fmt.Printf("  Success: %v\n", resp.Msg.Success)
	fmt.Printf("  State: %s\n", resp.Msg.State.String())
}

func listVMs(ctx context.Context, client vmprovisionerv1connect.VmServiceClient) {
	req := &vmprovisionerv1.ListVmsRequest{
		PageSize: 50,
	}

	resp, err := client.ListVms(ctx, connect.NewRequest(req))
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	fmt.Printf("VMs (total: %d):\n", resp.Msg.TotalCount)
	for _, vm := range resp.Msg.Vms {
		fmt.Printf("  - %s: %s (CPUs: %d, Memory: %d MB)\n",
			vm.VmId,
			vm.State.String(),
			vm.VcpuCount,
			vm.MemorySizeBytes/(1024*1024),
		)
	}
}

func getVMInfo(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, vmID string) {
	req := &vmprovisionerv1.GetVmInfoRequest{
		VmId: vmID,
	}

	resp, err := client.GetVmInfo(ctx, connect.NewRequest(req))
	if err != nil {
		log.Fatalf("Failed to get VM info: %v", err)
	}

	fmt.Printf("VM Information:\n")
	fmt.Printf("  VM ID: %s\n", resp.Msg.VmId)
	fmt.Printf("  State: %s\n", resp.Msg.State.String())

	if resp.Msg.Config != nil {
		fmt.Printf("  Configuration:\n")
		fmt.Printf("    CPUs: %d (max: %d)\n", resp.Msg.Config.Cpu.VcpuCount, resp.Msg.Config.Cpu.MaxVcpuCount)
		fmt.Printf("    Memory: %d MB\n", resp.Msg.Config.Memory.SizeBytes/(1024*1024))
		fmt.Printf("    Storage devices: %d\n", len(resp.Msg.Config.Storage))
		fmt.Printf("    Network interfaces: %d\n", len(resp.Msg.Config.Network))
	}

	if resp.Msg.Metrics != nil {
		fmt.Printf("  Metrics:\n")
		fmt.Printf("    CPU usage: %.2f%%\n", resp.Msg.Metrics.CpuUsagePercent)
		fmt.Printf("    Memory usage: %d MB\n", resp.Msg.Metrics.MemoryUsageBytes/(1024*1024))
		fmt.Printf("    Uptime: %d seconds\n", resp.Msg.Metrics.UptimeSeconds)
	}

	if resp.Msg.NetworkInfo != nil {
		fmt.Printf("  Network:\n")
		fmt.Printf("    IP: %s\n", resp.Msg.NetworkInfo.IpAddress)
		fmt.Printf("    MAC: %s\n", resp.Msg.NetworkInfo.MacAddress)
		fmt.Printf("    TAP: %s\n", resp.Msg.NetworkInfo.TapDevice)
	}
}