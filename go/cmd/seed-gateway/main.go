package seedgateway

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/cli"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/partition/db"
	"google.golang.org/protobuf/proto"
)

var SeedGatewayFlags = []cli.Flag{
	cli.String("hostname", "Hostname for the gateway (e.g., api.myapp.com)", cli.Required()),
	cli.String("deployment-id", "Deployment ID for the gateway", cli.Required()),
	cli.StringSlice("vm-ids", "Comma-separated list of VM IDs", cli.Required()),
	cli.StringSlice("vm-ips", "Comma-separated list of VM private IPs (must match vm-ids order)", cli.Required()),
	cli.StringSlice("vm-ports", "Comma-separated list of VM ports (must match vm-ids order)", cli.Default([]string{"8080"})),
	cli.String("region", "Region for the VMs", cli.Default("us-east-1")),
	cli.Bool("enabled", "Whether the gateway is enabled", cli.Default(true)),
	cli.String("dsn", "Database connection string", cli.EnvVar("DATABASE_DSN"), cli.Required()),
}

var Cmd = &cli.Command{
	Name:  "seed-gateway",
	Usage: "Create gateway configurations in the database for testing",
	Description: `Create a gateway configuration in the database with the specified hostname,
deployment ID, and VM IDs. This is useful for testing the gateway service
locally or setting up development environments.

EXAMPLES:
    # Basic usage
    unkey seed-gateway \
      --hostname="api.myapp.com" \
      --deployment-id="deploy-123" \
      --vm-ids="vm-001,vm-002,vm-003" \
      --vm-ips="192.168.1.10,192.168.1.11,192.168.1.12" \
      --vm-ports="8080,8080,8080" \
      --dsn="user:password@tcp(localhost:3306)/database_name"

    # Using environment variable for DSN and default ports
    export DATABASE_DSN="user:password@tcp(localhost:3306)/database_name"
    unkey seed-gateway \
      --hostname="api.myapp.com" \
      --deployment-id="deploy-123" \
      --vm-ids="vm-001,vm-002,vm-003" \
      --vm-ips="192.168.1.10,192.168.1.11,192.168.1.12"

    # Create disabled gateway with custom region
    unkey seed-gateway \
      --hostname="staging.myapp.com" \
      --deployment-id="deploy-staging" \
      --vm-ids="vm-staging-001" \
      --vm-ips="10.0.1.100" \
      --vm-ports="3000" \
      --region="us-west-2" \
      --enabled=false

    # Test the gateway afterward
    curl -H "Host: api.myapp.com" http://localhost:7070/`,
	Flags:  SeedGatewayFlags,
	Action: SeedGatewayAction,
}

