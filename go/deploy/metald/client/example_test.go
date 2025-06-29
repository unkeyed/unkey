package client_test

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/client"
	vmprovisionerv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
)

// AIDEV-NOTE: Example demonstrating metald client usage with SPIFFE authentication
// This shows the complete VM lifecycle using the high-level client interface

func ExampleClient_CreateAndBootVM() {
	ctx := context.Background()

	// Create client with SPIFFE authentication
	config := client.Config{
		ServerAddress:    "https://metald:8080",
		TenantID:         "example-customer-123",
		TLSMode:          "spiffe",
		SPIFFESocketPath: "/var/lib/spire/agent/agent.sock",
		Timeout:          30 * time.Second,
	}

	metaldClient, err := client.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create metald client: %v", err)
	}
	defer metaldClient.Close()

	// Create VM configuration
	vmConfig := &vmprovisionerv1.VmConfig{
		Cpu: &vmprovisionerv1.CpuConfig{
			VcpuCount:    2,
			MaxVcpuCount: 4,
		},
		Memory: &vmprovisionerv1.MemoryConfig{
			SizeBytes:      1 * 1024 * 1024 * 1024, // 1GB
			HotplugEnabled: true,
			MaxSizeBytes:   4 * 1024 * 1024 * 1024, // 4GB max
		},
		Boot: &vmprovisionerv1.BootConfig{
			KernelPath: "/opt/vm-assets/vmlinux",
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
			"tenant":      config.TenantID,
		},
	}

	// Create the VM
	createReq := &client.CreateVMRequest{
		VMID:   "", // Let metald generate a VM ID
		Config: vmConfig,
	}

	createResp, err := metaldClient.CreateVM(ctx, createReq)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	fmt.Printf("VM created: %s (state: %s)\n", createResp.VMID, createResp.State)

	// Boot the VM
	bootResp, err := metaldClient.BootVM(ctx, createResp.VMID)
	if err != nil {
		log.Fatalf("Failed to boot VM: %v", err)
	}

	fmt.Printf("VM booted: success=%v (state: %s)\n", bootResp.Success, bootResp.State)

	// Get VM info
	vmInfo, err := metaldClient.GetVMInfo(ctx, createResp.VMID)
	if err != nil {
		log.Fatalf("Failed to get VM info: %v", err)
	}

	fmt.Printf("VM info: %s (state: %s)\n", vmInfo.VMID, vmInfo.State)
	if vmInfo.Config != nil {
		fmt.Printf("  CPUs: %d, Memory: %d MB\n",
			vmInfo.Config.Cpu.VcpuCount,
			vmInfo.Config.Memory.SizeBytes/(1024*1024))
	}

	// List all VMs
	listReq := &client.ListVMsRequest{
		PageSize: 10,
	}

	listResp, err := metaldClient.ListVMs(ctx, listReq)
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	fmt.Printf("Total VMs for tenant %s: %d\n", config.TenantID, listResp.TotalCount)

	// Shutdown the VM
	shutdownReq := &client.ShutdownVMRequest{
		VMID:           createResp.VMID,
		Force:          false,
		TimeoutSeconds: 30,
	}

	shutdownResp, err := metaldClient.ShutdownVM(ctx, shutdownReq)
	if err != nil {
		log.Fatalf("Failed to shutdown VM: %v", err)
	}

	fmt.Printf("VM shutdown: success=%v (state: %s)\n", shutdownResp.Success, shutdownResp.State)

	// Output:
	// VM created: vm-123 (state: VM_STATE_CREATED)
	// VM booted: success=true (state: VM_STATE_RUNNING)
	// VM info: vm-123 (state: VM_STATE_RUNNING)
	//   CPUs: 2, Memory: 1024 MB
	// Total VMs for customer example-customer-123: 1
	// VM shutdown: success=true (state: VM_STATE_SHUTDOWN)
}

