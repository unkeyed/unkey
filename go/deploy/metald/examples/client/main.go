package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"

	"google.golang.org/protobuf/encoding/protojson"
)

func main() {
	// Create ConnectRPC client
	client := vmprovisionerv1connect.NewVmServiceClient(
		http.DefaultClient,
		"http://localhost:8080",
	)

	// Example 1: Basic VM Configuration (2 CPU, 2GB RAM)
	basicVMConfig := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount: 2,
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes: 2147483648, // 2GB in bytes
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/opt/vm-assets/vmlinux",
			KernelArgs: "console=ttyS0 root=/dev/vda rw init=/bin/sh",
		},
		Storage: []*metaldv1.StorageDevice{
			{
				Id:           "rootfs",
				Path:         "/opt/vm-assets/rootfs.ext4",
				IsRootDevice: true,
				ReadOnly:     false,
			},
		},
	}

	// Example 2: Advanced VM Configuration with Network
	advancedVMConfig := &metaldv1.VmConfig{
		Cpu: &metaldv1.CpuConfig{
			VcpuCount:    2,
			MaxVcpuCount: 4,
			Topology: &metaldv1.CpuTopology{
				Sockets:        1,
				CoresPerSocket: 2,
				ThreadsPerCore: 1,
			},
		},
		Memory: &metaldv1.MemoryConfig{
			SizeBytes:      2147483648, // 2GB
			HotplugEnabled: false,
			MaxSizeBytes:   4294967296, // 4GB max
		},
		Boot: &metaldv1.BootConfig{
			KernelPath: "/opt/vm-assets/vmlinux",
			InitrdPath: "/opt/vm-assets/initrd",
			KernelArgs: "console=ttyS0 root=/dev/vda rw",
		},
		Storage: []*metaldv1.StorageDevice{
			{
				Id:            "rootfs",
				Path:          "/opt/vm-assets/rootfs.ext4",
				IsRootDevice:  true,
				ReadOnly:      false,
				InterfaceType: "virtio-blk",
			},
			{
				Id:            "data",
				Path:          "/opt/vm-assets/data.ext4",
				IsRootDevice:  false,
				ReadOnly:      false,
				InterfaceType: "virtio-blk",
			},
		},
		Network: []*metaldv1.NetworkInterface{
			{
				Id:            "eth0",
				MacAddress:    "AA:FC:00:00:00:01",
				TapDevice:     "tap0",
				InterfaceType: "virtio-net",
			},
		},
		Console: &metaldv1.ConsoleConfig{
			Enabled:     true,
			Output:      "/tmp/vm-console.log",
			ConsoleType: "serial",
		},
		Metadata: map[string]string{
			"environment": "development",
			"project":     "vmm-controlplane",
			"created_by":  "example-client",
		},
	}

	// Create requests with customer ID
	basicRequest := &metaldv1.CreateVmRequest{
		VmId:       "basic-vm-example",
		Config:     basicVMConfig,
		CustomerId: "acme-corp", // Example customer ID
	}

	advancedRequest := &metaldv1.CreateVmRequest{
		VmId:       "advanced-vm-example",
		Config:     advancedVMConfig,
		CustomerId: "acme-corp", // Example customer ID
	}

	fmt.Println("=== ConnectRPC Client VM Creation Examples ===")

	// Display Basic Configuration
	fmt.Println("1. BASIC VM CONFIGURATION")
	fmt.Println("=============================")
	displayVMCreationData(client, basicRequest)

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Display Advanced Configuration
	fmt.Println("2. ADVANCED VM CONFIGURATION")
	fmt.Println("==============================")
	displayVMCreationData(client, advancedRequest)

	fmt.Println("\n" + strings.Repeat("=", 60) + "\n")

	// Show what the actual ConnectRPC call would look like
	fmt.Println("3. ACTUAL CONNECTRPC CALL EXAMPLE")
	fmt.Println("===================================")
	showConnectRPCCall(client, basicRequest)
}

