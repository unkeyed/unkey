package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/unkeyed/unkey/go/apps/assetmanagerd/client"
	assetv1 "github.com/unkeyed/unkey/go/gen/proto/assetmanagerd/v1"
)

// AIDEV-NOTE: CLI tool demonstrating assetmanagerd client usage with SPIFFE integration
// This provides a command-line interface for asset management operations

func main() {
	var (
		serverAddr   = flag.String("server", getEnvOrDefault("UNKEY_ASSETMANAGERD_SERVER_ADDRESS", "https://localhost:8083"), "assetmanagerd server address")
		userID       = flag.String("user", getEnvOrDefault("UNKEY_ASSETMANAGERD_USER_ID", "cli-user"), "user ID for authentication")
		tlsMode      = flag.String("tls-mode", getEnvOrDefault("UNKEY_ASSETMANAGERD_TLS_MODE", "spiffe"), "TLS mode: disabled, file, or spiffe")
		spiffeSocket = flag.String("spiffe-socket", getEnvOrDefault("UNKEY_ASSETMANAGERD_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"), "SPIFFE agent socket path")
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

	// Create assetmanagerd client
	config := client.Config{
		ServerAddress:    *serverAddr,
		UserID:           *userID,
		TLSMode:          *tlsMode,
		SPIFFESocketPath: *spiffeSocket,
		TLSCertFile:      *tlsCert,
		TLSKeyFile:       *tlsKey,
		TLSCAFile:        *tlsCA,
		Timeout:          *timeout,
	}

	assetClient, err := client.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create assetmanagerd client: %v", err)
	}
	defer assetClient.Close()

	// Execute command
	command := flag.Arg(0)
	switch command {
	case "list":
		handleList(ctx, assetClient, *jsonOutput)
	case "get":
		handleGet(ctx, assetClient, *jsonOutput)
	case "register":
		handleRegister(ctx, assetClient, *jsonOutput)
	case "query":
		handleQuery(ctx, assetClient, *jsonOutput)
	case "prepare":
		handlePrepare(ctx, assetClient, *jsonOutput)
	case "acquire":
		handleAcquire(ctx, assetClient, *jsonOutput)
	case "release":
		handleRelease(ctx, assetClient, *jsonOutput)
	case "delete":
		handleDelete(ctx, assetClient, *jsonOutput)
	case "gc":
		handleGarbageCollect(ctx, assetClient, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`assetmanagerd-cli - CLI tool for assetmanagerd operations

Usage: %s [flags] <command> [args...]

Commands:
  list                        List all assets
  get <asset-id>              Get detailed asset information
  register <asset-file>       Register a new asset from JSON file
  query <requirements>        Query assets with auto-build
  prepare <asset-ids...>      Prepare assets for deployment
  acquire <asset-id>          Acquire asset reference
  release <asset-id>          Release asset reference
  delete <asset-id>           Delete an asset
  gc                          Trigger garbage collection

Environment Variables:
  UNKEY_ASSETMANAGERD_SERVER_ADDRESS  Server address (default: https://localhost:8083)
  UNKEY_ASSETMANAGERD_USER_ID         User ID for authentication (default: cli-user)
  UNKEY_ASSETMANAGERD_TLS_MODE        TLS mode (default: spiffe)
  UNKEY_ASSETMANAGERD_SPIFFE_SOCKET   SPIFFE socket path (default: /var/lib/spire/agent/agent.sock)

Examples:
  # List all assets with SPIFFE authentication
  %s -user=prod-user-123 list

  # Get detailed asset information
  %s get asset-12345

  # Query assets for a specific Docker image
  %s query -docker-image=nginx:alpine

  # Prepare assets for deployment
  %s prepare asset-123 asset-456

  # List assets with disabled TLS (development)
  %s -tls-mode=disabled -server=http://localhost:8083 list

  # Get asset info with JSON output
  %s get asset-12345 -json

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func handleList(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	req := &client.ListAssetsRequest{
		PageSize: 50,
	}

	resp, err := assetClient.ListAssets(ctx, req)
	if err != nil {
		log.Fatalf("Failed to list assets: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		for _, asset := range resp.Assets {
			fmt.Printf("  - %s: %s (%s, %d bytes)\n",
				asset.Id,
				asset.Name,
				asset.Type.String(),
				asset.SizeBytes,
			)
		}
	}
}

func handleGet(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Asset ID is required for get command")
	}
	assetID := flag.Arg(1)

	resp, err := assetClient.GetAsset(ctx, assetID)
	if err != nil {
		log.Fatalf("Failed to get asset: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		asset := resp.Asset
		fmt.Printf("Asset Information:\n")
		fmt.Printf("  ID: %s\n", asset.Id)
		fmt.Printf("  Name: %s\n", asset.Name)
		fmt.Printf("  Type: %s\n", asset.Type.String())
		fmt.Printf("  Backend: %s\n", asset.Backend.String())
		fmt.Printf("  Location: %s\n", asset.Location)
		fmt.Printf("  Size: %d bytes\n", asset.SizeBytes)
		fmt.Printf("  Created by: %s\n", asset.CreatedBy)
		fmt.Printf("  Created at: %d\n", asset.CreatedAt)

		if len(asset.Labels) > 0 {
			fmt.Printf("  Labels:\n")
			for k, v := range asset.Labels {
				fmt.Printf("    %s: %s\n", k, v)
			}
		}
	}
}

func handleRegister(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Asset file path is required for register command")
	}
	assetFile := flag.Arg(1)

	// Read asset from JSON file
	data, err := os.ReadFile(assetFile)
	if err != nil {
		log.Fatalf("Failed to read asset file: %v", err)
	}

	var asset assetv1.Asset
	if err := json.Unmarshal(data, &asset); err != nil {
		log.Fatalf("Failed to parse asset JSON: %v", err)
	}

	req := &client.RegisterAssetRequest{
		Name:      asset.Name,
		Type:      asset.Type,
		Backend:   asset.Backend,
		Location:  asset.Location,
		SizeBytes: asset.SizeBytes,
		Checksum:  asset.Checksum,
		Labels:    asset.Labels,
		CreatedBy: asset.CreatedBy,
	}

	resp, err := assetClient.RegisterAsset(ctx, req)
	if err != nil {
		log.Fatalf("Failed to register asset: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Asset registered successfully:\n")
		fmt.Printf("  Asset ID: %s\n", resp.Asset.Id)
		fmt.Printf("  Asset Name: %s\n", resp.Asset.Name)
	}
}

func handleQuery(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	// Simple query example - in real usage, this would parse requirements from CLI args
	req := &client.QueryAssetsRequest{
		Type: assetv1.AssetType_ASSET_TYPE_ROOTFS,
		Labels: map[string]string{
			"docker_image": "nginx:alpine",
		},
	}

	resp, err := assetClient.QueryAssets(ctx, req)
	if err != nil {
		log.Fatalf("Failed to query assets: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Query Results:\n")
		fmt.Printf("  Found assets: %d\n", len(resp.Assets))
		for _, asset := range resp.Assets {
			fmt.Printf("    - %s (%s)\n", asset.Id, asset.Name)
		}
	}
}

func handlePrepare(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("At least one asset ID is required for prepare command")
	}

	assetIDs := flag.Args()[1:]
	req := &client.PrepareAssetsRequest{
		AssetIds: assetIDs,
		JailerId: "default",
		CacheDir: "/tmp/asset-cache",
	}

	resp, err := assetClient.PrepareAssets(ctx, req)
	if err != nil {
		log.Fatalf("Failed to prepare assets: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Asset preparation:\n")
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Prepared paths: %d\n", len(resp.PreparedPaths))
		for i, path := range resp.PreparedPaths {
			fmt.Printf("    - %s: %s\n", assetIDs[i], path)
		}
	}
}

func handleAcquire(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Asset ID is required for acquire command")
	}
	assetID := flag.Arg(1)

	resp, err := assetClient.AcquireAsset(ctx, assetID)
	if err != nil {
		log.Fatalf("Failed to acquire asset: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Asset acquisition:\n")
		fmt.Printf("  Asset ID: %s\n", assetID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Reference count: %d\n", resp.ReferenceCount)
	}
}

func handleRelease(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Asset ID is required for release command")
	}
	assetID := flag.Arg(1)

	resp, err := assetClient.ReleaseAsset(ctx, assetID)
	if err != nil {
		log.Fatalf("Failed to release asset: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Asset release:\n")
		fmt.Printf("  Asset ID: %s\n", assetID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Reference count: %d\n", resp.ReferenceCount)
	}
}

func handleDelete(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("Asset ID is required for delete command")
	}
	assetID := flag.Arg(1)

	resp, err := assetClient.DeleteAsset(ctx, assetID)
	if err != nil {
		log.Fatalf("Failed to delete asset: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Asset deletion:\n")
		fmt.Printf("  Asset ID: %s\n", assetID)
		fmt.Printf("  Success: %v\n", resp.Success)
	}
}

func handleGarbageCollect(ctx context.Context, assetClient *client.Client, jsonOutput bool) {
	req := &client.GarbageCollectRequest{
		DryRun:      true, // Default to dry run for safety
		MaxAgeHours: 24,   // Clean up assets older than 24 hours
	}

	// Check for --force flag
	if flag.NArg() > 1 && flag.Arg(1) == "--force" {
		req.DryRun = false
		req.ForceCleanup = true
	}

	resp, err := assetClient.GarbageCollect(ctx, req)
	if err != nil {
		log.Fatalf("Failed to garbage collect: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		mode := "DRY RUN"
		if !req.DryRun {
			mode = "EXECUTED"
		}
		fmt.Printf("Garbage Collection (%s):\n", mode)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Assets removed: %d\n", len(resp.RemovedAssets))
		fmt.Printf("  Bytes freed: %d\n", resp.FreedBytes)
		if len(resp.RemovedAssets) > 0 {
			fmt.Printf("  Removed asset IDs:\n")
			for _, assetID := range resp.RemovedAssets {
				fmt.Printf("    - %s\n", assetID)
			}
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
