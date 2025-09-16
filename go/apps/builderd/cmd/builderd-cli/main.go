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

	"github.com/unkeyed/unkey/go/apps/builderd/client"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/builderd/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func main() {
	var (
		serverAddr   = flag.String("server", getEnvOrDefault("UNKEY_BUILDERD_SERVER_ADDRESS", "https://localhost:8082"), "builderd server address")
		tlsMode      = flag.String("tls-mode", getEnvOrDefault("UNKEY_BUILDERD_TLS_MODE", "spiffe"), "TLS mode: disabled, file, or spiffe")
		spiffeSocket = flag.String("spiffe-socket", getEnvOrDefault("UNKEY_BUILDERD_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"), "SPIFFE agent socket path")
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

	// Create builderd client
	config := client.Config{
		ServerAddress:    *serverAddr,
		TLSMode:          *tlsMode,
		SPIFFESocketPath: *spiffeSocket,
		TLSCertFile:      *tlsCert,
		TLSKeyFile:       *tlsKey,
		TLSCAFile:        *tlsCA,
		Timeout:          *timeout,
	}

	builderClient, err := client.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create builderd client: %v", err)
	}
	defer builderClient.Close()

	// Execute command
	command := flag.Arg(0)
	switch command {
	case "create-build":
		handleCreateBuild(ctx, builderClient, *jsonOutput)
	case "get-build":
		handleGetBuild(ctx, builderClient, *jsonOutput)
	case "list-builds":
		handleListBuilds(ctx, builderClient, *jsonOutput)
	case "cancel-build":
		handleCancelBuild(ctx, builderClient, *jsonOutput)
	case "delete-build":
		handleDeleteBuild(ctx, builderClient, *jsonOutput)
	case "stream-logs":
		handleStreamLogs(ctx, builderClient, *jsonOutput)
	case "get-stats":
		handleGetStats(ctx, builderClient, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`builderd-cli - CLI tool for builderd operations

Usage: %s [flags] <command> [args...]

Commands:
  create-build <image-uri>        Create a new build from Docker image
  get-build <build-id>            Get build status and details
  list-builds                     List builds
  cancel-build <build-id>         Cancel a running build
  delete-build <build-id>         Delete a build and its artifacts
  stream-logs <build-id>          Stream build logs in real-time
  get-stats                       Get build statistics

Environment Variables:
  UNKEY_BUILDERD_SERVER_ADDRESS   Server address (default: https://localhost:8082)
  UNKEY_BUILDERD_TLS_MODE         TLS mode (default: spiffe)
  UNKEY_BUILDERD_SPIFFE_SOCKET    SPIFFE socket path (default: /var/lib/spire/agent/agent.sock)

Examples:
  # Create build from Docker image with SPIFFE authentication
  %s create-build ubuntu:latest

  # Get build status
  %s get-build build-12345

  # List builds
  %s list-builds

  # Stream build logs
  %s stream-logs build-12345


  # Get build statistics
  %s get-stats

  # Get response with JSON output
  %s get-build build-12345 -json

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func handleCreateBuild(ctx context.Context, builderClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Image URI is required for create-build command")
	}
	imageURI := flag.Arg(1)

	// Create a basic Docker image build configuration
	config := &builderv1.BuildConfig{
		Source: &builderv1.BuildSource{
			SourceType: &builderv1.BuildSource_DockerImage{
				DockerImage: &builderv1.DockerImageSource{
					ImageUri: imageURI,
				},
			},
		},
		Target: &builderv1.BuildTarget{
			TargetType: &builderv1.BuildTarget_MicrovmRootfs{
				MicrovmRootfs: &builderv1.MicroVMRootfs{},
			},
		},
		Strategy: &builderv1.BuildStrategy{
			StrategyType: &builderv1.BuildStrategy_DockerExtract{
				DockerExtract: &builderv1.DockerExtractStrategy{
					FlattenFilesystem: true,
				},
			},
		},
		BuildName: fmt.Sprintf("cli-build-%d", time.Now().Unix()),
	}

	req := &client.CreateBuildRequest{
		Config: config,
	}

	resp, err := builderClient.CreateBuild(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create build: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Build created:\n")
		fmt.Printf("  Build ID: %s\n", resp.BuildID)
		fmt.Printf("  State: %s\n", resp.State.String())
		fmt.Printf("  Created at: %s\n", resp.CreatedAt.AsTime().Format(time.RFC3339))
		if resp.RootfsPath != "" {
			fmt.Printf("  Rootfs path: %s\n", resp.RootfsPath)
		}
	}
}

func handleGetBuild(ctx context.Context, builderClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Build ID is required for get-build command")
	}
	buildID := flag.Arg(1)

	req := &client.GetBuildRequest{
		BuildID: buildID,
	}

	resp, err := builderClient.GetBuild(ctx, req)
	if err != nil {
		log.Fatalf("Failed to get build: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		build := resp.Build
		fmt.Printf("Build details:\n")
		fmt.Printf("  Build ID: %s\n", build.BuildId)
		fmt.Printf("  State: %s\n", build.State.String())
		fmt.Printf("  Created at: %s\n", build.CreatedAt.AsTime().Format(time.RFC3339))
		if build.CompletedAt != nil {
			fmt.Printf("  Completed at: %s\n", build.CompletedAt.AsTime().Format(time.RFC3339))
		}
		if build.RootfsPath != "" {
			fmt.Printf("  Rootfs path: %s\n", build.RootfsPath)
		}
	}
}

func handleListBuilds(ctx context.Context, builderClient *client.Client, jsonOutput bool) {
	req := &client.ListBuildsRequest{
		PageSize: 50,
	}

	resp, err := builderClient.ListBuilds(ctx, req)
	if err != nil {
		log.Fatalf("Failed to list builds: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Builds (total: %d):\n", resp.TotalCount)
		for _, build := range resp.Builds {
			fmt.Printf("  %s: %s (created: %s)\n",
				build.BuildId,
				build.State.String(),
				build.CreatedAt.AsTime().Format(time.RFC3339))
		}
	}
}

func handleCancelBuild(ctx context.Context, builderClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Build ID is required for cancel-build command")
	}
	buildID := flag.Arg(1)

	req := &client.CancelBuildRequest{
		BuildID: buildID,
	}

	resp, err := builderClient.CancelBuild(ctx, req)
	if err != nil {
		log.Fatalf("Failed to cancel build: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Build cancellation:\n")
		fmt.Printf("  Build ID: %s\n", buildID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  State: %s\n", resp.State.String())
	}
}

func handleDeleteBuild(ctx context.Context, builderClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Build ID is required for delete-build command")
	}
	buildID := flag.Arg(1)

	// Check for force flag from additional args
	force := false
	if flag.NArg() > 2 && flag.Arg(2) == "--force" {
		force = true
	}

	req := &client.DeleteBuildRequest{
		BuildID: buildID,
		Force:   force,
	}

	resp, err := builderClient.DeleteBuild(ctx, req)
	if err != nil {
		log.Fatalf("Failed to delete build: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Build deletion:\n")
		fmt.Printf("  Build ID: %s\n", buildID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Force: %v\n", force)
	}
}

func handleStreamLogs(ctx context.Context, builderClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Build ID is required for stream-logs command")
	}
	buildID := flag.Arg(1)

	req := &client.StreamBuildLogsRequest{
		BuildID: buildID,
		Follow:  true,
	}

	stream, err := builderClient.StreamBuildLogs(ctx, req)
	if err != nil {
		log.Fatalf("Failed to stream build logs: %v", err)
	}

	fmt.Printf("Streaming logs for build %s (press Ctrl+C to stop):\n", buildID)
	fmt.Println("---")

	for stream.Receive() {
		msg := stream.Msg()
		timestamp := msg.Timestamp.AsTime().Format("15:04:05")

		if jsonOutput {
			outputJSON(msg)
		} else {
			fmt.Printf("[%s] %s: %s\n", timestamp, msg.Component, msg.Message)
		}
	}

	if err := stream.Err(); err != nil {
		log.Fatalf("Stream error: %v", err)
	}
}

func handleGetStats(ctx context.Context, builderClient *client.Client, jsonOutput bool) {
	// Default to last 24 hours
	endTime := time.Now()
	startTime := endTime.Add(-24 * time.Hour)

	// Allow custom time range from CLI args
	if flag.NArg() > 1 {
		if hours, err := strconv.Atoi(flag.Arg(1)); err == nil {
			startTime = endTime.Add(-time.Duration(hours) * time.Hour)
		}
	}

	req := &client.GetBuildStatsRequest{
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	}

	resp, err := builderClient.GetBuildStats(ctx, req)
	if err != nil {
		log.Fatalf("Failed to get build stats: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		stats := resp.Stats
		duration := endTime.Sub(startTime)
		fmt.Printf("Build statistics (last %v):\n", duration)
		fmt.Printf("  Total builds: %d\n", stats.TotalBuilds)
		fmt.Printf("  Successful builds: %d\n", stats.SuccessfulBuilds)
		fmt.Printf("  Failed builds: %d\n", stats.FailedBuilds)
		fmt.Printf("  Average build time: %d ms\n", stats.AvgBuildTimeMs)
		fmt.Printf("  Total storage: %d bytes\n", stats.TotalStorageBytes)
		fmt.Printf("  Total compute minutes: %d\n", stats.TotalComputeMinutes)
		if len(stats.RecentBuilds) > 0 {
			fmt.Printf("  Recent builds: %d\n", len(stats.RecentBuilds))
		}
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
