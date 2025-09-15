package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/unkeyed/unkey/go/apps/billaged/client"
	billingv1 "github.com/unkeyed/unkey/go/gen/proto/billaged/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AIDEV-NOTE: CLI tool demonstrating billaged client usage with SPIFFE integration
// This provides a command-line interface for billing operations with proper tenant isolation

func main() {
	var (
		serverAddr   = flag.String("server", getEnvOrDefault("UNKEY_BILLAGED_SERVER_ADDRESS", "https://localhost:8081"), "billaged server address")
		userID       = flag.String("user", getEnvOrDefault("UNKEY_BILLAGED_USER_ID", "cli-user"), "user ID for authentication")
		tenantID     = flag.String("tenant", getEnvOrDefault("UNKEY_BILLAGED_TENANT_ID", "cli-tenant"), "tenant ID for data scoping")
		tlsMode      = flag.String("tls-mode", getEnvOrDefault("UNKEY_BILLAGED_TLS_MODE", "spiffe"), "TLS mode: disabled, file, or spiffe")
		spiffeSocket = flag.String("spiffe-socket", getEnvOrDefault("UNKEY_BILLAGED_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"), "SPIFFE agent socket path")
		tlsCert      = flag.String("tls-cert", "", "TLS certificate file (for file mode)")
		tlsKey       = flag.String("tls-key", "", "TLS key file (for file mode)")
		tlsCA        = flag.String("tls-ca", "", "TLS CA file (for file mode)")
		timeout      = flag.Duration("timeout", 30*time.Second, "request timeout")
		jsonOutput   = flag.Bool("json", false, "output results as JSON")
	)
	flag.Parse()

	if flag.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Create billaged client
	config := client.Config{
		ServerAddress:    *serverAddr,
		UserID:           *userID,
		TenantID:         *tenantID,
		TLSMode:          *tlsMode,
		SPIFFESocketPath: *spiffeSocket,
		TLSCertFile:      *tlsCert,
		TLSKeyFile:       *tlsKey,
		TLSCAFile:        *tlsCA,
		Timeout:          *timeout,
	}

	billingClient, err := client.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create billaged client: %v", err)
	}
	defer billingClient.Close()

	// Execute command
	command := flag.Arg(0)
	switch command {
	case "send-metrics":
		handleSendMetrics(ctx, billingClient, *jsonOutput)
	case "heartbeat":
		handleHeartbeat(ctx, billingClient, *jsonOutput)
	case "notify-started":
		handleNotifyVmStarted(ctx, billingClient, *jsonOutput)
	case "notify-stopped":
		handleNotifyVmStopped(ctx, billingClient, *jsonOutput)
	case "notify-gap":
		handleNotifyPossibleGap(ctx, billingClient, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`billaged-cli - CLI tool for billaged operations

Usage: %s [flags] <command> [args...]

Commands:
  send-metrics                Send VM metrics batch
  heartbeat                   Send heartbeat with active VMs
  notify-started <vm-id>      Notify that a VM has started
  notify-stopped <vm-id>      Notify that a VM has stopped
  notify-gap <vm-id>          Notify about a possible gap in metrics

Environment Variables:
  UNKEY_BILLAGED_SERVER_ADDRESS  Server address (default: https://localhost:8081)
  UNKEY_BILLAGED_USER_ID         User ID for authentication (default: cli-user)
  UNKEY_BILLAGED_TENANT_ID       Tenant ID for data scoping (default: cli-tenant)
  UNKEY_BILLAGED_TLS_MODE        TLS mode (default: spiffe)
  UNKEY_BILLAGED_SPIFFE_SOCKET   SPIFFE socket path (default: /var/lib/spire/agent/agent.sock)

Examples:
  # Send heartbeat with SPIFFE authentication
  %s -user=prod-user-123 -tenant=prod-tenant-456 heartbeat

  # Notify VM started
  %s notify-started vm-12345

  # Send metrics batch
  %s send-metrics

  # Get response with JSON output
  %s heartbeat -json

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func handleSendMetrics(ctx context.Context, billingClient *client.Client, jsonOutput bool) {
	// Example metrics batch - in real usage, this would be provided via CLI args or file
	metrics := []*billingv1.VMMetrics{
		{
			Timestamp:        timestamppb.Now(),
			CpuTimeNanos:     1000000000,        // 1 second of CPU time
			MemoryUsageBytes: 512 * 1024 * 1024, // 512MB
			DiskReadBytes:    1024 * 1024,       // 1MB
			DiskWriteBytes:   512 * 1024,        // 512KB
			NetworkRxBytes:   2048,              // 2KB
			NetworkTxBytes:   1024,              // 1KB
		},
	}

	req := &client.SendMetricsBatchRequest{
		VmID:       "example-vm-123",
		CustomerID: billingClient.GetTenantID(),
		Metrics:    metrics,
	}

	resp, err := billingClient.SendMetricsBatch(ctx, req)
	if err != nil {
		log.Fatalf("Failed to send metrics: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Metrics sent:\n")
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Message: %s\n", resp.Message)
		fmt.Printf("  Metrics count: %d\n", len(metrics))
	}
}

func handleHeartbeat(ctx context.Context, billingClient *client.Client, jsonOutput bool) {
	// Example heartbeat - in real usage, this would get actual active VMs
	req := &client.SendHeartbeatRequest{
		InstanceID: "metald-instance-1",
		ActiveVMs:  []string{"vm-123", "vm-456", "vm-789"},
	}

	resp, err := billingClient.SendHeartbeat(ctx, req)
	if err != nil {
		log.Fatalf("Failed to send heartbeat: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Heartbeat sent:\n")
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Active VMs: %d\n", len(req.ActiveVMs))
	}
}

func handleNotifyVmStarted(ctx context.Context, billingClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for notify-started command")
	}
	vmID := flag.Arg(1)

	req := &client.NotifyVmStartedRequest{
		VmID:       vmID,
		CustomerID: billingClient.GetTenantID(),
		StartTime:  time.Now().Unix(),
	}

	resp, err := billingClient.NotifyVmStarted(ctx, req)
	if err != nil {
		log.Fatalf("Failed to notify VM started: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("VM started notification:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Start time: %d\n", req.StartTime)
	}
}

func handleNotifyVmStopped(ctx context.Context, billingClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for notify-stopped command")
	}
	vmID := flag.Arg(1)

	req := &client.NotifyVmStoppedRequest{
		VmID:     vmID,
		StopTime: time.Now().Unix(),
	}

	resp, err := billingClient.NotifyVmStopped(ctx, req)
	if err != nil {
		log.Fatalf("Failed to notify VM stopped: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("VM stopped notification:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Stop time: %d\n", req.StopTime)
	}
}

func handleNotifyPossibleGap(ctx context.Context, billingClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for notify-gap command")
	}
	vmID := flag.Arg(1)

	// Default to a 5-minute gap ending now
	gapDuration := 5 * time.Minute
	resumeTime := time.Now()
	lastSent := resumeTime.Add(-gapDuration)

	// Allow custom gap duration from CLI args
	if flag.NArg() > 2 {
		if minutes, err := strconv.Atoi(flag.Arg(2)); err == nil {
			gapDuration = time.Duration(minutes) * time.Minute
			lastSent = resumeTime.Add(-gapDuration)
		}
	}

	req := &client.NotifyPossibleGapRequest{
		VmID:       vmID,
		LastSent:   lastSent.Unix(),
		ResumeTime: resumeTime.Unix(),
	}

	resp, err := billingClient.NotifyPossibleGap(ctx, req)
	if err != nil {
		log.Fatalf("Failed to notify possible gap: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Possible gap notification:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Gap duration: %v\n", gapDuration)
		fmt.Printf("  Last sent: %s\n", lastSent.Format(time.RFC3339))
		fmt.Printf("  Resume time: %s\n", resumeTime.Format(time.RFC3339))
	}
}

func outputJSON(data interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
