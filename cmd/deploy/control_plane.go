package deploy

import (
	"context"
	"fmt"
	"time"

	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/unkey/pkg/git"
	"github.com/unkeyed/unkey/pkg/otel/logging"
)

// DeploymentStatusEvent represents a status change event
type DeploymentStatusEvent struct {
	DeploymentID   string
	PreviousStatus components.Status
	CurrentStatus  components.Status
	Deployment     *components.V2DeployGetDeploymentResponseData
}

// DeploymentStepEvent represents a step update event
type DeploymentStepEvent struct {
	DeploymentID string
	Step         *components.V2DeployDeploymentStep
	Status       components.Status
}

// ControlPlaneClient handles API operations with the control plane
type ControlPlaneClient struct {
	sdk  *unkey.Unkey
	opts DeployOptions
}

// NewControlPlaneClient creates a new control plane client
func NewControlPlaneClient(opts DeployOptions) *ControlPlaneClient {
	sdkOpts := []unkey.SDKOption{
		unkey.WithSecurity(opts.RootKey),
	}

	// If not specified, SDK will use its default prod URL.
	// This is needed for easier local testing.
	if opts.APIBaseURL != "" {
		sdkOpts = append(sdkOpts, unkey.WithServerURL(opts.APIBaseURL))
	}

	sdk := unkey.New(sdkOpts...)

	return &ControlPlaneClient{
		sdk:  sdk,
		opts: opts,
	}
}

// CreateDeployment creates a new deployment in the control plane using a pre-built docker image
func (c *ControlPlaneClient) CreateDeployment(ctx context.Context, dockerImage string) (string, error) {
	commitInfo := git.GetInfo()

	var keyspaceID *string
	if c.opts.KeyspaceID != "" {
		keyspaceID = &c.opts.KeyspaceID
	}

	var gitCommit *components.V2DeployGitCommit
	if commitInfo.CommitSHA != "" {
		gitCommit = &components.V2DeployGitCommit{
			CommitSha:       &commitInfo.CommitSHA,
			CommitMessage:   &commitInfo.Message,
			AuthorHandle:    &commitInfo.AuthorHandle,
			AuthorAvatarURL: &commitInfo.AuthorAvatarURL,
			Timestamp:       &commitInfo.CommitTimestamp,
		}
	}

	reqBody := components.V2DeployCreateDeploymentRequestBody{
		ProjectID:       c.opts.ProjectID,
		Branch:          c.opts.Branch,
		EnvironmentSlug: c.opts.Environment,
		DockerImage:     dockerImage,
		KeyspaceID:      keyspaceID,
		GitCommit:       gitCommit,
	}

	res, err := c.sdk.Internal.CreateDeployment(ctx, reqBody)
	if err != nil {
		return "", err
	}

	if res.V2DeployCreateDeploymentResponseBody == nil {
		return "", fmt.Errorf("empty response from create deployment")
	}

	deploymentID := res.V2DeployCreateDeploymentResponseBody.Data.DeploymentID
	if deploymentID == "" {
		return "", fmt.Errorf("empty deployment ID returned from API")
	}

	return deploymentID, nil
}

// GetDeployment retrieves deployment information from the control plane
func (c *ControlPlaneClient) GetDeployment(ctx context.Context, deploymentID string) (*components.V2DeployGetDeploymentResponseData, error) {
	res, err := c.sdk.Internal.GetDeployment(ctx, components.V2DeployGetDeploymentRequestBody{
		DeploymentID: deploymentID,
	})
	if err != nil {
		return nil, err
	}

	if res.V2DeployGetDeploymentResponseBody == nil {
		return nil, fmt.Errorf("empty response from get deployment")
	}

	return &res.V2DeployGetDeploymentResponseBody.Data, nil
}

// PollDeploymentStatus polls for deployment changes and calls event handlers
func (c *ControlPlaneClient) PollDeploymentStatus(
	ctx context.Context,
	logger logging.Logger,
	deploymentID string,
	onStatusChange func(DeploymentStatusEvent) error,
) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(300 * time.Second)
	defer timeout.Stop()

	// Track processed steps by creation time to avoid duplicates
	lastStatus := components.StatusUnspecified

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("deployment timeout after 5 minutes")
		case <-ticker.C:
			deployment, err := c.GetDeployment(ctx, deploymentID)
			if err != nil {
				logger.Debug("Failed to get deployment status",
					"error", err,
					"deployment_id", deploymentID)
				continue
			}

			currentStatus := deployment.GetStatus()

			if currentStatus != lastStatus {
				event := DeploymentStatusEvent{
					DeploymentID:   deploymentID,
					PreviousStatus: lastStatus,
					CurrentStatus:  currentStatus,
					Deployment:     deployment,
				}

				if err := onStatusChange(event); err != nil {
					return err
				}
				lastStatus = currentStatus
			}

			// Check for completion
			if currentStatus == components.StatusReady {
				return nil
			}
		}
	}
}

// getFailureMessage extracts failure message from deployment
func (c *ControlPlaneClient) getFailureMessage(deployment *components.V2DeployGetDeploymentResponseData) string {
	if deployment.GetErrorMessage() != nil && *deployment.GetErrorMessage() != "" {
		return *deployment.GetErrorMessage()
	}

	for _, step := range deployment.GetSteps() {
		if step.GetErrorMessage() != nil && *step.GetErrorMessage() != "" {
			return *step.GetErrorMessage()
		}
	}

	return "Unknown deployment error"
}
