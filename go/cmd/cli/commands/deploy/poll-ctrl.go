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

	processedSteps := make(map[string]bool)
	lastStatus := ctrlv1.VersionStatus_VERSION_STATUS_UNSPECIFIED
	deployStepStarted := false

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			tracker.FailStep("activate", "Deployment timeout after 5 minutes")
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
				switch currentStatus {
				case ctrlv1.VersionStatus_VERSION_STATUS_PENDING:
					tracker.UpdateStep("deploy", "Version queued and ready to start")

				case ctrlv1.VersionStatus_VERSION_STATUS_BUILDING:
					tracker.UpdateStep("deploy", "Building deployment image")

				case ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING:
					if !deployStepStarted {
						tracker.CompleteStep("deploy", "Deployment initiated")
						tracker.StartStep("activate", "Deploying to unkey")
						deployStepStarted = true
					} else {
						tracker.UpdateStep("activate", "Deploying to unkey")
					}

				case ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE:
					if deployStepStarted {
						tracker.CompleteStep("activate", "Version is now active")
					} else {
						tracker.CompleteStep("deploy", "Deployment completed")
						tracker.CompleteStep("activate", "Version is now active")
					}

					// Give the animation loop time to render the completed state
					// TODO: Improve this later
					select {
					case <-time.After(200 * time.Millisecond):
					case <-ctx.Done():
						return ctx.Err()
					}

					return nil

				case ctrlv1.VersionStatus_VERSION_STATUS_FAILED:
					errorMsg := getFailureMessage(version)
					if deployStepStarted {
						tracker.FailStep("activate", errorMsg)
					} else {
						tracker.FailStep("deploy", errorMsg)
					}
					return fmt.Errorf("deployment failed: %s", errorMsg)
				}
				lastStatus = currentStatus
			}

			// Process deployment steps for additional detail
			steps := version.GetSteps()
			for _, step := range steps {
				stepKey := step.GetStatus()
				if !processedSteps[stepKey] && step.GetMessage() != "" {
					// Update the current step with detailed messages
					switch currentStatus {
					case ctrlv1.VersionStatus_VERSION_STATUS_BUILDING:
						tracker.UpdateStep("deploy", step.GetMessage())
					case ctrlv1.VersionStatus_VERSION_STATUS_DEPLOYING:
						if deployStepStarted {
							tracker.UpdateStep("activate", step.GetMessage())
						} else {
							tracker.UpdateStep("deploy", step.GetMessage())
						}
					}

					// Handle step errors
					if step.GetErrorMessage() != "" {
						errorMsg := step.GetErrorMessage()
						if deployStepStarted {
							tracker.FailStep("activate", errorMsg)
						} else {
							tracker.FailStep("deploy", errorMsg)
						}
						return fmt.Errorf("deployment failed: %s", errorMsg)
					}

					processedSteps[stepKey] = true
				}
			}
		}
	}
}
