package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/unkeyed/unkey/go/apps/metald/client"
	metaldv1 "github.com/unkeyed/unkey/go/gen/proto/metald/v1"
)

var (
	serverAddr   = flag.String("server", getEnvOrDefault("UNKEY_METALD_SERVER_ADDRESS", "https://localhost:8080"), "metald server address")
	userID       = flag.String("user", getEnvOrDefault("UNKEY_METALD_USER_ID", "cli-user"), "user ID for authentication")
	tlsMode      = flag.String("tls-mode", getEnvOrDefault("UNKEY_METALD_TLS_MODE", "spiffe"), "TLS mode: disabled, file, or spiffe")
	spiffeSocket = flag.String("spiffe-socket", getEnvOrDefault("UNKEY_METALD_SPIFFE_SOCKET", "/var/lib/spire/agent/agent.sock"), "SPIFFE agent socket path")
	tlsCert      = flag.String("tls-cert", "", "TLS certificate file (for file mode)")
	tlsKey       = flag.String("tls-key", "", "TLS key file (for file mode)")
	tlsCA        = flag.String("tls-ca", "", "TLS CA file (for file mode)")
	timeout      = flag.Duration("timeout", 30*time.Second, "request timeout")
	jsonOutput   = flag.Bool("json", false, "output results as JSON")

	// VM configuration options
	configFile  = flag.String("config", "", "path to VM configuration file (JSON)")
	template    = flag.String("template", "standard", "VM template: minimal, standard, high-cpu, high-memory, development")
	cpuCount    = flag.Uint("cpu", 0, "number of vCPUs (overrides template)")
	memoryMB    = flag.Uint64("memory", 0, "memory in MB (overrides template)")
	dockerImage = flag.String("docker-image", "", "Docker image to run in VM")
	forceBuild  = flag.Bool("force-build", false, "force rebuild assets even if cached versions exist")

	// Deployment configuration options
	deploymentID = flag.String("deployment-id", "", "deployment ID for deployment operations")
	image        = flag.String("image", "", "container image for deployment")
	vmCount      = flag.Uint("vm-count", 1, "number of VMs in deployment")
)

func main() {
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
	case "create-deployment":
		handleCreateDeployment(ctx, metaldClient, *deploymentID, *image, uint32(*vmCount), uint32(*cpuCount), *memoryMB, *jsonOutput)
	case "update-deployment":
		handleUpdateDeployment(ctx, metaldClient, *deploymentID, *image, uint32(*vmCount), uint32(*cpuCount), *memoryMB, *jsonOutput)
	case "delete-deployment":
		handleDeleteDeployment(ctx, metaldClient, *deploymentID, *jsonOutput)
	case "get-deployment":
		handleGetDeployment(ctx, metaldClient, *deploymentID, *jsonOutput)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Printf(`%s - CLI tool for metald VM operations

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

  Deployment Commands:
  create-deployment           Create a new deployment
  update-deployment           Update an existing deployment
  delete-deployment           Delete a deployment
  get-deployment              Get deployment information
`, os.Args[0], os.Args[0])
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

func handleCreate(ctx context.Context, metaldClient *client.Client, options VMConfigOptions, jsonOutput bool) {
	vmID := ""
	if flag.NArg() > 1 {
		vmID = flag.Arg(1)
	}

	// Create VM configuration from options

	req := &metaldv1.CreateVmRequest{
		VmId:   vmID,
		Config: &metaldv1.VmConfig{},
	}

	// DO STUFF
	_, err := metaldClient.CreateVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}
}

