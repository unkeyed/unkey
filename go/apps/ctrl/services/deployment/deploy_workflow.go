package deployment

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1/vmprovisionerv1connect"
	partitionv1 "github.com/unkeyed/unkey/go/gen/proto/partition/v1"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	partitiondb "github.com/unkeyed/unkey/go/pkg/partition/db"
	"github.com/unkeyed/unkey/go/pkg/uid"
	"google.golang.org/protobuf/proto"
)

// DeployWorkflow orchestrates the complete build and deployment process using Hydra
type DeployWorkflow struct {
	db           db.Database
	partitionDB  db.Database
	logger       logging.Logger
	metaldClient vmprovisionerv1connect.VmServiceClient
}

// NewDeployWorkflow creates a new deploy workflow instance
func NewDeployWorkflow(database db.Database, partitionDB db.Database,
	logger logging.Logger, metaldClient vmprovisionerv1connect.VmServiceClient,
) *DeployWorkflow {
	return &DeployWorkflow{
		db:           database,
		partitionDB:  partitionDB,
		logger:       logger,
		metaldClient: metaldClient,
	}
}

// Name returns the workflow name for registration
func (w *DeployWorkflow) Name() string {
	return "deployment"
}

// DeployRequest defines the input for the deploy workflow
type DeployRequest struct {
	WorkspaceID  string `json:"workspace_id"`
	ProjectID    string `json:"project_id"`
	KeyspaceID   string `json:"keyspace_id"`
	DeploymentID string `json:"deployment_id"`
	DockerImage  string `json:"docker_image"`
	Hostname     string `json:"hostname"`
}

