package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/unkeyed/unkey/go/deploy/metald/client"
	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
)

func main() {
	var (
		serverAddr    = flag.String("server", getEnvOrDefault("UNKEY_METALD_SERVER_ADDRESS", "https://localhost:8080"), "metald server address")
		userID        = flag.String("user", getEnvOrDefault("UNKEY_METALD_USER_ID", "cli-user"), "user ID for authentication")
		tenantID      = flag.String("tenant", getEnvOrDefault("UNKEY_METALD_TENANT_ID", "cli-tenant"), "tenant ID for data scoping")
		projectID     = flag.String("project-id", getEnvOrDefault("UNKEY_METALD_PROJECT_ID", "metald-cli-test"), "project ID for data scoping")
		environmentID = flag.String("environment-id", getEnvOrDefault("UNKEY_METALD_ENVIRONMENT_ID", "development"), "environment ID for data scoping")
		tlsMode       = flag.String("tls-mode", getEnvOrDefault("UNKEY_METALD_TLS_MODE", "spiffe"), "TLS mode: disabled, file, or spiffe")
		spiffeSocket  = flag.String("spiffe-socket", getEnvOrDefault("UNKEY_METALD_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"), "SPIFFE agent socket path")
		tlsCert       = flag.String("tls-cert", "", "TLS certificate file (for file mode)")
		tlsKey        = flag.String("tls-key", "", "TLS key file (for file mode)")
		tlsCA         = flag.String("tls-ca", "", "TLS CA file (for file mode)")
		timeout       = flag.Duration("timeout", 30*time.Second, "request timeout")
		jsonOutput    = flag.Bool("json", false, "output results as JSON")

		// VM configuration options
		configFile  = flag.String("config", "", "path to VM configuration file (JSON)")
		template    = flag.String("template", "standard", "VM template: minimal, standard, high-cpu, high-memory, development")
		cpuCount    = flag.Uint("cpu", 0, "number of vCPUs (overrides template)")
		memoryMB    = flag.Uint64("memory", 0, "memory in MB (overrides template)")
		dockerImage = flag.String("docker-image", "", "Docker image to run in VM")
		forceBuild  = flag.Bool("force-build", false, "force rebuild assets even if cached versions exist")
	)
	flag.Parse()

	if flag.NArg() == 0 {
		printUsage()
		os.Exit(1)
	}

	ctx := context.Background()

	// Create metald client
	config := client.Config{
		ServerAddress:    *serverAddr,
		UserID:           *userID,
		TenantID:         *tenantID,
		ProjectID:        *projectID,
		EnvironmentID:    *environmentID,
		TLSMode:          *tlsMode,
		SPIFFESocketPath: *spiffeSocket,
		TLSCertFile:      *tlsCert,
		TLSKeyFile:       *tlsKey,
		TLSCAFile:        *tlsCA,
		Timeout:          *timeout,
	}

	metaldClient, err := client.New(ctx, config)
	if err != nil {
		log.Fatalf("Failed to create metald client: %v", err)
	}
	defer metaldClient.Close()

	// VM configuration options for create commands
	vmConfigOptions := VMConfigOptions{
		ConfigFile:  *configFile,
		Template:    *template,
		CPUCount:    uint32(*cpuCount),
		MemoryMB:    *memoryMB,
		DockerImage: *dockerImage,
		ForceBuild:  *forceBuild,
	}

	// Execute command
	command := flag.Arg(0)
	switch command {
	case "create":
		handleCreate(ctx, metaldClient, vmConfigOptions, *jsonOutput)
	case "boot":
		handleBoot(ctx, metaldClient, *jsonOutput)
	case "shutdown":
		handleShutdown(ctx, metaldClient, *jsonOutput)
	case "delete":
		handleDelete(ctx, metaldClient, *jsonOutput)
	case "info":
		handleInfo(ctx, metaldClient, *jsonOutput)
	case "list":
		handleList(ctx, metaldClient, *jsonOutput)
	case "pause":
		handlePause(ctx, metaldClient, *jsonOutput)
	case "resume":
		handleResume(ctx, metaldClient, *jsonOutput)
	case "reboot":
		handleReboot(ctx, metaldClient, *jsonOutput)
	case "create-and-boot":
		handleCreateAndBoot(ctx, metaldClient, vmConfigOptions, *jsonOutput)
	case "config-gen":
		handleConfigGen(vmConfigOptions, *jsonOutput)
	case "config-validate":
		handleConfigValidate(*configFile, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`metald-cli - CLI tool for metald VM operations

Usage: %s [flags] <command> [args...]

Commands:
  create [vm-id]              Create a new VM (VM ID optional)
  boot <vm-id>                Boot a created VM
  shutdown <vm-id> [force]    Shutdown a running VM
  delete <vm-id> [force]      Delete a VM
  info <vm-id>                Get detailed VM information
  list                        List all VMs for customer
  pause <vm-id>               Pause a running VM
  resume <vm-id>              Resume a paused VM
  reboot <vm-id> [force]      Reboot a running VM
  create-and-boot [vm-id]     Create and immediately boot a VM
  config-gen                  Generate a VM configuration file
  config-validate <file>      Validate a VM configuration file

Environment Variables:
  UNKEY_METALD_SERVER_ADDRESS  Server address (default: https://localhost:8080)
  UNKEY_METALD_USER_ID         User ID for authentication (default: cli-user)
  UNKEY_METALD_TENANT_ID       Tenant ID for data scoping (default: cli-tenant)
  UNKEY_METALD_TLS_MODE        TLS mode (default: spiffe)
  UNKEY_METALD_SPIFFE_SOCKET   SPIFFE socket path (default: /var/lib/spire/agent/agent.sock)

VM Configuration Options:
  -config <file>              Use VM configuration from JSON file
  -template <name>            Use built-in template (minimal, standard, high-cpu, high-memory, development)
  -cpu <count>                Override CPU count from template
  -memory <mb>                Override memory in MB from template
  -docker-image <image>       Configure VM for Docker image
  -force-build                Force rebuild assets even if cached versions exist

Examples:
  # Create and boot a VM with SPIFFE authentication
  %s -user=prod-user-123 -tenant=prod-tenant-456 create-and-boot

  # Create VM from configuration file
  %s -config=my-vm.json create

  # Create VM with template and overrides
  %s -template=high-cpu -memory=4096 create-and-boot

  # Create VM for Docker image
  %s -docker-image=nginx:alpine create-and-boot

  # Create VM for Docker image with force build (bypass cache)
  %s -docker-image=nginx:alpine -force-build create-and-boot

  # Generate configuration file
  %s -template=development config-gen > dev-vm.json

  # List VMs with disabled TLS (development)
  %s -tls-mode=disabled -server=http://localhost:8080 list

  # Get VM info with JSON output
  %s info vm-12345 -json

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

// VMConfigOptions holds options for VM configuration
type VMConfigOptions struct {
	ConfigFile  string
	Template    string
	CPUCount    uint32
	MemoryMB    uint64
	DockerImage string
	ForceBuild  bool
}

// createVMConfig creates a VM configuration from the provided options
func createVMConfig(options VMConfigOptions) (*vmprovisionerv1.VmConfig, error) {
	// If config file is specified, load from file
	if options.ConfigFile != "" {
		configFile, err := client.LoadVMConfigFromFile(options.ConfigFile)
		if err != nil {
			return nil, fmt.Errorf("failed to load config file: %w", err)
		}
		return configFile.ToVMConfig()
	}

	builder := client.NewVMConfigBuilder()

	// Apply Docker image configuration if specified
	if options.DockerImage != "" {
		builder.ForDockerImage(options.DockerImage)
		if options.ForceBuild {
			builder.ForceBuild(true)
		}
	}

	// Apply overrides
	if options.CPUCount > 0 {
		// Keep max CPU at 2x current CPU or original max, whichever is higher
		maxCPU := options.CPUCount * 2
		originalMaxCPU := uint32(builder.Build().Cpu.MaxVcpuCount)
		if originalMaxCPU > maxCPU {
			maxCPU = originalMaxCPU
		}
		builder.WithCPU(options.CPUCount, maxCPU)
	}

	if options.MemoryMB > 0 {
		// Keep max memory at 2x current memory or original max, whichever is higher
		maxMemoryMB := options.MemoryMB * 2
		originalMaxMB := uint64(builder.Build().Memory.MaxSizeBytes / (1024 * 1024))
		if originalMaxMB > maxMemoryMB {
			maxMemoryMB = originalMaxMB
		}
		builder.WithMemoryMB(options.MemoryMB, maxMemoryMB, builder.Build().Memory.HotplugEnabled)
	}

	// Add CLI metadata
	builder.AddMetadata("created_by", "metald-cli")
	builder.AddMetadata("creation_time", time.Now().Format(time.RFC3339))

	// Validate configuration
	config := builder.Build()
	if err := client.ValidateVMConfig(config); err != nil {
		return nil, fmt.Errorf("VM configuration validation failed: %w", err)
	}

	fmt.Printf("DEBUG: Final VM config:\n")
	outputJSON(config)

	return config, nil
}

func handleCreate(ctx context.Context, metaldClient *client.Client, options VMConfigOptions, jsonOutput bool) {
	vmID := ""
	if flag.NArg() > 1 {
		vmID = flag.Arg(1)
	}

	// Create VM configuration from options
	config, err := createVMConfig(options)
	if err != nil {
		log.Fatalf("Failed to create VM configuration: %v", err)
	}

	// DEBUG: Log config right before sending
	fmt.Printf("DEBUG CLIENT: Sending VM config metadata:\n")
	outputJSON(config)
	for i, storage := range config.Storage {
		fmt.Printf("DEBUG CLIENT: Storage[%d]: id=%s, path=%s, isRoot=%v, options=%v\n",
			i, storage.Id, storage.Path, storage.IsRootDevice, storage.Options)
	}

	req := &client.CreateVMRequest{
		VMID:   vmID,
		Config: config,
	}

	resp, err := metaldClient.CreateVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	// Fetch VM info to get IP address
	var ipAddress string
	vmInfo, err := metaldClient.GetVMInfo(ctx, resp.VMID)
	if err != nil {
		// Don't fail on network info error, just log it
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch IP address: %v\n", err)
	} else if vmInfo.NetworkInfo != nil {
		ipAddress = vmInfo.NetworkInfo.IpAddress
	}

	if jsonOutput {
		result := map[string]any{
			"vm_id": resp.VMID,
			"state": resp.State.String(),
		}
		if ipAddress != "" {
			result["ip_address"] = ipAddress
		}
		outputJSON(result)
	} else {
		fmt.Printf("VM created successfully:\n")
		fmt.Printf("  VM ID: %s\n", resp.VMID)
		fmt.Printf("  State: %s\n", resp.State.String())
		if ipAddress != "" {
			fmt.Printf("  IP Address: %s\n", ipAddress)
		}
	}
}

func handleBoot(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for boot command")
	}
	vmID := flag.Arg(1)

	resp, err := metaldClient.BootVM(ctx, vmID)
	if err != nil {
		log.Fatalf("Failed to boot VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"vm_id":   vmID,
			"success": resp.Success,
			"state":   resp.State.String(),
		})
	} else {
		fmt.Printf("VM boot operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  State: %s\n", resp.State.String())
	}
}

func handleShutdown(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for shutdown command")
	}
	vmID := flag.Arg(1)

	force := false
	if flag.NArg() > 2 && flag.Arg(2) == "force" {
		force = true
	}

	req := &client.ShutdownVMRequest{
		VMID:           vmID,
		Force:          force,
		TimeoutSeconds: 30,
	}

	resp, err := metaldClient.ShutdownVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to shutdown VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"vm_id":   vmID,
			"success": resp.Success,
			"state":   resp.State.String(),
			"force":   force,
		})
	} else {
		fmt.Printf("VM shutdown operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  State: %s\n", resp.State.String())
		fmt.Printf("  Force: %v\n", force)
	}
}

func handleDelete(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for delete command")
	}
	vmID := flag.Arg(1)

	force := false
	if flag.NArg() > 2 && flag.Arg(2) == "force" {
		force = true
	}

	req := &client.DeleteVMRequest{
		VMID:  vmID,
		Force: force,
	}

	resp, err := metaldClient.DeleteVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to delete VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"vm_id":   vmID,
			"success": resp.Success,
			"force":   force,
		})
	} else {
		fmt.Printf("VM delete operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  Force: %v\n", force)
	}
}

func handleInfo(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for info command")
	}
	vmID := flag.Arg(1)

	vmInfo, err := metaldClient.GetVMInfo(ctx, vmID)
	if err != nil {
		log.Fatalf("Failed to get VM info: %v", err)
	}

	if jsonOutput {
		outputJSON(vmInfo)
	} else {
		fmt.Printf("VM Information:\n")
		fmt.Printf("  VM ID: %s\n", vmInfo.VMID)
		fmt.Printf("  State: %s\n", vmInfo.State.String())

		if vmInfo.Config != nil {
			fmt.Printf("  Configuration:\n")
			fmt.Printf("    CPUs: %d (max: %d)\n", vmInfo.Config.Cpu.VcpuCount, vmInfo.Config.Cpu.MaxVcpuCount)
			fmt.Printf("    Memory: %d MB\n", vmInfo.Config.Memory.SizeBytes/(1024*1024))
			fmt.Printf("    Storage devices: %d\n", len(vmInfo.Config.Storage))
			fmt.Printf("    Network interfaces: %d\n", len(vmInfo.Config.Network))
		}

		if vmInfo.Metrics != nil {
			fmt.Printf("  Metrics:\n")
			fmt.Printf("    CPU usage: %.2f%%\n", vmInfo.Metrics.CpuUsagePercent)
			fmt.Printf("    Memory usage: %d MB\n", vmInfo.Metrics.MemoryUsageBytes/(1024*1024))
			fmt.Printf("    Uptime: %d seconds\n", vmInfo.Metrics.UptimeSeconds)
		}

		if vmInfo.NetworkInfo != nil {
			fmt.Printf("  Network:\n")
			fmt.Printf("    IP: %s\n", vmInfo.NetworkInfo.IpAddress)
			fmt.Printf("    MAC: %s\n", vmInfo.NetworkInfo.MacAddress)
			fmt.Printf("    TAP: %s\n", vmInfo.NetworkInfo.TapDevice)

			if len(vmInfo.NetworkInfo.PortMappings) > 0 {
				fmt.Printf("    Port Mappings:\n")
				for _, mapping := range vmInfo.NetworkInfo.PortMappings {
					fmt.Printf("      %d:%d/%s\n", mapping.HostPort, mapping.ContainerPort, mapping.Protocol)
				}
			}
		}
	}
}

func handleList(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	req := &client.ListVMsRequest{
		PageSize: 50,
	}

	resp, err := metaldClient.ListVMs(ctx, req)
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("VMs for tenant %s (total: %d):\n", metaldClient.GetTenantID(), resp.TotalCount)
		for _, vm := range resp.VMs {
			fmt.Printf("  - %s: %s (CPUs: %d, Memory: %d MB)\n",
				vm.VmId,
				vm.State.String(),
				vm.VcpuCount,
				vm.MemorySizeBytes/(1024*1024),
			)
		}
	}
}

func handlePause(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for pause command")
	}
	vmID := flag.Arg(1)

	resp, err := metaldClient.PauseVM(ctx, vmID)
	if err != nil {
		log.Fatalf("Failed to pause VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"vm_id":   vmID,
			"success": resp.Success,
			"state":   resp.State.String(),
		})
	} else {
		fmt.Printf("VM pause operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  State: %s\n", resp.State.String())
	}
}

func handleResume(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for resume command")
	}
	vmID := flag.Arg(1)

	resp, err := metaldClient.ResumeVM(ctx, vmID)
	if err != nil {
		log.Fatalf("Failed to resume VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]any{
			"vm_id":   vmID,
			"success": resp.Success,
			"state":   resp.State.String(),
		})
	} else {
		fmt.Printf("VM resume operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  State: %s\n", resp.State.String())
	}
}

func handleReboot(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for reboot command")
	}
	vmID := flag.Arg(1)

	force := false
	if flag.NArg() > 2 && flag.Arg(2) == "force" {
		force = true
	}

	req := &client.RebootVMRequest{
		VMID:  vmID,
		Force: force,
	}

	resp, err := metaldClient.RebootVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to reboot VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]any{
			"vm_id":   vmID,
			"success": resp.Success,
			"state":   resp.State.String(),
			"force":   force,
		})
	} else {
		fmt.Printf("VM reboot operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  Success: %v\n", resp.Success)
		fmt.Printf("  State: %s\n", resp.State.String())
		fmt.Printf("  Force: %v\n", force)
	}
}

func handleCreateAndBoot(ctx context.Context, metaldClient *client.Client, options VMConfigOptions, jsonOutput bool) {
	vmID := ""
	if flag.NArg() > 1 {
		vmID = flag.Arg(1)
	}

	// Create VM configuration from options
	config, err := createVMConfig(options)
	if err != nil {
		log.Fatalf("Failed to create VM configuration: %v", err)
	}

	createReq := &client.CreateVMRequest{
		VMID:   vmID,
		Config: config,
	}
	log.Printf("createReq: %+v/n", createReq)
	createResp, err := metaldClient.CreateVM(ctx, createReq)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	// Wait a moment for VM to be fully created
	time.Sleep(2 * time.Second)

	// Boot VM
	bootResp, err := metaldClient.BootVM(ctx, createResp.VMID)
	if err != nil {
		log.Fatalf("Failed to boot VM: %v", err)
	}

	// Wait a moment for VM to boot and get IP address
	time.Sleep(3 * time.Second)

	// Fetch VM info to get IP address
	var ipAddress string
	vmInfo, err := metaldClient.GetVMInfo(ctx, createResp.VMID)
	if err != nil {
		// Don't fail on network info error, just log it
		fmt.Fprintf(os.Stderr, "Warning: Could not fetch IP address: %v\n", err)
	} else if vmInfo.NetworkInfo != nil {
		ipAddress = vmInfo.NetworkInfo.IpAddress
	}

	if jsonOutput {
		result := map[string]any{
			"vm_id":        createResp.VMID,
			"create_state": createResp.State.String(),
			"boot_success": bootResp.Success,
			"boot_state":   bootResp.State.String(),
		}
		if ipAddress != "" {
			result["ip_address"] = ipAddress
		}
		outputJSON(result)
	} else {
		fmt.Printf("VM created and booted successfully:\n")
		fmt.Printf("  VM ID: %s\n", createResp.VMID)
		fmt.Printf("  Create State: %s\n", createResp.State.String())
		fmt.Printf("  Boot Success: %v\n", bootResp.Success)
		fmt.Printf("  Boot State: %s\n", bootResp.State.String())
		if ipAddress != "" {
			fmt.Printf("  IP Address: %s\n", ipAddress)
		}
	}
}

func createDefaultVMConfig() *vmprovisionerv1.VmConfig {
	return &vmprovisionerv1.VmConfig{
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
			"purpose":     "cli-created",
			"environment": "development",
			"tool":        "metald-cli",
		},
	}
}

func outputJSON(data interface{}) {
	encoder := json.NewEncoder(os.Stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(data); err != nil {
		log.Fatalf("Failed to encode JSON: %v", err)
	}
}

func handleConfigGen(options VMConfigOptions, jsonOutput bool) {
	// Create VM configuration
	config, err := createVMConfig(options)
	if err != nil {
		log.Fatalf("Failed to create VM configuration: %v", err)
	}

	// Convert to config file format
	templateName := options.Template
	if templateName == "" {
		templateName = "standard"
	}
	configFile := client.FromVMConfig(config, templateName, fmt.Sprintf("Generated %s VM configuration", templateName))
	configFile.Template = templateName

	// Output as JSON
	data, err := json.MarshalIndent(configFile, "", "  ")
	if err != nil {
		log.Fatalf("Failed to marshal configuration: %v", err)
	}

	fmt.Printf("%s\n", data)
}

func handleConfigValidate(configFile string, jsonOutput bool) {
	if configFile == "" {
		log.Fatal("Configuration file path is required")
	}

	// Load configuration file
	config, err := client.LoadVMConfigFromFile(configFile)
	if err != nil {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			})
		} else {
			fmt.Printf("Configuration validation failed: %v\n", err)
		}
		os.Exit(1)
	}

	// Convert to VM config and validate
	vmConfig, err := config.ToVMConfig()
	if err != nil {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			})
		} else {
			fmt.Printf("Configuration conversion failed: %v\n", err)
		}
		os.Exit(1)
	}

	// Validate the VM configuration
	if err := client.ValidateVMConfig(vmConfig); err != nil {
		if jsonOutput {
			outputJSON(map[string]interface{}{
				"valid": false,
				"error": err.Error(),
			})
		} else {
			fmt.Printf("Configuration validation failed: %v\n", err)
		}
		os.Exit(1)
	}

	// Configuration is valid
	if jsonOutput {
		outputJSON(map[string]interface{}{
			"valid":       true,
			"name":        config.Name,
			"description": config.Description,
			"template":    config.Template,
		})
	} else {
		fmt.Printf("Configuration is valid:\n")
		fmt.Printf("  Name: %s\n", config.Name)
		fmt.Printf("  Description: %s\n", config.Description)
		if config.Template != "" {
			fmt.Printf("  Template: %s\n", config.Template)
		}
		fmt.Printf("  CPU: %d vCPUs (max: %d)\n", config.CPU.VCPUCount, config.CPU.MaxVCPUCount)
		fmt.Printf("  Memory: %d MB (max: %d MB)\n", config.Memory.SizeMB, config.Memory.MaxSizeMB)
		fmt.Printf("  Storage devices: %d\n", len(config.Storage))
		fmt.Printf("  Network interfaces: %d\n", len(config.Network))
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
