package deploy

import (
	"context"
	"fmt"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// pollVersionStatus polls the control plane API and displays deployment steps as they occur
func pollVersionStatus(ctx context.Context, logger logging.Logger, client ctrlv1connect.VersionServiceClient, authToken, versionId string) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(300 * time.Second) // 5 minute timeout for full deployment
	defer timeout.Stop()

	displayedSteps := make(map[string]bool)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			fmt.Printf("Error: Deployment timeout after 5 minutes\n")
			return fmt.Errorf("deployment timeout")
		case <-ticker.C:
			// Poll version status
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

			// Display version steps in real-time
			steps := version.GetSteps()
			for _, step := range steps {
				stepKey := step.GetStatus()
				if !displayedSteps[stepKey] {
					displayVersionStep(step)
					displayedSteps[stepKey] = true
				}
			}

			// Check if deployment is complete
			if version.GetStatus() == ctrlv1.VersionStatus_VERSION_STATUS_ACTIVE {
				return nil
			}

			// Check if deployment failed
			if version.GetStatus() == ctrlv1.VersionStatus_VERSION_STATUS_FAILED {
				return fmt.Errorf("deployment failed")
			}
		}
	}
}

// displayVersionStep shows a version step with appropriate formatting
func displayVersionStep(step *ctrlv1.VersionStep) {
	message := step.GetMessage()
	// Display only the actual message from the database, indented under "Creating Version"
	if message != "" {
		fmt.Printf("  %s\n", message)
	}

	// Show error message if present
	if step.GetErrorMessage() != "" {
		fmt.Printf("  Error: %s\n", step.GetErrorMessage())
	}
}
