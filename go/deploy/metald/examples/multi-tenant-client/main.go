package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	metaldv1 "github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"

	"connectrpc.com/connect"
	"go.opentelemetry.io/otel/baggage"
)

func main() {
	// Example of multi-tenant VM creation with baggage context
	client := vmprovisionerv1connect.NewVmServiceClient(
		http.DefaultClient,
		"http://localhost:8092",
	)

	// Create tenant context with baggage
	ctx := context.Background()

	// Scenario 1: Acme Corp tenant
	acmeBaggage, err := baggage.Parse("tenant_id=acme-corp,user_id=alice@acme.com,workspace_id=prod,plan=enterprise")
	if err != nil {
		log.Fatal(err)
	}
	acmeCtx := baggage.ContextWithBaggage(ctx, acmeBaggage)

	fmt.Println("=== Creating VM for Acme Corp (Enterprise Plan) ===")
	acmeVM, err := createVM(acmeCtx, client, "acme-vm", 4, 4*1024*1024*1024, "dev_customer_acme-corp") // 4 vCPU, 4GB
	if err != nil {
		log.Printf("Failed to create Acme VM: %v", err)
	} else {
		fmt.Printf("✅ Created Acme VM: %s\n", acmeVM.VmId)
	}

	// Scenario 2: StartupCo tenant (smaller resources)
	startupBaggage, err := baggage.Parse("tenant_id=startup-co,user_id=bob@startup.co,workspace_id=dev,plan=starter")
	if err != nil {
		log.Fatal(err)
	}
	startupCtx := baggage.ContextWithBaggage(ctx, startupBaggage)

	fmt.Println("\n=== Creating VM for StartupCo (Starter Plan) ===")
	startupVM, err := createVM(startupCtx, client, "startup-vm", 1, 512*1024*1024, "dev_customer_startup-co") // 1 vCPU, 512MB
	if err != nil {
		log.Printf("Failed to create Startup VM: %v", err)
	} else {
		fmt.Printf("✅ Created Startup VM: %s\n", startupVM.VmId)
	}

	// Scenario 3: Attempt cross-tenant access (should be logged/audited)
	fmt.Println("\n=== Cross-tenant access example ===")
	if acmeVM != nil && startupVM != nil {
		// Try to access StartupCo VM with Acme authentication (this should be denied)
		fmt.Printf("Acme trying to access StartupCo VM %s...\n", startupVM.VmId)
		req := connect.NewRequest(&metaldv1.GetVmInfoRequest{
			VmId: startupVM.VmId,
		})
		req.Header().Set("Authorization", "Bearer dev_customer_acme-corp")
		
		_, err := client.GetVmInfo(acmeCtx, req)
		if err != nil {
			fmt.Printf("⚠️  Access denied (expected): %v\n", err)
		} else {
			fmt.Printf("🔓 Access granted (potential security issue!)\n")
		}
	}

	fmt.Println("\n=== Check metald logs for tenant context ===")
	fmt.Println("Look for log entries with tenant_id, user_id, and workspace_id fields")
	fmt.Println("This demonstrates how all VM operations are audited with tenant context")
}

func createVM(ctx context.Context, client vmprovisionerv1connect.VmServiceClient, name string, vcpus int32, memoryBytes int64, customerToken string) (*metaldv1.CreateVmResponse, error) {
	req := &metaldv1.CreateVmRequest{
		VmId: name,
		Config: &metaldv1.VmConfig{
			Cpu: &metaldv1.CpuConfig{
				VcpuCount: vcpus,
			},
			Memory: &metaldv1.MemoryConfig{
				SizeBytes: memoryBytes,
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
		},
	}

	// Create ConnectRPC request and add authentication
	connectReq := connect.NewRequest(req)
	connectReq.Header().Set("Authorization", "Bearer "+customerToken)

	resp, err := client.CreateVm(ctx, connectReq)
	if err != nil {
		return nil, err
	}

	// Pretty print the response
	responseJSON, _ := json.MarshalIndent(resp.Msg, "", "  ")
	fmt.Printf("Response: %s\n", responseJSON)

	return resp.Msg, nil
}