func displayVMCreationData(client vmprovisionerv1connect.VmServiceClient, req *metaldv1.CreateVmRequest) {
	// Convert to JSON for inspection
	jsonData, err := protojson.MarshalOptions{
		Multiline: true,
		Indent:    "  ",
	}.Marshal(req)
	if err != nil {
		log.Fatalf("Failed to marshal request: %v", err)
	}

	fmt.Printf("ConnectRPC Request Data:\n")
	fmt.Printf("VM ID: %s\n", req.VmId)
	fmt.Printf("Config JSON:\n%s\n", string(jsonData))

	// Show resource summary
	cpu := req.Config.Cpu
	memory := req.Config.Memory
	fmt.Printf("Resource Summary:\n")
	fmt.Printf("  CPU: %d vCPUs", cpu.VcpuCount)
	if cpu.MaxVcpuCount > 0 {
		fmt.Printf(" (max: %d)", cpu.MaxVcpuCount)
	}
	fmt.Printf("\n")
	fmt.Printf("  Memory: %d MB (%d GB)\n", memory.SizeBytes/(1024*1024), memory.SizeBytes/(1024*1024*1024))
	fmt.Printf("  Storage Devices: %d\n", len(req.Config.Storage))
	fmt.Printf("  Network Interfaces: %d\n", len(req.Config.Network))

	// Show file dependencies
	fmt.Printf("File Dependencies:\n")
	fmt.Printf("  Kernel: %s\n", req.Config.Boot.KernelPath)
	if req.Config.Boot.InitrdPath != "" {
		fmt.Printf("  Initrd: %s\n", req.Config.Boot.InitrdPath)
	}
	for i, storage := range req.Config.Storage {
		fmt.Printf("  Storage[%d]: %s", i, storage.Path)
		if storage.IsRootDevice {
			fmt.Printf(" (root)")
		}
		fmt.Printf("\n")
	}
}

func showConnectRPCCall(client vmprovisionerv1connect.VmServiceClient, req *metaldv1.CreateVmRequest) {
	fmt.Printf("Go Code Example with Authentication:\n")
	fmt.Printf(`
// Create ConnectRPC client
client := metaldv1connect.NewVmServiceClient(
    http.DefaultClient,
    "http://localhost:8080",
)

// Create VM request
req := connect.NewRequest(req)

// Add authentication header (development token format)
req.Header().Set("Authorization", "Bearer dev_customer_acme-corp")

// Make the call (commented out to prevent actual execution)
// resp, err := client.CreateVm(context.Background(), req)
// if err != nil {
//     log.Fatalf("CreateVm failed: %%v", err)
// }
//
// fmt.Printf("VM Created: %%s, State: %%s\n",
//     resp.Msg.VmId, resp.Msg.State.String())

fmt.Println("Call would be made to: POST /vmm.v1.VmService/CreateVm")
`)

	// Show equivalent curl command
	fmt.Printf("Equivalent curl command:\n")
	jsonData, _ := protojson.MarshalOptions{Multiline: false}.Marshal(req)

	// Pretty print the JSON for curl
	var prettyJSON map[string]interface{}
	json.Unmarshal(jsonData, &prettyJSON)
	prettyBytes, _ := json.MarshalIndent(prettyJSON, "", "  ")

	fmt.Printf(`
curl -X POST http://localhost:8080/vmm.v1.VmService/CreateVm \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer dev_customer_acme-corp" \
  -d '%s'
`, string(prettyBytes))
}

// Example of how you'd actually make the call (if you wanted to)
func createVMExample(client vmprovisionerv1connect.VmServiceClient, req *metaldv1.CreateVmRequest) {
	fmt.Println("=== ACTUAL VM CREATION (COMMENTED OUT) ===")
	fmt.Println("// Uncomment the following code to actually create a VM:")
	fmt.Printf(`
	ctx := context.Background()

	// Create the request
	connectReq := connect.NewRequest(req)

	// Add any headers if needed
	connectReq.Header().Set("User-Agent", "vmm-client/1.0")

	// Make the call
	resp, err := client.CreateVm(ctx, connectReq)
	if err != nil {
		log.Fatalf("CreateVm failed: %%v", err)
	}

	fmt.Printf("VM Created Successfully!\n")
	fmt.Printf("VM ID: %%s\n", resp.Msg.VmId)
	fmt.Printf("State: %%s\n", resp.Msg.State.String())

	// Now you could boot it
	bootReq := connect.NewRequest(&metaldv1.BootVmRequest{
		VmId: resp.Msg.VmId,
	})

	bootResp, err := client.BootVm(ctx, bootReq)
	if err != nil {
		log.Fatalf("BootVm failed: %%v", err)
	}

	fmt.Printf("VM Booted! State: %%s\n", bootResp.Msg.State.String())
`)
}