func SeedGatewayAction(ctx context.Context, cmd *cli.Command) error {
	hostname := cmd.RequireString("hostname")
	deploymentID := cmd.RequireString("deployment-id")
	vmIDs := cmd.RequireStringSlice("vm-ids")
	vmIPs := cmd.RequireStringSlice("vm-ips")
	vmPorts := cmd.StringSlice("vm-ports")
	region := cmd.String("region")
	enabled := cmd.Bool("enabled")
	dsn := cmd.RequireString("dsn")

	// Validate that VM IDs and IPs have the same length
	if len(vmIDs) != len(vmIPs) {
		return fmt.Errorf("number of VM IDs (%d) must match number of VM IPs (%d)", len(vmIDs), len(vmIPs))
	}

	// Handle ports - if only one port is provided, use it for all VMs
	if len(vmPorts) == 1 && len(vmIDs) > 1 {
		// Repeat the single port for all VMs
		singlePort := vmPorts[0]
		vmPorts = make([]string, len(vmIDs))
		for i := range vmPorts {
			vmPorts[i] = singlePort
		}
	} else if len(vmPorts) != len(vmIDs) {
		return fmt.Errorf("number of VM ports (%d) must match number of VM IDs (%d), or provide a single port for all VMs", len(vmPorts), len(vmIDs))
	}

	logger := logging.New()

	// Connect to database
	database, err := db.New(db.Config{
		PrimaryDSN: dsn,
		Logger:     logger,
	})
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer database.Close()

	// Create VM entries in the database first
	fmt.Printf("Creating %d VM entries...\n", len(vmIDs))
	for i, vmID := range vmIDs {
		vmID = strings.TrimSpace(vmID)
		vmIP := strings.TrimSpace(vmIPs[i])
		vmPortStr := strings.TrimSpace(vmPorts[i])

		// Parse port to int32
		vmPort, err := strconv.ParseInt(vmPortStr, 10, 32)
		if err != nil {
			return fmt.Errorf("invalid port %q for VM %s: %w", vmPortStr, vmID, err)
		}

		// Create VM record in database
		vmParams := db.UpsertVMParams{
			ID:           vmID,
			DeploymentID: deploymentID,
			Region:       region,
			PrivateIp: sql.NullString{
				String: vmIP,
				Valid:  true,
			},
			Port: sql.NullInt32{
				Int32: int32(vmPort),
				Valid: true,
			},
			CpuMillicores: 1000, // Default 1 CPU
			MemoryMb:      512,  // Default 512MB
			Status:        db.VmsStatusRunning,
			HealthStatus:  db.VmsHealthStatusHealthy,
		}

		if err := db.Query.UpsertVM(ctx, database.RW(), vmParams); err != nil {
			return fmt.Errorf("failed to create VM %s: %w", vmID, err)
		}

		fmt.Printf("  Created VM: %s (%s:%d)\n", vmID, vmIP, vmPort)
	}

	// Create VM protobuf objects for gateway config
	vms := make([]*partitionv1.VM, len(vmIDs))
	for i, vmID := range vmIDs {
		vms[i] = &partitionv1.VM{
			Id:     strings.TrimSpace(vmID),
			Region: region,
		}
	}

	// Create gateway config protobuf
	gatewayConfig := &partitionv1.GatewayConfig{
		DeploymentId: deploymentID,
		IsEnabled:    enabled,
		Vms:          vms,
	}

	// Marshal protobuf to bytes
	configBytes, err := proto.Marshal(gatewayConfig)
	if err != nil {
		return fmt.Errorf("failed to marshal gateway config: %w", err)
	}

	// Insert gateway config into database
	params := db.UpsertGatewayParams{
		Hostname: hostname,
		Config:   configBytes,
	}

	if err := db.Query.UpsertGateway(ctx, database.RW(), params); err != nil {
		return fmt.Errorf("failed to upsert gateway config: %w", err)
	}
	fmt.Printf("Created gateway config for hostname: %s\n", hostname)

	// Verify by reading it back
	fmt.Println("\nVerifying gateway config...")
	gatewayRow, err := db.Query.FindGatewayByHostname(ctx, database.RO(), hostname)
	if err != nil {
		return fmt.Errorf("failed to read gateway config: %w", err)
	}

	// Unmarshal to verify
	var readConfig partitionv1.GatewayConfig
	if err := proto.Unmarshal(gatewayRow.Config, &readConfig); err != nil {
		return fmt.Errorf("failed to unmarshal gateway config: %w", err)
	}

	fmt.Printf("Successfully read gateway config:\n")
	fmt.Printf("  Deployment ID: %s\n", readConfig.DeploymentId)
	fmt.Printf("  Enabled: %t\n", readConfig.IsEnabled)
	fmt.Printf("  VMs: %d\n", len(readConfig.Vms))
	for i, vm := range readConfig.Vms {
		// Read VM details from database to show IP and port
		vmDetails, err := db.Query.FindVMById(ctx, database.RO(), vm.Id)
		if err != nil {
			fmt.Printf("    VM %d: %s (details unavailable: %v)\n", i+1, vm.Id, err)
		} else {
			var ipPort string
			if vmDetails.PrivateIp.Valid && vmDetails.Port.Valid {
				ipPort = fmt.Sprintf(" (%s:%d)", vmDetails.PrivateIp.String, vmDetails.Port.Int32)
			}
			fmt.Printf("    VM %d: %s%s [%s, %s]\n", i+1, vm.Id, ipPort, vmDetails.Status, vmDetails.HealthStatus)
		}
	}

	fmt.Println("\nTest data created successfully!")
	fmt.Printf("\nYou can now test the gateway with:\n")
	fmt.Printf("  curl -H 'Host: %s' http://localhost:7070/\n", hostname)
	fmt.Printf("\nOr if running the gateway on a different port:\n")
	fmt.Printf("  curl -H 'Host: %s' http://localhost:<gateway-port>/\n", hostname)

	return nil
}