func handleBoot(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for boot command")
	}
	vmID := flag.Arg(1)

	req := &metaldv1.BootVmRequest{
		VmId: vmID,
	}
	resp, err := metaldClient.BootVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to boot VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"vm_id": vmID,
			"state": resp.State.String(),
		})
	} else {
		fmt.Printf("VM boot operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
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

	req := &metaldv1.ShutdownVmRequest{
		VmId:           vmID,
		Force:          force,
		TimeoutSeconds: 30,
	}

	resp, err := metaldClient.ShutdownVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to shutdown VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"vm_id": vmID,
			"state": resp.State.String(),
			"force": force,
		})
	} else {
		fmt.Printf("VM shutdown operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
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

	req := &metaldv1.DeleteVmRequest{
		VmId:  vmID,
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

	req := &metaldv1.GetVmInfoRequest{
		VmId: vmID,
	}
	vmInfo, err := metaldClient.GetVMInfo(ctx, req)
	if err != nil {
		log.Fatalf("Failed to get VM info: %v", err)
	}

	if jsonOutput {
		outputJSON(vmInfo)
	} else {
		fmt.Printf("VM Information:\n")
		fmt.Printf("  VM ID: %s\n", vmInfo.GetVmId())
		fmt.Printf("  State: %s\n", vmInfo.GetState().String())

		if vmInfo.Config != nil {
			fmt.Printf("  Configuration:\n")
			fmt.Printf("    CPUs: %d\n", vmInfo.Config.GetVcpuCount())
			fmt.Printf("    Memory: %d MiB\n", vmInfo.Config.GetMemorySizeMib())
			fmt.Printf("    IP: \n") // DO STUFF
		}

		if vmInfo.Metrics != nil {
			fmt.Printf("  Metrics:\n")
			fmt.Printf("    CPU usage: %.2f%%\n", vmInfo.Metrics.CpuUsagePercent)
			fmt.Printf("    Memory: %d MiB\n", vmInfo.Config.MemorySizeMib)
			fmt.Printf("    Uptime: %d seconds\n", vmInfo.Metrics.UptimeSeconds)
		}
	}
}

func handleList(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	req := &metaldv1.ListVmsRequest{
		PageSize: 50,
	}

	resp, err := metaldClient.ListVMs(ctx, req)
	if err != nil {
		log.Fatalf("Failed to list VMs: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		for _, vm := range resp.Vms {
			fmt.Printf("  - %s: %s (CPUs: %d, Memory: %d MB)\n",
				vm.GetVmId(),
				vm.GetState().String(),
				vm.GetVcpuCount(),
				vm.GetMemorySizeMib(),
			)
		}
	}
}

func handlePause(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for pause command")
	}
	vmID := flag.Arg(1)

	req := &metaldv1.PauseVmRequest{
		VmId: vmID,
	}
	resp, err := metaldClient.PauseVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to pause VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"vm_id": vmID,
			"state": resp.State.String(),
		})
	} else {
		fmt.Printf("VM pause operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  State: %s\n", resp.State.String())
	}
}

func handleResume(ctx context.Context, metaldClient *client.Client, jsonOutput bool) {
	if flag.NArg() < 2 {
		log.Fatal("VM ID is required for resume command")
	}
	vmID := flag.Arg(1)

	req := &metaldv1.ResumeVmRequest{
		VmId: vmID,
	}
	resp, err := metaldClient.ResumeVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to resume VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]any{
			"vm_id": vmID,
			"state": resp.State.String(),
		})
	} else {
		fmt.Printf("VM resume operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
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

	req := &metaldv1.RebootVmRequest{
		VmId:  vmID,
		Force: force,
	}

	resp, err := metaldClient.RebootVM(ctx, req)
	if err != nil {
		log.Fatalf("Failed to reboot VM: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]any{
			"vm_id": vmID,
			"state": resp.State.String(),
			"force": force,
		})
	} else {
		fmt.Printf("VM reboot operation:\n")
		fmt.Printf("  VM ID: %s\n", vmID)
		fmt.Printf("  State: %s\n", resp.State.String())
		fmt.Printf("  Force: %v\n", force)
	}
}

func handleCreateAndBoot(ctx context.Context, metaldClient *client.Client, options VMConfigOptions, jsonOutput bool) {
	vmID := ""
	if flag.NArg() > 1 {
		vmID = flag.Arg(1)
	}

	createReq := &metaldv1.CreateVmRequest{
		VmId:   vmID,
		Config: &metaldv1.VmConfig{},
	}
	log.Printf("createReq: %+v/n", createReq)
	createResp, err := metaldClient.CreateVM(ctx, createReq)
	if err != nil {
		log.Fatalf("Failed to create VM: %v", err)
	}

	// Wait a moment for VM to be fully created
	time.Sleep(2 * time.Second)

	// Boot VM
	bootReq := &metaldv1.BootVmRequest{
		VmId: createReq.GetVmId(),
	}
	bootResp, err := metaldClient.BootVM(ctx, bootReq)
	if err != nil {
		log.Fatalf("Failed to boot VM: %v", err)
	}

	if jsonOutput {
		result := map[string]any{
			"vm_id":        createReq.GetVmId(),
			"create_state": createResp.State.String(),
			"boot_state":   bootResp.State.String(),
		}
		outputJSON(result)
	} else {
		fmt.Printf("VM created and booted successfully:\n")
		fmt.Printf("  VM ID: %s\n", createReq.GetVmId())
		fmt.Printf("  Create State: %s\n", createResp.State.String())
		fmt.Printf("  Boot State: %s\n", bootResp.State.String())
	}
}

