package version

import (
	"context"
	"database/sql"
	"fmt"
	"time"

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
}

// NewDeployWorkflow creates a new deploy workflow instance
func NewDeployWorkflow(database db.Database, logger logging.Logger, builderService builder.Service) *DeployWorkflow {
	return &DeployWorkflow{
		db:             database,
		logger:         logger,
		builderService: builderService,
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
		err := db.Query.InsertBuild(stepCtx, w.db.RW(), db.InsertBuildParams{
			ID:          buildID,
			WorkspaceID: req.WorkspaceID,
			ProjectID:   req.ProjectID,
			VersionID:   req.VersionID,
			CreatedAt:   time.Now().UnixMilli(),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to create build record: %w", err)
		}
		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to insert build", "error", err, "build_id", buildID)
		return err
	}

	// Step 5: Update version status to building
	_, err = hydra.Step(ctx, "update-version-building", func(stepCtx context.Context) (*struct{}, error) {
		err := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
			ID:     req.VersionID,
			Status: db.VersionsStatusBuilding,
			Now:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to update version status to building: %w", err)
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

		err := w.builderService.SubmitBuild(stepCtx, buildID, req.DockerImage)
		if err != nil {
			return nil, fmt.Errorf("failed to submit build to builder service: %w", err)
		}

		return &struct{}{}, nil
	})
	if err != nil {
		w.logger.Error("failed to submit build", "error", err, "build_id", buildID)
		return err
	}

	// Step 7: Wait for build completion with polling
	buildResult, err := hydra.Step(ctx, "wait-for-completion", func(stepCtx context.Context) (*BuildResult, error) {
		w.logger.Info("waiting for build completion", "build_id", buildID)

		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()

		lastStatus := ""

		for {
			select {
			case <-stepCtx.Done():
				return nil, stepCtx.Err()
			case <-ticker.C:
				buildStatus, err := w.builderService.GetBuildStatus(stepCtx, buildID)
				if err != nil {
					w.logger.Error("failed to get build status", "build_id", buildID, "error", err)
					continue
				}

				currentStatus := string(buildStatus.Status)

				// Only update if status changed
				if currentStatus == lastStatus {
					continue
				}

				w.logger.Info("build status update", "build_id", buildID, "status", currentStatus)
				lastStatus = currentStatus

				_, err = hydra.Step(ctx, "update-build-status", func(updateCtx context.Context) (*struct{}, error) {
					now := time.Now().UnixMilli()

					switch buildStatus.Status {
					case builder.BuildStatusRunning:
						err := db.Query.UpdateBuildStatus(updateCtx, w.db.RW(), db.UpdateBuildStatusParams{
							ID:     buildID,
							Status: db.BuildsStatusRunning,
							Now:    sql.NullInt64{Valid: true, Int64: now},
						})
						if err != nil {
							return nil, fmt.Errorf("failed to update build status to running: %w", err)
						}

					case builder.BuildStatusSuccess:
						err := db.Query.UpdateBuildSucceeded(updateCtx, w.db.RW(), db.UpdateBuildSucceededParams{
							ID:  buildID,
							Now: sql.NullInt64{Valid: true, Int64: now},
						})
						if err != nil {
							return nil, fmt.Errorf("failed to update build status to succeeded: %w", err)
						}

					case builder.BuildStatusFailed:
						err := db.Query.UpdateBuildFailed(updateCtx, w.db.RW(), db.UpdateBuildFailedParams{
							ID:           buildID,
							ErrorMessage: sql.NullString{String: buildStatus.ErrorMsg, Valid: buildStatus.ErrorMsg != ""},
							Now:          sql.NullInt64{Valid: true, Int64: now},
						})
						if err != nil {
							return nil, fmt.Errorf("failed to update build status to failed: %w", err)
						}

						// Also update version status to failed
						err = db.Query.UpdateVersionStatus(updateCtx, w.db.RW(), db.UpdateVersionStatusParams{
							ID:     req.VersionID,
							Status: db.VersionsStatusFailed,
							Now:    sql.NullInt64{Valid: true, Int64: now},
						})
						if err != nil {
							return nil, fmt.Errorf("failed to update version status to failed: %w", err)
						}
					}

					return &struct{}{}, nil
				})
				if err != nil {
					w.logger.Error("failed to update build status", "error", err, "status", currentStatus)
					if buildStatus.Status != builder.BuildStatusFailed {
						continue // For non-failed states, continue polling
					}
				}

				// Return appropriate result based on final status
				switch buildStatus.Status {
				case builder.BuildStatusSuccess:
					return &BuildResult{
						BuildID: buildID,
						Status:  "succeeded",
					}, nil

				case builder.BuildStatusFailed:
					return &BuildResult{
						BuildID:  buildID,
						Status:   "failed",
						ErrorMsg: buildStatus.ErrorMsg,
					}, fmt.Errorf("build failed: %s", buildStatus.ErrorMsg)
				}
			}
		}
	})
	if err != nil {
		w.logger.Error("build failed", "error", err, "build_id", buildID)
		return err
	}

	// Step 8: Deploy if build succeeded
	if buildResult.Status == "succeeded" {

		// Step 8b: Update version status to deploying
		_, err = hydra.Step(ctx, "update-version-deploying", func(stepCtx context.Context) (*struct{}, error) {
			w.logger.Info("starting deployment", "version_id", req.VersionID)

			err := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
				ID:     req.VersionID,
				Status: db.VersionsStatusDeploying,
				Now:    sql.NullInt64{Valid: true, Int64: time.Now().UnixMilli()},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update version status to deploying: %w", err)
			}
			return &struct{}{}, nil
		})
		if err != nil {
			w.logger.Error("failed to update version status to deploying", "error", err, "version_id", req.VersionID)
			return err
		}

		// Step 8c: Simulate deployment process
		_, err = hydra.Step(ctx, "simulate-deployment", func(stepCtx context.Context) (*struct{}, error) {
			// Simulate deployment process (in real implementation, this would orchestrate actual deployment)
			time.Sleep(3 * time.Second)
			return &struct{}{}, nil
		})
		if err != nil {
			w.logger.Error("deployment simulation failed", "error", err, "version_id", req.VersionID)
			return err
		}

		// Step 8d: Generate completion timestamp
		completionTime, err := hydra.Step(ctx, "generate-completion-timestamp", func(stepCtx context.Context) (int64, error) {
			return time.Now().UnixMilli(), nil
		})
		if err != nil {
			w.logger.Error("failed to generate completion timestamp", "error", err)
			return err
		}

		// Step 8e: Update version status to active
		_, err = hydra.Step(ctx, "update-version-active", func(stepCtx context.Context) (*DeploymentResult, error) {
			err := db.Query.UpdateVersionStatus(stepCtx, w.db.RW(), db.UpdateVersionStatusParams{
				ID:     req.VersionID,
				Status: db.VersionsStatusActive,
				Now:    sql.NullInt64{Valid: true, Int64: completionTime},
			})
			if err != nil {
				return nil, fmt.Errorf("failed to update version status to active: %w", err)
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
