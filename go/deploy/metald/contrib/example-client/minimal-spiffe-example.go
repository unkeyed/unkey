// +build ignore

package main

import (
	"context"
	"fmt"
	"log"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/deploy/metald/gen/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/deploy/pkg/spiffe"
)

// AIDEV-NOTE: Minimal example showing SPIFFE-only client setup
// This demonstrates the simplest way to connect to metald with mTLS

func main() {
	ctx := context.Background()

	// Create SPIFFE client
	spiffeClient, err := spiffe.New(ctx)
	if err != nil {
		log.Fatalf("Failed to create SPIFFE client: %v", err)
	}
	defer spiffeClient.Close()

	// Get HTTP client with mTLS
	httpClient := spiffeClient.HTTPClient()

	// Create metald client
	client := vmprovisionerv1connect.NewVmServiceClient(
		httpClient,
		"https://metald:8080", // Use service name for internal communication
	)

	// List VMs
	resp, err := client.ListVms(ctx, connect.NewRequest(&vmprovisionerv1.ListVmsRequest{
		PageSize: 10,
	}))
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	fmt.Printf("Found %d VMs\n", resp.Msg.TotalCount)
	for _, vm := range resp.Msg.Vms {
		fmt.Printf("- %s: %s\n", vm.VmId, vm.State.String())
	}
}