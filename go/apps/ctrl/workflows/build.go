package workflows

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/unkeyed/unkey/go/pkg/builder"
	"github.com/unkeyed/unkey/go/pkg/db"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
	"github.com/unkeyed/unkey/go/pkg/uid"
)

type Build struct {
	Logger         logging.Logger
	DB             db.Database
	BuilderService builder.Service
}

type BuildRequest struct {
	WorkspaceID string
	ProjectID   string
	VersionID   string
	DockerImage string
}

func (wf Build) Run(ctx context.Context, req BuildRequest) error {
	now := time.Now().UnixMilli()

	buildID := uid.New(uid.BuildPrefix)

	// Insert build into database with version_id
	err := db.Query.InsertBuild(ctx, wf.DB.RW(), db.InsertBuildParams{
		ID:          buildID,
		WorkspaceID: req.WorkspaceID,
		ProjectID:   req.ProjectID,
		VersionID:   req.VersionID,
		CreatedAt:   now,
	})
	if err != nil {
		return err
	}

	wf.Logger.Info("submitting build to builder service", "build_id", buildID, "version_id", req.VersionID, "docker_image", req.DockerImage)

	// Submit build to builder service
	err = wf.BuilderService.SubmitBuild(ctx, buildID, req.DockerImage)
	if err != nil {
		wf.Logger.Error("failed to submit build to builder service", "build_id", buildID, "error", err)
		return err
	}

	// Update version status to building
	err = db.Query.UpdateVersionStatus(ctx, wf.DB.RW(), db.UpdateVersionStatusParams{
		ID:     req.VersionID,
		Status: db.VersionsStatusBuilding,
		Now:    sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		wf.Logger.Error("failed to update version status to building", "error", err)
	}

	// Start polling the builder service for updates
	err = wf.pollBuilderService(ctx, buildID, req.VersionID)
	if err != nil {
		wf.Logger.Error("build polling failed", "build_id", buildID, "error", err)
		return err
	}

	wf.Logger.Info("build and deployment complete", "build_id", buildID, "version_id", req.VersionID)
	return nil
}

// pollBuilderService polls the builder service for build status updates
func (wf Build) pollBuilderService(ctx context.Context, buildID, versionID string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	timeout := time.NewTimer(5 * time.Minute) // 5 minute timeout for builds
	defer timeout.Stop()

	lastStatus := ""

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			wf.Logger.Error("build timeout", "build_id", buildID, "timeout", "5m")
			// Mark build as failed due to timeout
			now := time.Now().UnixMilli()
			_ = db.Query.UpdateBuildFailed(ctx, wf.DB.RW(), db.UpdateBuildFailedParams{
				ID:           buildID,
				ErrorMessage: sql.NullString{String: "Build timeout after 5 minutes", Valid: true},
				Now:          sql.NullInt64{Valid: true, Int64: now},
			})
			return fmt.Errorf("build timeout after 5 minutes")
		case <-ticker.C:
			// Poll builder service for status
			buildInfo, err := wf.BuilderService.GetBuildStatus(ctx, buildID)
			if err != nil {
				wf.Logger.Error("failed to get build status from builder service", "build_id", buildID, "error", err)
				continue
			}

			currentStatus := string(buildInfo.Status)
			now := time.Now().UnixMilli()

			// Only update if status changed
			if currentStatus != lastStatus {
				wf.Logger.Info("build status update", "build_id", buildID, "status", currentStatus)
				lastStatus = currentStatus

				switch buildInfo.Status {
				case builder.BuildStatusQueued:
					// Build is queued, no database update needed (already pending)

				case builder.BuildStatusRunning:
					// Update build status to running
					err = db.Query.UpdateBuildStatus(ctx, wf.DB.RW(), db.UpdateBuildStatusParams{
						ID:     buildID,
						Status: db.BuildsStatusRunning,
						Now:    sql.NullInt64{Valid: true, Int64: now},
					})
					if err != nil {
						wf.Logger.Error("failed to update build status to running", "error", err)
					}

				case builder.BuildStatusSuccess:
					// Build succeeded
					err = db.Query.UpdateBuildSucceeded(ctx, wf.DB.RW(), db.UpdateBuildSucceededParams{
						ID:  buildID,
						Now: sql.NullInt64{Valid: true, Int64: now},
					})
					if err != nil {
						wf.Logger.Error("failed to update build status to succeeded", "error", err)
						return err
					}

					// Start deployment phase
					return wf.startDeployment(ctx, versionID)

				case builder.BuildStatusFailed:
					// Build failed
					err = db.Query.UpdateBuildFailed(ctx, wf.DB.RW(), db.UpdateBuildFailedParams{
						ID:           buildID,
						ErrorMessage: sql.NullString{String: buildInfo.ErrorMsg, Valid: buildInfo.ErrorMsg != ""},
						Now:          sql.NullInt64{Valid: true, Int64: now},
					})
					if err != nil {
						wf.Logger.Error("failed to update build status to failed", "error", err)
					}

					// Update version status to failed
					err = db.Query.UpdateVersionStatus(ctx, wf.DB.RW(), db.UpdateVersionStatusParams{
						ID:     versionID,
						Status: db.VersionsStatusFailed,
						Now:    sql.NullInt64{Valid: true, Int64: now},
					})
					if err != nil {
						wf.Logger.Error("failed to update version status to failed", "error", err)
					}

					return fmt.Errorf("build failed: %s", buildInfo.ErrorMsg)
				}
			}
		}
	}
}

// startDeployment handles the deployment phase after build success
func (wf Build) startDeployment(ctx context.Context, versionID string) error {
	now := time.Now().UnixMilli()

	wf.Logger.Info("starting deployment", "version_id", versionID)

	// Update version status to deploying
	err := db.Query.UpdateVersionStatus(ctx, wf.DB.RW(), db.UpdateVersionStatusParams{
		ID:     versionID,
		Status: db.VersionsStatusDeploying,
		Now:    sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		wf.Logger.Error("failed to update version status to deploying", "error", err)
		return err
	}

	// Simulate deployment process
	time.Sleep(3 * time.Second)

	// Update version status to active
	err = db.Query.UpdateVersionStatus(ctx, wf.DB.RW(), db.UpdateVersionStatusParams{
		ID:     versionID,
		Status: db.VersionsStatusActive,
		Now:    sql.NullInt64{Valid: true, Int64: now},
	})
	if err != nil {
		wf.Logger.Error("failed to update version status to active", "error", err)
		return err
	}

	wf.Logger.Info("deployment complete", "version_id", versionID)
	return nil
}
