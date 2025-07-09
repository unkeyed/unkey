package version

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"connectrpc.com/connect"
	vmprovisionerv1 "github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1"
	"github.com/unkeyed/unkey/go/gen/proto/metal/vmprovisioner/v1/vmprovisionerv1connect"
	"github.com/unkeyed/unkey/go/pkg/builder"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/hydra"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

// DeployWorkflow orchestrates the complete build and deployment process using Hydra
type DeployWorkflow struct {
	db             db.Database
	logger         logging.Logger
	builderService builder.Service
	metaldClient   vmprovisionerv1connect.VmServiceClient
}

// NewDeployWorkflow creates a new deploy workflow instance
func NewDeployWorkflow(database db.Database, logger logging.Logger, builderService builder.Service, metaldClient vmprovisionerv1connect.VmServiceClient) *DeployWorkflow {
	return &DeployWorkflow{
		db:             database,
		logger:         logger,
		builderService: builderService,
		metaldClient:   metaldClient,
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
		"docker_image", req.DockerImage)

	// Step 1: Generate build ID
	buildID, err := hydra.Step(ctx, "generate-build-id", func(stepCtx context.Context) (string, error) {
		return uid.New(uid.BuildPrefix), nil
	})
	if err != nil {
		w.logger.Error("failed to generate build ID", "error", err)
		return err
	}

	// Step 4: Insert build into database
	_, err = hydra.Step(ctx, "insert-build", func(stepCtx context.Context) (*struct{}, error) {
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
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to insert build", "error", err, "build_id", buildID)
		return err
	}

	// Step 5: Update version status to building
	_, err = hydra.Step(ctx, "update-version-building", func(stepCtx context.Context) (*struct{}, error) {
		updateErr := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
			ID:     req.VersionID,
			Status: db.VersionsStatusBuilding,
			Now:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if updateErr != nil {
			return nil, fmt.Errorf("failed to update version status to building: %w", updateErr)
		}
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to initialize build", "error", err, "version_id", req.VersionID)
		return err
	}

	// Step 6: Submit build to builder service
	_, err = hydra.Step(ctx, "submit-build", func(stepCtx context.Context) (*struct{}, error) {
		w.logger.Info("submitting build to builder service",
			"build_id", buildID,
			"docker_image", req.DockerImage)

		submitErr := w.builderService.SubmitBuild(stepCtx, buildID, req.DockerImage)
		if submitErr != nil {
			return nil, fmt.Errorf("failed to submit build to builder service: %w", submitErr)
		}

		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to submit build", "error", err, "build_id", buildID)
		return err
	}

	// Wait for build completion with polling (max 150 attempts = 5 minutes)
	var buildResult *BuildResult
	lastStatus := ""

	for attempt := 1; attempt <= 150; attempt++ {
		currentBuildStatus, err := hydra.Step(ctx, fmt.Sprintf("check-build-status-%d", attempt), func(stepCtx context.Context) (*builder.BuildInfo, error) {
			buildStatus, statusErr := w.builderService.GetBuildStatus(stepCtx, buildID)
			if statusErr != nil {
				return nil, fmt.Errorf("failed to get build status: %w", statusErr)
			}

			w.logger.Info("build status check", "build_id", buildID, "status", string(buildStatus.Status), "attempt", attempt)
			return buildStatus, nil
		})
		if err != nil {
			w.logger.Error("failed to check build status", "error", err, "build_id", buildID, "attempt", attempt)
			return err
		}

		currentStatus := string(currentBuildStatus.Status)

		// Skip database update if status hasn't changed
		if currentStatus == lastStatus {
			// Status unchanged, continue to build completion check
		} else {
			_, err = hydra.Step(ctx, fmt.Sprintf("update-build-status-%d", attempt), func(updateCtx context.Context) (*struct{}, error) {
				now := time.Now().UnixMilli()

				switch currentBuildStatus.Status {
				case builder.BuildStatusQueued:
					// Build is queued, no update needed yet

				case builder.BuildStatusRunning:
					runningErr := db.Query.UpdateBuildStatus(updateCtx, w.db.RW(), db.UpdateBuildStatusParams{
						ID:     buildID,
						Status: db.BuildsStatusRunning,
						Now:    sql.NullInt64{Valid: true, Int64: now},
					})
					if runningErr != nil {
						return nil, fmt.Errorf("failed to update build status to running: %w", runningErr)
					}

				case builder.BuildStatusSuccess:
					successErr := db.Query.UpdateBuildSucceeded(updateCtx, w.db.RW(), db.UpdateBuildSucceededParams{
						ID:  buildID,
						Now: sql.NullInt64{Valid: true, Int64: now},
					})
					if successErr != nil {
						return nil, fmt.Errorf("failed to update build status to succeeded: %w", successErr)
					}

				case builder.BuildStatusFailed:
					failedErr := db.Query.UpdateBuildFailed(updateCtx, w.db.RW(), db.UpdateBuildFailedParams{
						ID:           buildID,
						ErrorMessage: sql.NullString{String: currentBuildStatus.ErrorMsg, Valid: currentBuildStatus.ErrorMsg != ""},
						Now:          sql.NullInt64{Valid: true, Int64: now},
					})
					if failedErr != nil {
						return nil, fmt.Errorf("failed to update build status to failed: %w", failedErr)
					}

					// Also update version status to failed
					versionErr := db.Query.UpdateVersionStatus(updateCtx, w.db.RW(), db.UpdateVersionStatusParams{
						ID:     req.VersionID,
						Status: db.VersionsStatusFailed,
						Now:    sql.NullInt64{Valid: true, Int64: now},
					})
					if versionErr != nil {
						return nil, fmt.Errorf("failed to update version status to failed: %w", versionErr)
					}
				}

				return &struct{}{}, nil
			})
			if err != nil {
				w.logger.Error("failed to update build status", "error", err, "status", currentStatus, "attempt", attempt)
				return err
			}
			lastStatus = currentStatus
		}

		// Check if build is complete
		switch currentBuildStatus.Status {
		case builder.BuildStatusSuccess:
			buildResult = &BuildResult{
				BuildID:  buildID,
				Status:   "succeeded",
				ErrorMsg: "",
			}
			goto buildComplete

		case builder.BuildStatusFailed:
			buildResult = &BuildResult{
				BuildID:  buildID,
				Status:   "failed",
				ErrorMsg: currentBuildStatus.ErrorMsg,
			}
			goto buildComplete

		default:
			// Still building, sleep before next attempt
			err = hydra.Sleep(ctx, 2*time.Second)
			if err != nil {
				w.logger.Error("failed to sleep between build checks", "error", err, "attempt", attempt)
				return err
			}
		}
	}

	// If we reach here, we exceeded max attempts
	return fmt.Errorf("build polling timed out after 150 attempts (5 minutes)")

buildComplete:

	// Handle build failure
	if buildResult.Status == "failed" {
		w.logger.Error("build failed", "build_id", buildID, "error", buildResult.ErrorMsg)
		return fmt.Errorf("build failed: %s", buildResult.ErrorMsg)
	}

	// Deploy if build succeeded
	if buildResult.Status == "succeeded" {

		// Create VM first
		createResult, err := hydra.Step(ctx, "create-vm", func(stepCtx context.Context) (*vmprovisionerv1.CreateVmResponse, error) {
			w.logger.Info("creating VM for deployment", "version_id", req.VersionID, "docker_image", req.DockerImage)

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
					"template":      "standard",
					"purpose":       "general",
					"docker_image":  req.DockerImage,
					"runtime":       "docker",
					"version_id":    req.VersionID,
					"workspace_id":  req.WorkspaceID,
					"project_id":    req.ProjectID,
					"created_by":    "deploy-workflow",
				},
			}

			resp, createErr := w.metaldClient.CreateVm(stepCtx, connect.NewRequest(&vmprovisionerv1.CreateVmRequest{
				Config: vmConfig,
			}))
			if createErr != nil {
				return nil, fmt.Errorf("failed to create VM: %w", createErr)
			}

			w.logger.Info("VM created successfully", "vm_id", resp.Msg.VmId, "state", resp.Msg.State.String())
			return resp.Msg, nil
		})
		if err != nil {
			w.logger.Error("VM creation failed", "error", err, "version_id", req.VersionID)
			return err
		}

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
		for attempt := 1; attempt <= 30; attempt++ {
			vmInfo, err := hydra.Step(ctx, fmt.Sprintf("check-vm-status-%d", attempt), func(stepCtx context.Context) (*vmprovisionerv1.GetVmInfoResponse, error) {
				resp, getErr := w.metaldClient.GetVmInfo(stepCtx, connect.NewRequest(&vmprovisionerv1.GetVmInfoRequest{
					VmId: createResult.VmId,
				}))
				if getErr != nil {
					return nil, fmt.Errorf("failed to get VM info: %w", getErr)
				}

				w.logger.Info("VM status check", "vm_id", createResult.VmId, "state", resp.Msg.State.String(), "attempt", attempt)
				return resp.Msg, nil
			})
			if err != nil {
				w.logger.Error("failed to check VM status", "error", err, "vm_id", createResult.VmId, "attempt", attempt)
				return err
			}

			// Check if VM is ready for boot
			if vmInfo.State == vmprovisionerv1.VmState_VM_STATE_CREATED || 
			   vmInfo.State == vmprovisionerv1.VmState_VM_STATE_RUNNING {
				w.logger.Info("VM is ready", "vm_id", createResult.VmId, "state", vmInfo.State.String())
				goto vmReady
			}

			// Sleep before next attempt (except on last attempt)
			if attempt < 30 {
				err = hydra.Sleep(ctx, 1*time.Second)
				if err != nil {
					w.logger.Error("failed to sleep between VM checks", "error", err, "attempt", attempt)
					return err
				}
			}
		}

		// If we reach here, VM never became ready
		return fmt.Errorf("VM polling timed out after 30 attempts (30 seconds)")

vmReady:

		// Boot VM
		_, err = hydra.Step(ctx, "boot-vm", func(stepCtx context.Context) (*vmprovisionerv1.BootVmResponse, error) {
			w.logger.Info("booting VM", "vm_id", createResult.VmId)

			resp, bootErr := w.metaldClient.BootVm(stepCtx, connect.NewRequest(&vmprovisionerv1.BootVmRequest{
				VmId: createResult.VmId,
			}))
			if bootErr != nil {
				return nil, fmt.Errorf("failed to boot VM: %w", bootErr)
			}

			if !resp.Msg.Success {
				return nil, fmt.Errorf("VM boot was not successful, state: %s", resp.Msg.State.String())
			}

			w.logger.Info("VM booted successfully", "vm_id", createResult.VmId, "state", resp.Msg.State.String())
			return resp.Msg, nil
		})
		if err != nil {
			w.logger.Error("VM boot failed", "error", err, "vm_id", createResult.VmId)
			return err
		}

		// Generate completion timestamp
		completionTime, err := hydra.Step(ctx, "generate-completion-timestamp", func(stepCtx context.Context) (int64, error) {
			return time.Now().UnixMilli(), nil
		})
		if err != nil {
			w.logger.Error("failed to generate completion timestamp", "error", err)
			return err
		}

		// Update version status to active
		_, err = hydra.Step(ctx, "update-version-active", func(stepCtx context.Context) (*DeploymentResult, error) {
			activeErr := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
				ID:     req.VersionID,
				Status: db.VersionsStatusActive,
				Now:    sql.NullInt64{Valid: true, Int64: completionTime},
			})
			if activeErr != nil {
				return nil, fmt.Errorf("failed to update version status to active: %w", activeErr)
			}

			w.logger.Info("deployment complete", "version_id", req.VersionID)

			return &DeploymentResult{
				VersionID: req.VersionID,
				Status:    "active",
			}, nil
		})
		if err != nil {
			w.logger.Error("deployment failed", "error", err, "version_id", req.VersionID)
			return err
		}
	}

	w.logger.Info("deployment workflow completed",
		"execution_id", ctx.ExecutionID(),
		"build_id", buildID,
		"version_id", req.VersionID,
		"status", buildResult.Status)

	return nil
}