// BuildInfo holds build metadata from initialization step
type BuildInfo struct {
	BuildID     string `json:"build_id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	VersionID   string `json:"version_id"`
	DockerImage string `json:"docker_image"`
}

// SubmissionResult holds the result of build submission
type SubmissionResult struct {
	BuildID   string `json:"build_id"`
	Submitted bool   `json:"submitted"`
}

// BuildResult holds the final build outcome
type BuildResult struct {
	BuildID  string `json:"build_id"`
	Status   string `json:"status"`
	ErrorMsg string `json:"error_message,omitempty"`
}

// DeploymentResult holds the deployment outcome
type DeploymentResult struct {
	DeploymentID string `json:"deployment_id"`
	Status       string `json:"status"`
}

// Run executes the complete build and deployment workflow
func (w *DeployWorkflow) Run(ctx hydra.WorkflowContext, req *DeployRequest) error {
	w.logger.Info("starting deployment workflow",
		"execution_id", ctx.ExecutionID(),
		"deployment_id", req.DeploymentID,
		"docker_image", req.DockerImage,
		"workspace_id", req.WorkspaceID,
		"project_id", req.ProjectID,
		"hostname", req.Hostname)

	// Step 1: Generate build ID
	buildID, err := hydra.Step(ctx, "generate-build-id", func(stepCtx context.Context) (string, error) {
		id := uid.New(uid.BuildPrefix)
		w.logger.Info("generated build ID", "build_id", id)
		return id, nil
	})
	if err != nil {
		w.logger.Error("failed to generate build ID", "error", err)
		return err
	}

	w.logger.Info("proceeding with build", "build_id", buildID)

	// Step 2: Log deployment pending
	err = hydra.StepVoid(ctx, "log-deployment-pending", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "pending",
			Message:      sql.NullString{String: "Deployment queued and ready to start", Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log deployment pending", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 3: Insert build into database
	err = hydra.StepVoid(ctx, "insert-build", func(stepCtx context.Context) error {
		w.logger.Info("inserting build into database", "build_id", buildID)
		insertErr := db.Query.InsertBuild(stepCtx, w.db.RW(), db.InsertBuildParams{
			ID:           buildID,
			WorkspaceID:  req.WorkspaceID,
			ProjectID:    req.ProjectID,
			DeploymentID: req.DeploymentID,
			CreatedAt:    time.Now().UnixMilli(),
		})
		if insertErr != nil {
			return fmt.Errorf("failed to create build record: %w", insertErr)
		}
		w.logger.Info("build record created successfully", "build_id", buildID)
		return nil
	})
	if err != nil {
		w.logger.Error("failed to insert build", "error", err, "build_id", buildID)
		return err
	}

	// Step 4: Update version status to building
	_, err = hydra.Step(ctx, "update-version-building", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("updating deployment status to building", "deployment_id", req.DeploymentID)
		updateErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusBuilding,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		w.logger.Info("deployment status updated to building", "deployment_id", req.DeploymentID)
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to initialize build", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 5: Update build status to running
	_, err = hydra.Step(ctx, "update-build-running", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("updating build status to running", "build_id", buildID)
		now := time.Now().UnixMilli()
		runningErr := db.Query.UpdateBuildStatus(stepCtx, w.db.RW(), db.UpdateBuildStatusParams{
			ID:     buildID,
			Status: db.BuildsStatusRunning,
			Now:    sql.NullInt64{Valid: true, Int64: now},
		})
		if runningErr != nil {
			return nil, fmt.Errorf("failed to update build status to running: %w", runningErr)
		}
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to update build status to running", "error", err, "build_id", buildID)
		return err
	}

	// Step 6: Log downloading Docker image
	err = hydra.StepVoid(ctx, "log-downloading-docker-image", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "downloading_docker_image",
			Message:      sql.NullString{String: fmt.Sprintf("Downloading Docker image: %s", req.DockerImage), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log downloading Docker image", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 7: Create VM (network call to metald)
	createResult, err := hydra.Step(ctx, "metald-create-vm", func(stepCtx context.Context) (*vmprovisionerv1.CreateVmResponse, error) {
		w.logger.Info("creating VM for deployment", "deployment_id", req.DeploymentID, "docker_image", req.DockerImage, "workspace_id", req.WorkspaceID, "project_id", req.ProjectID)

		// Create VM configuration for Docker backend
		vmConfig := &vmprovisionerv1.VmConfig{
			Cpu: &vmprovisionerv1.CpuConfig{
				VcpuCount: 1,
			},
			Memory: &vmprovisionerv1.MemoryConfig{
				SizeBytes: 536870912, // 512MB
			},
			Boot: &vmprovisionerv1.BootConfig{
				KernelPath: "/boot/vmlinux",
				InitrdPath: "/boot/initrd",
				KernelArgs: "console=ttyS0 quiet",
			},
			Storage: []*vmprovisionerv1.StorageDevice{{
				Id:   "root",
				Path: "/dev/vda",
			}},
			Metadata: map[string]string{
				"docker_image":  req.DockerImage,
				"exposed_ports": "8080/tcp",
				"env_vars":      "PORT=8080",
				"version_id":    req.DeploymentID,
				"workspace_id":  req.WorkspaceID,
				"project_id":    req.ProjectID,
				"created_by":    "deploy-workflow",
			},
		}

		// Make real metald CreateVm call
		resp, err := w.metaldClient.CreateVm(stepCtx, connect.NewRequest(&vmprovisionerv1.CreateVmRequest{
			Config: vmConfig,
		}))
		if err != nil {
			w.logger.Error("metald CreateVm call failed", "error", err, "docker_image", req.DockerImage)
			return nil, fmt.Errorf("failed to create VM: %w", err)
		}

		w.logger.Info("VM created successfully", "vm_id", resp.Msg.VmId, "state", resp.Msg.State.String(), "docker_image", req.DockerImage)

		return resp.Msg, nil
	})
	if err != nil {
		w.logger.Error("VM creation failed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	w.logger.Info("VM creation completed", "vm_id", createResult.VmId, "state", createResult.State.String())

	// Step 8: Log building rootfs
	err = hydra.StepVoid(ctx, "log-building-rootfs", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "building_rootfs",
			Message:      sql.NullString{String: fmt.Sprintf("Building rootfs from Docker image: %s", req.DockerImage), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log building rootfs", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 9: Log uploading rootfs
	err = hydra.StepVoid(ctx, "log-uploading-rootfs", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "uploading_rootfs",
			Message:      sql.NullString{String: "Uploading rootfs image to storage", Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log uploading rootfs", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 10: Update build status to succeeded
	_, err = hydra.Step(ctx, "update-build-succeeded", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("updating build status to succeeded", "build_id", buildID)
		successErr := db.Query.UpdateBuildSucceeded(stepCtx, w.db.RW(), db.UpdateBuildSucceededParams{
			ID:  buildID,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if successErr != nil {
			return nil, fmt.Errorf("failed to update build status to succeeded: %w", successErr)
		}
		w.logger.Info("build status updated to succeeded", "build_id", buildID)
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to update build status to succeeded", "error", err, "build_id", buildID)
		return err
	}

	// Step 11: Log creating VM
	err = hydra.StepVoid(ctx, "log-creating-vm", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "creating_vm",
			Message:      sql.NullString{String: fmt.Sprintf("Creating VM for version: %s", req.DeploymentID), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log creating VM", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 12: Update version status to deploying
	_, err = hydra.Step(ctx, "update-version-deploying", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("starting deployment", "deployment_id", req.DeploymentID)

		deployingErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusDeploying,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if deployingErr != nil {
			return nil, fmt.Errorf("failed to update version status to deploying: %w", deployingErr)
		}
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to update version status to deploying", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 13: Skip VM status polling for Docker backend (VM is immediately ready)
	w.logger.Info("skipping VM status polling for Docker backend", "vm_id", createResult.VmId)

	// Step 14: Boot VM (network call to metald)
	_, err = hydra.Step(ctx, "metald-boot-vm", func(stepCtx context.Context) (*vmprovisionerv1.BootVmResponse, error) {
		w.logger.Info("booting VM", "vm_id", createResult.VmId)

		// Make real metald BootVm call
		resp, err := w.metaldClient.BootVm(stepCtx, connect.NewRequest(&vmprovisionerv1.BootVmRequest{
			VmId: createResult.VmId,
		}))
		if err != nil {
			w.logger.Error("metald BootVm call failed", "error", err, "vm_id", createResult.VmId)
			return nil, fmt.Errorf("failed to boot VM: %w", err)
		}

		if !resp.Msg.Success {
			w.logger.Error("VM boot was not successful", "vm_id", createResult.VmId, "state", resp.Msg.State.String())
			return nil, fmt.Errorf("VM boot was not successful, state: %s", resp.Msg.State.String())
		}

		w.logger.Info("VM booted successfully", "vm_id", createResult.VmId, "state", resp.Msg.State.String())
		return resp.Msg, nil
	})
	if err != nil {
		w.logger.Error("VM boot failed", "error", err, "vm_id", createResult.VmId)
		return err
	}

	w.logger.Info("VM boot completed successfully", "vm_id", createResult.VmId)

	// Step 15: Get VM info to retrieve port mappings
	vmInfo, err := hydra.Step(ctx, "metald-get-vm-info", func(stepCtx context.Context) (*vmprovisionerv1.GetVmInfoResponse, error) {
		w.logger.Info("getting VM info for port mappings", "vm_id", createResult.VmId)

		resp, err := w.metaldClient.GetVmInfo(stepCtx, connect.NewRequest(&vmprovisionerv1.GetVmInfoRequest{
			VmId: createResult.VmId,
		}))
		if err != nil {
			w.logger.Error("metald GetVmInfo call failed", "error", err, "vm_id", createResult.VmId)
			return nil, fmt.Errorf("failed to get VM info: %w", err)
		}

		if resp.Msg.NetworkInfo != nil {
			w.logger.Info("VM info retrieved successfully", "vm_id", createResult.VmId, "port_mappings", len(resp.Msg.NetworkInfo.PortMappings))
		} else {
			w.logger.Warn("VM info retrieved but no network info", "vm_id", createResult.VmId)
		}

		return resp.Msg, nil
	})
	if err != nil {
		w.logger.Error("failed to get VM info", "error", err, "vm_id", createResult.VmId)
		return err
	}

	// Step 16: Insert VM into partition database
	err = hydra.StepVoid(ctx, "insert-vm-partition-db", func(stepCtx context.Context) error {
		w.logger.Info("inserting VM into partition database", "vm_id", createResult.VmId, "deployment_id", req.DeploymentID)

		// Validate partition DB connection before proceeding
		if w.partitionDB == nil {
			w.logger.Error("CRITICAL: partition database not initialized")
			return fmt.Errorf("partition database not initialized")
		}

		// Extract host port from VM info - find port mapping for container port 8080
		var hostPort int32 = 8080 // default fallback
		if vmInfo.NetworkInfo != nil && len(vmInfo.NetworkInfo.PortMappings) > 0 {
			for _, portMapping := range vmInfo.NetworkInfo.PortMappings {
				if portMapping.ContainerPort == 8080 {
					hostPort = portMapping.HostPort
					break
				}
			}
		}

		// Create VM record in partition database
		vmParams := partitiondb.UpsertVMParams{
			ID:           createResult.VmId,
			DeploymentID: req.DeploymentID,
			Region:       "us-east-1",
			PrivateIp: sql.NullString{
				String: "127.0.0.1",
				Valid:  true,
			},
			Port: sql.NullInt32{
				Int32: hostPort,
				Valid: true,
			},
			CpuMillicores: 1000,
			MemoryMb:      512,
			Status:        partitiondb.VmsStatusRunning,
			HealthStatus:  partitiondb.VmsHealthStatusHealthy,
		}

		if err := partitiondb.Query.UpsertVM(stepCtx, w.partitionDB.RW(), vmParams); err != nil {
			w.logger.Error("failed to create VM in partition DB", "error", err, "vm_id", createResult.VmId)
			return fmt.Errorf("failed to create VM %s in partition DB: %w", createResult.VmId, err)
		}

		w.logger.Info("VM inserted into partition database successfully", "vm_id", createResult.VmId, "host_port", hostPort)
		return nil
	})
	if err != nil {
		w.logger.Error("failed to insert VM into partition database", "error", err, "vm_id", createResult.VmId)
		return err
	}

	// Step 17: Create/update gateway configuration
	err = hydra.StepVoid(ctx, "create-gateway-config", func(stepCtx context.Context) error {
		// Only create gateway config if hostname is provided
		if req.Hostname == "" {
			w.logger.Info("no hostname provided, skipping gateway configuration")
			return nil
		}

		w.logger.Info("creating gateway configuration", "hostname", req.Hostname, "deployment_id", req.DeploymentID)

		// Validate partition DB connection
		if w.partitionDB == nil {
			w.logger.Error("CRITICAL: partition database not initialized for gateway config")
			return fmt.Errorf("partition database not initialized for gateway config")
		}

		// Create VM protobuf objects for gateway config
		vms := []*partitionv1.VM{
			{
				Id:     createResult.VmId,
				Region: "us-east-1", // TODO: make this configurable
			},
		}

		gatewayConfig := &partitionv1.GatewayConfig{
			DeploymentId: req.DeploymentID,
			IsEnabled:    true,
			Vms:          vms,
		}

		// Only add AuthConfig if we have a KeyspaceID
		if req.KeyspaceID != "" {
			gatewayConfig.AuthConfig = &partitionv1.AuthConfig{
				RequireApiKey:  true,
				RequiredScopes: []string{},
				KeyspaceId:     req.KeyspaceID,
				AllowAnonymous: false,
				Enabled:        true,
			}
		}

		// Marshal protobuf to bytes
		configBytes, err := proto.Marshal(gatewayConfig)
		if err != nil {
			w.logger.Error("failed to marshal gateway config", "error", err)
			return fmt.Errorf("failed to marshal gateway config: %w", err)
		}

		// Insert gateway config into partition database
		params := partitiondb.UpsertGatewayParams{
			Hostname: req.Hostname,
			Config:   configBytes,
		}

		if err := partitiondb.Query.UpsertGateway(stepCtx, w.partitionDB.RW(), params); err != nil {
			w.logger.Error("failed to upsert gateway config", "error", err, "hostname", req.Hostname)
			return fmt.Errorf("failed to upsert gateway config: %w", err)
		}
		w.logger.Info("gateway configuration created successfully", "hostname", req.Hostname, "vm_id", createResult.VmId)
		return nil
	})
	if err != nil {
		w.logger.Error("failed to create gateway configuration", "error", err, "hostname", req.Hostname)
		return err
	}

	// Step 18: Log booting VM
	err = hydra.StepVoid(ctx, "log-booting-vm", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "booting_vm",
			Message:      sql.NullString{String: fmt.Sprintf("VM booted successfully: %s", createResult.VmId), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log booting VM", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 19: Assign domains (create route entries)
	assignedHostnames, err := hydra.Step(ctx, "assign-domains", func(stepCtx context.Context) ([]string, error) {
		w.logger.Info("assigning domains to version", "deployment_id", req.DeploymentID)

		var hostnames []string

		// Generate primary hostname for this deployment
		// Use Git info for hostname generation
		gitInfo := git.GetInfo()
		branch := "main"               // Default branch
		identifier := req.DeploymentID // Use full version ID as identifier

		if gitInfo.IsRepo {
			if gitInfo.Branch != "" {
				branch = gitInfo.Branch
			}
			if gitInfo.CommitSHA != "" {
				identifier = gitInfo.CommitSHA
			}
		}

		// Generate hostnames: branch-identifier-workspace.unkey.app
		// Replace underscores with dashes for valid hostname format
		cleanIdentifier := strings.ReplaceAll(identifier, "_", "-")
		primaryHostname := fmt.Sprintf("%s-%s-%s.unkey.app", branch, cleanIdentifier, req.WorkspaceID)

		// Create route entry for primary hostname
		routeID := uid.New("route")
		insertErr := db.Query.InsertHostnameRoute(stepCtx, w.db.RW(), db.InsertHostnameRouteParams{
			ID:           routeID,
			WorkspaceID:  req.WorkspaceID,
			ProjectID:    req.ProjectID,
			Hostname:     primaryHostname,
			DeploymentID: req.DeploymentID,
			IsEnabled:    true,
			CreatedAt:    time.Now().UnixMilli(),
			UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if insertErr != nil {
			w.logger.Error("failed to create route", "error", insertErr, "hostname", primaryHostname, "deployment_id", req.DeploymentID)
			return nil, fmt.Errorf("failed to create route for hostname %s: %w", primaryHostname, insertErr)
		}

		hostnames = append(hostnames, primaryHostname)
		w.logger.Info("primary domain assigned successfully", "hostname", primaryHostname, "deployment_id", req.DeploymentID, "route_id", routeID)

		// Add localhost:port hostname for development
		w.logger.Info("checking for port mappings", "has_network_info", vmInfo.NetworkInfo != nil, "port_mappings_count", func() int {
			if vmInfo.NetworkInfo != nil {
				return len(vmInfo.NetworkInfo.PortMappings)
			}
			return 0
		}())

		if vmInfo.NetworkInfo != nil && len(vmInfo.NetworkInfo.PortMappings) > 0 {
			for _, portMapping := range vmInfo.NetworkInfo.PortMappings {
				localhostHostname := fmt.Sprintf("localhost:%d", portMapping.HostPort)

				// Create route entry for localhost:port
				localhostRouteID := uid.New("route")
				insertErr := db.Query.InsertHostnameRoute(stepCtx, w.db.RW(), db.InsertHostnameRouteParams{
					ID:           localhostRouteID,
					WorkspaceID:  req.WorkspaceID,
					ProjectID:    req.ProjectID,
					Hostname:     localhostHostname,
					DeploymentID: req.DeploymentID,
					IsEnabled:    true,
					CreatedAt:    time.Now().UnixMilli(),
					UpdatedAt:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				})
				if insertErr != nil {
					w.logger.Error("failed to create localhost route", "error", insertErr, "hostname", localhostHostname, "deployment_id", req.DeploymentID)
					return nil, fmt.Errorf("failed to create route for hostname %s: %w", localhostHostname, insertErr)
				}

				hostnames = append(hostnames, localhostHostname)
				w.logger.Info("localhost domain assigned successfully", "hostname", localhostHostname, "deployment_id", req.DeploymentID, "route_id", localhostRouteID, "container_port", portMapping.ContainerPort, "host_port", portMapping.HostPort)
			}
		}

		return hostnames, nil
	})
	if err != nil {
		w.logger.Error("domain assignment failed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	err = hydra.StepVoid(ctx, "generate-certificates", func(stepCtx context.Context) error {
		domains := []db.InsertDomainParams{}
		now := time.Now().UnixMilli()
		for _, domain := range assignedHostnames {
			domains = append(domains, db.InsertDomainParams{
				ID:              uid.New(uid.DomainPrefix),
				WorkspaceID:     req.WorkspaceID,
				ProjectID:       req.ProjectID,
				Domain:          domain,
				Type:            db.DomainsTypeGenerated,
				SubdomainConfig: []byte("{}"),
				CreatedAt:       now,
				UpdatedAt:       sql.NullInt64{Valid: true, Int64: now},
			})
		}

		if req.Hostname != "" {
			domainId := uid.New(uid.DomainPrefix)
			domains = append(domains, db.InsertDomainParams{
				ID:              domainId,
				WorkspaceID:     req.WorkspaceID,
				ProjectID:       req.ProjectID,
				Domain:          req.Hostname,
				Type:            db.DomainsTypeCustom,
				SubdomainConfig: []byte("{}"),
				CreatedAt:       now,
				UpdatedAt:       sql.NullInt64{Valid: true, Int64: now},
			})

			err = db.Query.InsertDomainChallenge(ctx.Context(), w.db.RW(), db.InsertDomainChallengeParams{
				WorkspaceID:   req.WorkspaceID,
				DomainID:      domainId,
				Token:         sql.NullString{Valid: false, String: ""},
				Authorization: sql.NullString{Valid: false, String: ""},
				Status:        db.DomainChallengesStatusWaiting,
				CreatedAt:     now,
				UpdatedAt:     sql.NullInt64{Valid: false, Int64: 0},
				ExpiresAt:     sql.NullInt64{Valid: false, Int64: 0},
			})
			if err != nil {
				w.logger.Error("failed to insert domain challenge", "error", err, "deployment_id", req.DeploymentID)
				return err
			}
		}

		if len(domains) > 0 {
			err = db.BulkQuery.InsertDomains(ctx.Context(), w.db.RW(), domains)
			if err != nil {
				w.logger.Error("failed to insert domains", "error", err, "deployment_id", req.DeploymentID)
				return err
			}
		}

		return nil
	})
	if err != nil {
		w.logger.Error("failed to insert domains", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 20: Log assigning domains
	err = hydra.StepVoid(ctx, "log-assigning-domains", func(stepCtx context.Context) error {
		var message string
		if len(assignedHostnames) > 0 {
			message = fmt.Sprintf("Assigned hostnames: %s", strings.Join(assignedHostnames, ", "))
		} else {
			message = "Domain assignment completed"
		}
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "assigning_domains",
			Message:      sql.NullString{String: message, Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log assigning domains", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 21: Update version status to active
	_, err = hydra.Step(ctx, "update-version-active", func(stepCtx context.Context) (*DeploymentResult, error) {
		completionTime := time.Now().UnixMilli()
		w.logger.Info("updating deployment status to active", "deployment_id", req.DeploymentID, "completion_time", completionTime)
		activeErr := db.Query.UpdateDeploymentStatus(stepCtx, w.db.RW(), db.UpdateDeploymentStatusParams{
			ID:        req.DeploymentID,
			Status:    db.DeploymentsStatusActive,
			UpdatedAt: sql.NullInt64{Valid: true, Int64: completionTime},
		})
		if activeErr != nil {
			w.logger.Error("failed to update version status to active", "error", activeErr, "deployment_id", req.DeploymentID)
			return nil, fmt.Errorf("failed to update version status to active: %w", activeErr)
		}

		w.logger.Info("deployment complete", "deployment_id", req.DeploymentID, "status", "active")

		return &DeploymentResult{
			DeploymentID: req.DeploymentID,
			Status:       "active",
		}, nil
	})
	if err != nil {
		w.logger.Error("deployment failed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 22: Health check container (using host port mapping)
	err = hydra.StepVoid(ctx, "health-check-container", func(stepCtx context.Context) error {
		if vmInfo.NetworkInfo == nil || len(vmInfo.NetworkInfo.PortMappings) == 0 {
			return fmt.Errorf("no port mappings available for container health check")
		}

		// Find the port mapping for container port 8080
		var hostPort int32
		for _, portMapping := range vmInfo.NetworkInfo.PortMappings {
			if portMapping.ContainerPort == 8080 {
				hostPort = portMapping.HostPort
				break
			}
		}

		if hostPort == 0 {
			return fmt.Errorf("no host port mapping found for container port 8080")
		}

		// Try multiple host addresses to reach the Docker host
		// Prioritize Docker's magic domain names
		hostAddresses := []string{
			"host.docker.internal",    // Docker Desktop (Windows/Mac) and some Linux setups
			"gateway.docker.internal", // Docker gateway
			"172.17.0.1",              // Default Docker bridge gateway
			"172.18.0.1",              // Alternative Docker bridge
		}

		client := &http.Client{Timeout: 10 * time.Second}

		for _, hostAddr := range hostAddresses {
			healthURL := fmt.Sprintf("http://%s:%d/v1/liveness", hostAddr, hostPort)
			w.logger.Info("trying container health check", "url", healthURL, "host_port", hostPort, "deployment_id", req.DeploymentID)

			resp, err := client.Get(healthURL)
			if err != nil {
				w.logger.Warn("health check failed for host address", "error", err, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode == http.StatusOK {
				w.logger.Info("container is healthy", "host_addr", hostAddr, "deployment_id", req.DeploymentID)
				return nil
			}

			w.logger.Warn("health check returned non-200 status", "status", resp.StatusCode, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
		}

		return fmt.Errorf("health check failed on all host addresses: %v", hostAddresses)
	})
	if err != nil {
		w.logger.Error("container health check failed", "error", err, "deployment_id", req.DeploymentID)
		// Don't fail the deployment, just skip OpenAPI scraping
	}

	// Step 23: Scrape OpenAPI spec from container (using host port mapping)
	openapiSpec, err := hydra.Step(ctx, "scrape-openapi-spec", func(stepCtx context.Context) (string, error) {
		if vmInfo.NetworkInfo == nil || len(vmInfo.NetworkInfo.PortMappings) == 0 {
			w.logger.Warn("no port mappings available for OpenAPI scraping", "deployment_id", req.DeploymentID)
			return "", nil
		}

		// Find the port mapping for container port 8080
		var hostPort int32
		for _, portMapping := range vmInfo.NetworkInfo.PortMappings {
			if portMapping.ContainerPort == 8080 {
				hostPort = portMapping.HostPort
				break
			}
		}

		if hostPort == 0 {
			w.logger.Warn("no host port mapping found for container port 8080", "deployment_id", req.DeploymentID)
			return "", nil
		}

		// Try multiple host addresses to reach the Docker host
		hostAddresses := []string{
			"host.docker.internal",    // Docker Desktop (Windows/Mac) and some Linux setups
			"gateway.docker.internal", // Docker gateway
			"172.17.0.1",              // Default Docker bridge gateway
			"172.18.0.1",              // Alternative Docker bridge
		}

		client := &http.Client{Timeout: 10 * time.Second}

		for _, hostAddr := range hostAddresses {
			openapiURL := fmt.Sprintf("http://%s:%d/openapi.yaml", hostAddr, hostPort)
			w.logger.Info("trying to scrape OpenAPI spec", "url", openapiURL, "host_port", hostPort, "deployment_id", req.DeploymentID)

			resp, err := client.Get(openapiURL)
			if err != nil {
				w.logger.Warn("OpenAPI scraping failed for host address", "error", err, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
				continue
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				w.logger.Warn("OpenAPI endpoint returned non-200 status", "status", resp.StatusCode, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
				continue
			}

			// Read the OpenAPI spec
			specBytes, err := io.ReadAll(resp.Body)
			if err != nil {
				w.logger.Warn("failed to read OpenAPI spec response", "error", err, "host_addr", hostAddr, "deployment_id", req.DeploymentID)
				continue
			}

			w.logger.Info("OpenAPI spec scraped successfully", "host_addr", hostAddr, "deployment_id", req.DeploymentID, "spec_size", len(specBytes))
			return string(specBytes), nil
		}

		return "", fmt.Errorf("failed to scrape OpenAPI spec from all host addresses: %v", hostAddresses)
	})
	if err != nil {
		w.logger.Error("failed to scrape OpenAPI spec", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 24: Update gateway config with OpenAPI spec
	err = hydra.StepVoid(ctx, "update-gateway-config-openapi", func(stepCtx context.Context) error {
		// Only update if we have both hostname and OpenAPI spec
		if req.Hostname == "" || openapiSpec == "" {
			w.logger.Info("skipping gateway config OpenAPI update",
				"has_hostname", req.Hostname != "",
				"has_openapi_spec", openapiSpec != "",
				"deployment_id", req.DeploymentID)
			return nil
		}

		w.logger.Info("updating gateway config with OpenAPI spec", "hostname", req.Hostname, "deployment_id", req.DeploymentID, "spec_size", len(openapiSpec))

		// Fetch existing gateway config
		existingConfig, err := partitiondb.Query.FindGatewayByHostname(stepCtx, w.partitionDB.RO(), req.Hostname)
		if err != nil {
			w.logger.Error("failed to fetch existing gateway config", "error", err, "hostname", req.Hostname)
			return fmt.Errorf("failed to fetch existing gateway config: %w", err)
		}

		// Unmarshal existing config
		var gatewayConfig partitionv1.GatewayConfig
		if err := proto.Unmarshal(existingConfig.Config, &gatewayConfig); err != nil {
			w.logger.Error("failed to unmarshal existing gateway config", "error", err, "hostname", req.Hostname)
			return fmt.Errorf("failed to unmarshal existing gateway config: %w", err)
		}

		// Add or update ValidationConfig with OpenAPI spec
		if gatewayConfig.ValidationConfig == nil {
			gatewayConfig.ValidationConfig = &partitionv1.ValidationConfig{}
		}
		gatewayConfig.ValidationConfig.OpenapiSpec = openapiSpec
		gatewayConfig.ValidationConfig.Enabled = true
		gatewayConfig.ValidationConfig.ValidateRequest = true
		gatewayConfig.ValidationConfig.ValidateResponse = false // Set to false by default
		gatewayConfig.ValidationConfig.StrictMode = false       // Set to false by default

		// Marshal updated config
		configBytes, err := proto.Marshal(&gatewayConfig)
		if err != nil {
			w.logger.Error("failed to marshal updated gateway config", "error", err)
			return fmt.Errorf("failed to marshal updated gateway config: %w", err)
		}

		// Update gateway config in partition database
		params := partitiondb.UpsertGatewayParams{
			Hostname: req.Hostname,
			Config:   configBytes,
		}

		if err := partitiondb.Query.UpsertGateway(stepCtx, w.partitionDB.RW(), params); err != nil {
			w.logger.Error("failed to update gateway config with OpenAPI spec", "error", err, "hostname", req.Hostname)
			return fmt.Errorf("failed to update gateway config with OpenAPI spec: %w", err)
		}

		w.logger.Info("gateway config updated with OpenAPI spec successfully", "hostname", req.Hostname, "deployment_id", req.DeploymentID)
		return nil
	})
	if err != nil {
		w.logger.Error("failed to update gateway config with OpenAPI spec", "error", err, "deployment_id", req.DeploymentID)
		// Don't fail the deployment for this
	}

	// Step 25: Store OpenAPI spec in database
	err = hydra.StepVoid(ctx, "store-openapi-spec", func(stepCtx context.Context) error {
		if openapiSpec == "" {
			w.logger.Info("no OpenAPI spec to store", "deployment_id", req.DeploymentID)
			return nil
		}

		// Store in database
		err := db.Query.UpdateDeploymentOpenapiSpec(stepCtx, w.db.RW(), db.UpdateDeploymentOpenapiSpecParams{
			ID:          req.DeploymentID,
			OpenapiSpec: sql.NullString{String: openapiSpec, Valid: true},
			UpdatedAt:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			w.logger.Warn("failed to store OpenAPI spec in database", "error", err, "deployment_id", req.DeploymentID)
			return nil // Don't fail the deployment
		}

		w.logger.Info("OpenAPI spec stored in database successfully", "deployment_id", req.DeploymentID, "spec_size", len(openapiSpec))
		return nil
	})
	if err != nil {
		w.logger.Error("failed to store OpenAPI spec", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	// Step 26: Log completed
	err = hydra.StepVoid(ctx, "log-completed", func(stepCtx context.Context) error {
		return db.Query.InsertDeploymentStep(stepCtx, w.db.RW(), db.InsertDeploymentStepParams{
			DeploymentID: req.DeploymentID,
			Status:       "completed",
			Message:      sql.NullString{String: "Version deployment completed successfully", Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log completed", "error", err, "deployment_id", req.DeploymentID)
		return err
	}

	w.logger.Info("deployment workflow stage completed successfully", "deployment_id", req.DeploymentID, "vm_id", createResult.VmId)

	w.logger.Info("deployment workflow completed",
		"execution_id", ctx.ExecutionID(),
		"build_id", buildID,
		"deployment_id", req.DeploymentID,
		"status", "succeeded",
		"workspace_id", req.WorkspaceID,
		"project_id", req.ProjectID,
		"docker_image", req.DockerImage,
		"hostname", req.Hostname)

	return nil
}
