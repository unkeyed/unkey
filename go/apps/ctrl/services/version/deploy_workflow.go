package version

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"connectrpc.com/connect"
	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// DeployWorkflow orchestrates the complete build and deployment process using Hydra
type DeployWorkflow struct {
	db           db.Database
	logger       logging.Logger
	metaldClient vmprovisionerv1connect.VmServiceClient
}

// NewDeployWorkflow creates a new deploy workflow instance
func NewDeployWorkflow(database db.Database, logger logging.Logger, metaldClient vmprovisionerv1connect.VmServiceClient) *DeployWorkflow {
	return &DeployWorkflow{
		db:           database,
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
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	VersionID   string `json:"version_id"`
	DockerImage string `json:"docker_image"`
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
	VersionID string `json:"version_id"`
	Status    string `json:"status"`
}

// Run executes the complete build and deployment workflow
func (w *DeployWorkflow) Run(ctx hydra.WorkflowContext, req *DeployRequest) error {
	w.logger.Info("starting deployment workflow",
		"execution_id", ctx.ExecutionID(),
		"version_id", req.VersionID,
		"docker_image", req.DockerImage,
		"workspace_id", req.WorkspaceID,
		"project_id", req.ProjectID)

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

	// Step 2: Log version pending
	err = hydra.StepVoid(ctx, "log-version-pending", func(stepCtx context.Context) error {
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "pending",
			Message:      sql.NullString{String: "Version queued and ready to start", Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log version pending", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 3: Insert build into database
	err = hydra.StepVoid(ctx, "insert-build", func(stepCtx context.Context) error {
		w.logger.Info("inserting build into database", "build_id", buildID)
		insertErr := db.Query.InsertBuild(stepCtx, w.db.RW(), db.InsertBuildParams{
			ID:          buildID,
			WorkspaceID: req.WorkspaceID,
			ProjectID:   req.ProjectID,
			VersionID:   req.VersionID,
			CreatedAt:   time.Now().UnixMilli(),
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
		w.logger.Info("updating version status to building", "version_id", req.VersionID)
		updateErr := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
			ID:     req.VersionID,
			Status: db.VersionsStatusBuilding,
			Now:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		w.logger.Info("version status updated to building", "version_id", req.VersionID)
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to initialize build", "error", err, "version_id", req.VersionID)
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
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "downloading_docker_image",
			Message:      sql.NullString{String: fmt.Sprintf("Downloading Docker image: %s", req.DockerImage), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log downloading Docker image", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 7: Create VM (network call to metald)
	createResult, err := hydra.Step(ctx, "metald-create-vm", func(stepCtx context.Context) (*vmprovisionerv1.CreateVmResponse, error) {
		w.logger.Info("creating VM for deployment", "version_id", req.VersionID, "docker_image", req.DockerImage, "workspace_id", req.WorkspaceID, "project_id", req.ProjectID)

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
				"version_id":    req.VersionID,
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
		w.logger.Error("VM creation failed", "error", err, "version_id", req.VersionID)
		return err
	}

	w.logger.Info("VM creation completed", "vm_id", createResult.VmId, "state", createResult.State.String())

	// Step 8: Log building rootfs
	err = hydra.StepVoid(ctx, "log-building-rootfs", func(stepCtx context.Context) error {
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "building_rootfs",
			Message:      sql.NullString{String: fmt.Sprintf("Building rootfs from Docker image: %s", req.DockerImage), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log building rootfs", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 9: Log uploading rootfs
	err = hydra.StepVoid(ctx, "log-uploading-rootfs", func(stepCtx context.Context) error {
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "uploading_rootfs",
			Message:      sql.NullString{String: "Uploading rootfs image to storage", Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log uploading rootfs", "error", err, "version_id", req.VersionID)
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
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "creating_vm",
			Message:      sql.NullString{String: fmt.Sprintf("Creating VM for version: %s", req.VersionID), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log creating VM", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 12: Update version status to deploying
	_, err = hydra.Step(ctx, "update-version-deploying", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("starting deployment", "version_id", req.VersionID)

		deployingErr := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
			ID:     req.VersionID,
			Status: db.VersionsStatusDeploying,
			Now:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if deployingErr != nil {
			return nil, fmt.Errorf("failed to update version status to deploying: %w", deployingErr)
		}
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to update version status to deploying", "error", err, "version_id", req.VersionID)
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

	// Step 16: Log booting VM
	err = hydra.StepVoid(ctx, "log-booting-vm", func(stepCtx context.Context) error {
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "booting_vm",
			Message:      sql.NullString{String: fmt.Sprintf("VM booted successfully: %s", createResult.VmId), Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log booting VM", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 17: Assign domains (create route entries)
	assignedHostnames, err := hydra.Step(ctx, "assign-domains", func(stepCtx context.Context) ([]string, error) {
		w.logger.Info("assigning domains to version", "version_id", req.VersionID)

		var hostnames []string

		// Generate primary hostname for this deployment
		// Use Git info for hostname generation
		gitInfo := git.GetInfo()
		branch := "main"            // Default branch
		identifier := req.VersionID // Use full version ID as identifier

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
			ID:          routeID,
			WorkspaceID: req.WorkspaceID,
			ProjectID:   req.ProjectID,
			Hostname:    primaryHostname,
			VersionID:   req.VersionID,
			IsEnabled:   true,
			CreatedAt:   time.Now().UnixMilli(),
			UpdatedAt:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if insertErr != nil {
			w.logger.Error("failed to create route", "error", insertErr, "hostname", primaryHostname, "version_id", req.VersionID)
			return nil, fmt.Errorf("failed to create route for hostname %s: %w", primaryHostname, insertErr)
		}

		hostnames = append(hostnames, primaryHostname)
		w.logger.Info("primary domain assigned successfully", "hostname", primaryHostname, "version_id", req.VersionID, "route_id", routeID)

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
					ID:          localhostRouteID,
					WorkspaceID: req.WorkspaceID,
					ProjectID:   req.ProjectID,
					Hostname:    localhostHostname,
					VersionID:   req.VersionID,
					IsEnabled:   true,
					CreatedAt:   time.Now().UnixMilli(),
					UpdatedAt:   sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
				})
				if insertErr != nil {
					w.logger.Error("failed to create localhost route", "error", insertErr, "hostname", localhostHostname, "version_id", req.VersionID)
					return nil, fmt.Errorf("failed to create route for hostname %s: %w", localhostHostname, insertErr)
				}

				hostnames = append(hostnames, localhostHostname)
				w.logger.Info("localhost domain assigned successfully", "hostname", localhostHostname, "version_id", req.VersionID, "route_id", localhostRouteID, "container_port", portMapping.ContainerPort, "host_port", portMapping.HostPort)
			}
		}

		return hostnames, nil
	})
	if err != nil {
		w.logger.Error("domain assignment failed", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 18: Log assigning domains
	err = hydra.StepVoid(ctx, "log-assigning-domains", func(stepCtx context.Context) error {
		var message string
		if len(assignedHostnames) > 0 {
			message = fmt.Sprintf("Assigned hostnames: %s", strings.Join(assignedHostnames, ", "))
		} else {
			message = "Domain assignment completed"
		}
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "assigning_domains",
			Message:      sql.NullString{String: message, Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log assigning domains", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 19: Update version status to active
	_, err = hydra.Step(ctx, "update-version-active", func(stepCtx context.Context) (*DeploymentResult, error) {
		completionTime := time.Now().UnixMilli()
		w.logger.Info("updating version status to active", "version_id", req.VersionID, "completion_time", completionTime)
		activeErr := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
			ID:     req.VersionID,
			Status: db.VersionsStatusActive,
			Now:    sql.NullInt64{Valid: true, Int64: completionTime},
		})
		if activeErr != nil {
			w.logger.Error("failed to update version status to active", "error", activeErr, "version_id", req.VersionID)
			return nil, fmt.Errorf("failed to update version status to active: %w", activeErr)
		}

		w.logger.Info("deployment complete", "version_id", req.VersionID, "status", "active")

		return &DeploymentResult{
			VersionID: req.VersionID,
			Status:    "active",
		}, nil
	})
	if err != nil {
		w.logger.Error("deployment failed", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 20: Log completed
	err = hydra.StepVoid(ctx, "log-completed", func(stepCtx context.Context) error {
		return db.Query.InsertVersionStep(stepCtx, w.db.RW(), db.InsertVersionStepParams{
			VersionID:    req.VersionID,
			Status:       "completed",
			Message:      sql.NullString{String: "Version deployment completed successfully", Valid: true},
			ErrorMessage: sql.NullString{String: "", Valid: false},
			CreatedAt:    time.Now().UnixMilli(),
		})
	})
	if err != nil {
		w.logger.Error("failed to log completed", "error", err, "version_id", req.VersionID)
		return err
	}

	w.logger.Info("deployment workflow stage completed successfully", "version_id", req.VersionID, "vm_id", createResult.VmId)

	w.logger.Info("deployment workflow completed",
		"execution_id", ctx.ExecutionID(),
		"build_id", buildID,
		"version_id", req.VersionID,
		"status", "succeeded",
		"workspace_id", req.WorkspaceID,
		"project_id", req.ProjectID,
		"docker_image", req.DockerImage)

	return nil
}
