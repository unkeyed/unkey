package deploy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/cmd/cli/progress"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// pollVersionStatus polls the control plane API and displays deployment steps as they occur
func pollVersionStatus(ctx context.Context, logger logging.Logger, client ctrlv1connect.VersionServiceClient, authToken, versionId string, tracker *progress.Tracker) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(300 * time.Second)
	defer timeout.Stop()

	// Track processed steps by creation time to avoid duplicates
	processedSteps := make(map[int64]bool)
	lastStatus := ctrlv1.VersionStatus_VERSION_STATUS_UNSPECIFIED

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			tracker.FailStep("activate", "deployment timeout after 5 minutes")
			return fmt.Errorf("deployment timeout")
		case <-ticker.C:
			getReq := connect.NewRequest(&ctrlv1.GetVersionRequest{
				VersionId: versionId,
			})
			getReq.Header().Set("Authorization", "Bearer "+authToken)
			getResp, err := client.GetVersion(ctx, getReq)
			if err != nil {
				logger.Debug("Failed to get version status", "error", err, "version_id", versionId)
				continue
			}

			version := getResp.Msg.GetVersion()
			currentStatus := version.GetStatus()

			// Handle version status changes
			if currentStatus != lastStatus {
				if err := handleStatusTransition(tracker, lastStatus, currentStatus, version); err != nil {
					return err
				}
				lastStatus = currentStatus
			}

			// Process new step updates
			if err := processNewSteps(tracker, version.GetSteps(), processedSteps, currentStatus); err != nil {
				return err
			}

			// Check for completion
			if currentStatus == ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE {
				return nil
			}
		}
	}
}

func handleStatusTransition(tracker *progress.Tracker, lastStatus, currentStatus ctrlv1.VersionStatus, version *ctrlv1.Version) error {
	switch currentStatus {
	case ctrlv1.VersionStatus_VERSION_STATUS_PENDING:
		tracker.UpdateStep("deploy", "Version queued and ready to start")

	case ctrlv1.VersionStatus_VERSION_STATUS_BUILDING:
		tracker.UpdateStep("deploy", "Building deployment image")

	case ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING:
		tracker.CompleteStep("deploy", "Deployment initiated")
		tracker.StartStep("activate", "Deploying to unkey")

	case ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE:
		if lastStatus == ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING {
			tracker.CompleteStep("activate", "Version is now active")
		} else {
			tracker.CompleteStep("deploy", "Deployment completed")
			tracker.CompleteStep("activate", "Version is now active")
		}

	case ctrlv1.VersionStatus_VERSION_STATUS_FAILED:
		errorMsg := getFailureMessage(version)
		if lastStatus == ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING {
			tracker.FailStep("activate", errorMsg)
		} else {
			tracker.FailStep("deploy", errorMsg)
		}
		return fmt.Errorf("deployment failed: %s", errorMsg)
	}
	return nil
}

func processNewSteps(tracker *progress.Tracker, steps []*ctrlv1.VersionStep, processedSteps map[int64]bool, currentStatus ctrlv1.VersionStatus) error {
	for _, step := range steps {
		// Creation timestamp as unique identifier
		stepTimestamp := step.GetCreatedAt()

		if processedSteps[stepTimestamp] {
			continue // Already processed this step
		}

		// Handle step errors first
		if step.GetErrorMessage() != "" {
			errorMsg := step.GetErrorMessage()
			if currentStatus == ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING {
				tracker.FailStep("activate", errorMsg)
			} else {
				tracker.FailStep("deploy", errorMsg)
			}
			return fmt.Errorf("deployment failed: %s", errorMsg)
		}

		// Show step updates to user
		if step.GetMessage() != "" {
			message := step.GetMessage()

			// Add status context if helpful
			if step.GetStatus() != "" {
				message = fmt.Sprintf("[%s] %s", step.GetStatus(), message)
			}

			switch currentStatus {
			case ctrlv1.VersionStatus_VERSION_STATUS_BUILDING:
				tracker.UpdateStep("deploy", message)
			case ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING:
				tracker.UpdateStep("activate", message)
			default:
				// For other statuses, show on deploy step
				tracker.UpdateStep("deploy", message)
			}
		}

		// Mark this step as processed
		processedSteps[stepTimestamp] = true
	}
	return nil
}