func ExampleClient_WithTLSModes() {
	ctx := context.Background()

	// Example 1: SPIFFE mode (production default)
	spiffeConfig := client.Config{
		ServerAddress:    "https://metald:8080",
		TenantID:         "prod-customer",
		TLSMode:          "spiffe",
		SPIFFESocketPath: "/var/lib/spire/agent/agent.sock",
	}

	spiffeClient, err := client.New(ctx, spiffeConfig)
	if err != nil {
		log.Printf("SPIFFE client error: %v", err)
	} else {
		defer spiffeClient.Close()
		fmt.Printf("SPIFFE client created for customer: %s\n", spiffeClient.GetTenantID())
	}

	// Example 2: File-based TLS mode
	fileConfig := client.Config{
		ServerAddress: "https://metald:8080",
		TenantID:      "test-customer",
		TLSMode:       "file",
		TLSCertFile:   "/etc/ssl/certs/client.crt",
		TLSKeyFile:    "/etc/ssl/private/client.key",
		TLSCAFile:     "/etc/ssl/certs/ca.crt",
	}

	fileClient, err := client.New(ctx, fileConfig)
	if err != nil {
		log.Printf("File TLS client error: %v", err)
	} else {
		defer fileClient.Close()
		fmt.Printf("File TLS client created for customer: %s\n", fileClient.GetTenantID())
	}

	// Example 3: Disabled TLS mode (development only)
	devConfig := client.Config{
		ServerAddress: "http://localhost:8080",
		TenantID:      "dev-customer",
		TLSMode:       "disabled",
	}

	devClient, err := client.New(ctx, devConfig)
	if err != nil {
		log.Printf("Development client error: %v", err)
	} else {
		defer devClient.Close()
		fmt.Printf("Development client created for customer: %s\n", devClient.GetTenantID())
	}

	// Output:
	// SPIFFE client created for customer: prod-customer
	// File TLS client created for customer: test-customer
	// Development client created for customer: dev-customer
}

func ExampleClient_VMLifecycleOperations() {
	ctx := context.Background()

	config := client.Config{
		ServerAddress:    "https://metald:8080",
		TenantID:         "lifecycle-demo",
		TLSMode:          "spiffe",
		SPIFFESocketPath: "/var/lib/spire/agent/agent.sock",
	}

	metaldClient, err := client.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}
	defer metaldClient.Close()

	// Assume we have a VM ID from previous creation
	vmID := "vm-lifecycle-example"

	// Pause VM
	pauseResp, err := metaldClient.PauseVM(ctx, vmID)
	if err != nil {
		log.Printf("Pause failed: %v", err)
	} else {
		fmt.Printf("VM paused: success=%v (state: %s)\n", pauseResp.Success, pauseResp.State)
	}

	// Resume VM
	resumeResp, err := metaldClient.ResumeVM(ctx, vmID)
	if err != nil {
		log.Printf("Resume failed: %v", err)
	} else {
		fmt.Printf("VM resumed: success=%v (state: %s)\n", resumeResp.Success, resumeResp.State)
	}

	// Reboot VM
	rebootReq := &client.RebootVMRequest{
		VMID:  vmID,
		Force: false,
	}

	rebootResp, err := metaldClient.RebootVM(ctx, rebootReq)
	if err != nil {
		log.Printf("Reboot failed: %v", err)
	} else {
		fmt.Printf("VM rebooted: success=%v (state: %s)\n", rebootResp.Success, rebootResp.State)
	}

	// Delete VM
	deleteReq := &client.DeleteVMRequest{
		VMID:  vmID,
		Force: false,
	}

	deleteResp, err := metaldClient.DeleteVM(ctx, deleteReq)
	if err != nil {
		log.Printf("Delete failed: %v", err)
	} else {
		fmt.Printf("VM deleted: success=%v\n", deleteResp.Success)
	}

	// Output:
	// VM paused: success=true (state: VM_STATE_PAUSED)
	// VM resumed: success=true (state: VM_STATE_RUNNING)
	// VM rebooted: success=true (state: VM_STATE_RUNNING)
	// VM deleted: success=true
}
