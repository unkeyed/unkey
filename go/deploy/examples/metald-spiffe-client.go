// Package main demonstrates minimal SPIFFE/mTLS client for metald
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/spiffe"
)

/*
This example shows the simplest way to connect to metald with SPIFFE/SPIRE mTLS.

Prerequisites:
1. SPIRE agent must be running and accessible at the socket path
2. Your workload must be registered in SPIRE with appropriate SPIFFE ID
3. Metald must be running with SPIFFE mode enabled

Usage:
   go run metald-spiffe-client.go

Environment variables:
   METALD_ENDPOINT - Metald endpoint (default: https://localhost:8080)
   SPIFFE_ENDPOINT_SOCKET - SPIFFE agent socket (default: unix:///run/spire/sockets/agent.sock)
*/

func main() {
	// Get configuration from environment
	endpoint := os.Getenv("METALD_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://localhost:8080"
	}

	spiffeSocket := os.Getenv("SPIFFE_ENDPOINT_SOCKET")
	if spiffeSocket == "" {
		spiffeSocket = "unix:///run/spire/sockets/agent.sock"
	}

	// Create SPIFFE client for mTLS
	ctx := context.Background()
	spiffeClient, err := spiffe.NewWithOptions(ctx, spiffe.Options{
		SocketPath: spiffeSocket,
	})
	if err != nil {
		log.Fatalf("Failed to create SPIFFE client: %v", err)
	}
	defer spiffeClient.Close()

	fmt.Printf("Connected to SPIFFE as: %s\n", spiffeClient.ServiceName())

	// Create HTTP client with SPIFFE mTLS
	httpClient := spiffeClient.HTTPClient()

	// Create metald client
	client := vmprovisionerv1connect.NewVmServiceClient(
		httpClient,
		endpoint,
		connect.WithInterceptors(
			// Add customer ID for tenant isolation
			func(next connect.UnaryFunc) connect.UnaryFunc {
				return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					req.Header().Set("X-Customer-ID", "example-customer")
					return next(ctx, req)
				}
			},
		),
	)

	// Example: List VMs
	listReq := &vmprovisionerv1.ListVmsRequest{
		PageSize: 10,
	}

	resp, err := client.ListVms(ctx, connect.NewRequest(listReq))
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	fmt.Printf("Found %d VMs:\n", resp.Msg.TotalCount)
	for _, vm := range resp.Msg.Vms {
		fmt.Printf("  - VM ID: %s, State: %s, Customer: %s\n",
			vm.VmId, vm.State.String(), vm.CustomerId)
	}

	// Example: Create a simple VM
	createReq := &vmprovisionerv1.CreateVmRequest{
		CustomerId: "example-customer",
		Config: &vmprovisionerv1.VmConfig{
			Cpu: &vmprovisionerv1.CpuConfig{
				VcpuCount: 1,
			},
			Memory: &vmprovisionerv1.MemoryConfig{
				SizeBytes: 512 * 1024 * 1024, // 512MB
			},
			Boot: &vmprovisionerv1.BootConfig{
				KernelPath: "/assets/vmlinux",
				KernelArgs: "console=ttyS0",
			},
			Metadata: map[string]string{
				"created_by": "spiffe-example",
			},
		},
	}

	createResp, err := client.CreateVm(ctx, connect.NewRequest(createReq))
	if err != nil {
		log.Printf("Failed to create VM: %v", err)
	} else {
		fmt.Printf("\nCreated VM: %s (state: %s)\n", 
			createResp.Msg.VmId, createResp.Msg.State.String())
	}
}