func handleCreateDeployment(ctx context.Context, metaldClient *client.Client, deploymentID, image string, vmCount, cpu uint32, memorySizeMB uint64, jsonOutput bool) {
	if deploymentID == "" {
		log.Fatal("Deployment ID is required for create-deployment command")
	}
	if image == "" {
		log.Fatal("Image is required for create-deployment command")
	}

	req := &metaldv1.CreateDeploymentRequest{
		Deployment: &metaldv1.DeploymentRequest{
			DeploymentId:  deploymentID,
			Image:         image,
			VmCount:       vmCount,
			Cpu:           cpu,
			MemorySizeMib: memorySizeMB,
		},
	}

	resp, err := metaldClient.CreateDeployment(ctx, req)
	if err != nil {
		log.Fatalf("Failed to create deployment: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"deployment_id": deploymentID,
			"vm_ids":        resp.VmIds,
			"vm_count":      len(resp.VmIds),
		})
	} else {
		fmt.Printf("Deployment created successfully:\n")
		fmt.Printf("  Deployment ID: %s\n", deploymentID)
		fmt.Printf("  VM Count: %d\n", len(resp.VmIds))
		fmt.Printf("  VM IDs:\n")
		for _, vmID := range resp.VmIds {
			fmt.Printf("    - %s\n", vmID)
		}
	}
}

func handleUpdateDeployment(ctx context.Context, metaldClient *client.Client, deploymentID, image string, vmCount, cpu uint32, memorySizeMB uint64, jsonOutput bool) {
	if deploymentID == "" {
		log.Fatal("Deployment ID is required for update-deployment command")
	}

	req := &metaldv1.UpdateDeploymentRequest{
		Deployment: &metaldv1.DeploymentRequest{
			DeploymentId:  deploymentID,
			Image:         image,
			VmCount:       vmCount,
			Cpu:           cpu,
			MemorySizeMib: memorySizeMB,
		},
	}

	resp, err := metaldClient.UpdateDeployment(ctx, req)
	if err != nil {
		log.Fatalf("Failed to update deployment: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"deployment_id": deploymentID,
			"vm_ids":        resp.VmIds,
			"vm_count":      len(resp.VmIds),
		})
	} else {
		fmt.Printf("Deployment updated successfully:\n")
		fmt.Printf("  Deployment ID: %s\n", deploymentID)
		fmt.Printf("  VM Count: %d\n", len(resp.VmIds))
		fmt.Printf("  VM IDs:\n")
		for _, vmID := range resp.VmIds {
			fmt.Printf("    - %s\n", vmID)
		}
	}
}

func handleDeleteDeployment(ctx context.Context, metaldClient *client.Client, deploymentID string, jsonOutput bool) {
	if deploymentID == "" {
		log.Fatal("Deployment ID is required for delete-deployment command")
	}

	req := &metaldv1.DeleteDeploymentRequest{
		DeploymentId: deploymentID,
	}

	_, err := metaldClient.DeleteDeployment(ctx, req)
	if err != nil {
		log.Fatalf("Failed to delete deployment: %v", err)
	}

	if jsonOutput {
		outputJSON(map[string]interface{}{
			"deployment_id": deploymentID,
			"deleted":       true,
		})
	} else {
		fmt.Printf("Deployment deleted successfully:\n")
		fmt.Printf("  Deployment ID: %s\n", deploymentID)
	}
}

func handleGetDeployment(ctx context.Context, metaldClient *client.Client, deploymentID string, jsonOutput bool) {
	if deploymentID == "" {
		log.Fatal("Deployment ID is required for get-deployment command")
	}

	req := &metaldv1.GetDeploymentRequest{
		DeploymentId: deploymentID,
	}

	resp, err := metaldClient.GetDeployment(ctx, req)
	if err != nil {
		log.Fatalf("Failed to get deployment: %v", err)
	}

	if jsonOutput {
		outputJSON(resp)
	} else {
		fmt.Printf("Deployment Information:\n")
		fmt.Printf("  Deployment ID: %s\n", resp.DeploymentId)
		fmt.Printf("  VM Count: %d\n", len(resp.Vms))
		fmt.Printf("  VMs:\n")
		for _, vm := range resp.Vms {
			fmt.Printf("    - ID: %s\n", vm.Id)
			fmt.Printf("      Host: %s\n", vm.Host)
			fmt.Printf("      State: %s\n", vm.State.String())
			fmt.Printf("      Port: %d\n", vm.Port)
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
