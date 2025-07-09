package version

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/pkg/db"
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

	// Step 4: Insert build into database
	_, err = hydra.Step(ctx, "insert-build", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("inserting build into database", "build_id", buildID)
		insertErr := db.Query.InsertBuild(stepCtx, w.db.RW(), db.InsertBuildParams{
			ID:          buildID,
			WorkspaceID: req.WorkspaceID,
			ProjectID:   req.ProjectID,
			VersionID:   req.VersionID,
			CreatedAt:   time.Now().UnixMilli(),
		})
		if insertErr != nil {
			return nil, fmt.Errorf("failed to create build record: %w", insertErr)
		}
		w.logger.Info("build record created successfully", "build_id", buildID)
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to insert build", "error", err, "build_id", buildID)
		return err
	}

	// Step 5: Update version status to building
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

	// Step 6: Create VM with Docker image (metald handles build internally)
	w.logger.Info("creating VM with Docker image - metald will handle build internally",
		"build_id", buildID,
		"docker_image", req.DockerImage,
		"version_id", req.VersionID)

	// Create VM with Docker image - metald handles build internally
	createResult, err := hydra.Step(ctx, "create-vm", func(stepCtx context.Context) (*vmprovisionerv1.CreateVmResponse, error) {
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

		w.logger.Info("creating VM for deployment", "version_id", req.VersionID, "docker_image", req.DockerImage, "workspace_id", req.WorkspaceID, "project_id", req.ProjectID)

		// Hardcoded VM configuration (TemplateStandard + ForDockerImage):
		vmConfig := &vmprovisionerv1.VmConfig{
			Cpu: &vmprovisionerv1.CpuConfig{
				VcpuCount:    2,
				MaxVcpuCount: 4,
			},
			Memory: &vmprovisionerv1.MemoryConfig{
				SizeBytes:      2 * 1024 * 1024 * 1024, // 2GB
				MaxSizeBytes:   8 * 1024 * 1024 * 1024, // 8GB
				HotplugEnabled: true,
			},
			Boot: &vmprovisionerv1.BootConfig{
				KernelPath: "/opt/vm-assets/vmlinux",
				KernelArgs: "console=ttyS0 reboot=k panic=1 pci=off",
			},
			Storage: []*vmprovisionerv1.StorageDevice{{
				Id:            "rootfs",
				Path:          "/opt/vm-assets/rootfs.ext4",
				ReadOnly:      false,
				IsRootDevice:  true,
				InterfaceType: "virtio-blk",
				Options: map[string]string{
					"docker_image": req.DockerImage,
					"auto_build":   "true",
				},
			}},
			Network: []*vmprovisionerv1.NetworkInterface{{
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
			}},
			Console: &vmprovisionerv1.ConsoleConfig{
				Enabled:     true,
				Output:      "/tmp/standard-vm-console.log",
				ConsoleType: "serial",
			},
			Metadata: map[string]string{
				"template":     "standard",
				"purpose":      "general",
				"docker_image": req.DockerImage,
				"runtime":      "docker",
				"version_id":   req.VersionID,
				"workspace_id": req.WorkspaceID,
				"project_id":   req.ProjectID,
				"created_by":   "deploy-workflow",
			},
		}

		w.logger.Info("sending VM creation request to metald", "docker_image", req.DockerImage)
		resp, createErr := w.metaldClient.CreateVm(stepCtx, connect.NewRequest(&vmprovisionerv1.CreateVmRequest{
			Config: vmConfig,
		}))
		if createErr != nil {
			w.logger.Error("metald VM creation request failed", "error", createErr, "docker_image", req.DockerImage)
			return nil, fmt.Errorf("failed to create VM: %w", createErr)
		}

		w.logger.Info("VM created successfully", "vm_id", resp.Msg.VmId, "state", resp.Msg.State.String(), "docker_image", req.DockerImage)

		// Update build status to succeeded since VM creation includes the build
		w.logger.Info("updating build status to succeeded", "build_id", buildID)
		successErr := db.Query.UpdateBuildSucceeded(stepCtx, w.db.RW(), db.UpdateBuildSucceededParams{
			ID:  buildID,
			Now: sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if successErr != nil {
			return nil, fmt.Errorf("failed to update build status to succeeded: %w", successErr)
		}
		w.logger.Info("build status updated to succeeded", "build_id", buildID)

		return resp.Msg, nil
	})
	if err != nil {
		w.logger.Error("VM creation failed", "error", err, "version_id", req.VersionID)
		return err
	}

	w.logger.Info("VM creation completed", "vm_id", createResult.VmId, "state", createResult.State.String())

	// Update version status to deploying (after successful VM creation)
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

	// Check VM readiness (max 30 attempts = 30 seconds)
	w.logger.Info("starting VM status polling", "vm_id", createResult.VmId, "max_attempts", 30)

	_, err = hydra.Step(ctx, "poll-vm-status", func(stepCtx context.Context) (*struct{}, error) {
		for attempt := 1; attempt <= 30; attempt++ {
			w.logger.Info("checking VM status", "vm_id", createResult.VmId, "attempt", attempt)

			w.logger.Info("sending VM status request to metald", "vm_id", createResult.VmId)
			resp, getErr := w.metaldClient.GetVmInfo(stepCtx, connect.NewRequest(&vmprovisionerv1.GetVmInfoRequest{
				VmId: createResult.VmId,
			}))
			if getErr != nil {
				w.logger.Error("metald VM status request failed", "error", getErr, "vm_id", createResult.VmId, "attempt", attempt)
				return nil, fmt.Errorf("failed to get VM info: %w", getErr)
			}

			w.logger.Info("VM status check", "vm_id", createResult.VmId, "state", resp.Msg.State.String(), "attempt", attempt)

			// Check if VM is ready for boot
			if resp.Msg.State == vmprovisionerv1.VmState_VM_STATE_CREATED ||
				resp.Msg.State == vmprovisionerv1.VmState_VM_STATE_RUNNING {
				w.logger.Info("VM is ready", "vm_id", createResult.VmId, "state", resp.Msg.State.String())
				return &struct{}{}, nil
			}

			// Sleep before next attempt (except on last attempt)
			if attempt < 30 {
				w.logger.Info("VM not ready yet, sleeping before next check", "vm_id", createResult.VmId, "state", resp.Msg.State.String(), "attempt", attempt, "sleep_duration", "1s")
				time.Sleep(1 * time.Second)
			}
		}

		// If we reach here, we exceeded max attempts
		return nil, fmt.Errorf("VM polling timed out after 30 attempts (30 seconds)")
	})
	if err != nil {
		w.logger.Error("VM status polling failed", "error", err, "vm_id", createResult.VmId)
		return err
	}

	// Boot VM
	_, err = hydra.Step(ctx, "boot-vm", func(stepCtx context.Context) (*vmprovisionerv1.BootVmResponse, error) {
		w.logger.Info("booting VM", "vm_id", createResult.VmId)

		w.logger.Info("sending VM boot request to metald", "vm_id", createResult.VmId)
		resp, bootErr := w.metaldClient.BootVm(stepCtx, connect.NewRequest(&vmprovisionerv1.BootVmRequest{
			VmId: createResult.VmId,
		}))
		if bootErr != nil {
			w.logger.Error("metald VM boot request failed", "error", bootErr, "vm_id", createResult.VmId)
			return nil, fmt.Errorf("failed to boot VM: %w", bootErr)
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

	// Generate completion timestamp
	completionTime, err := hydra.Step(ctx, "generate-completion-timestamp", func(stepCtx context.Context) (int64, error) {
		timestamp := time.Now().UnixMilli()
		w.logger.Info("generated completion timestamp", "timestamp", timestamp)
		return timestamp, nil
	})
	if err != nil {
		w.logger.Error("failed to generate completion timestamp", "error", err)
		return err
	}

	// Update version status to active
	_, err = hydra.Step(ctx, "update-version-active", func(stepCtx context.Context) (*DeploymentResult, error) {
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
