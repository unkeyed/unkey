package deploy

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// DeploymentStatusEvent represents a status change event
type DeploymentStatusEvent struct {
	DeploymentID   string
	PreviousStatus ctrlv1.DeploymentStatus
	CurrentStatus  ctrlv1.DeploymentStatus
	Deployment     *ctrlv1.Deployment
}

// DeploymentStepEvent represents a step update event
type DeploymentStepEvent struct {
	DeploymentID string
	Step         *ctrlv1.DeploymentStep
	Status       ctrlv1.DeploymentStatus
}

// ControlPlaneClient handles API operations with the control plane
type ControlPlaneClient struct {
	client ctrlv1connect.DeploymentServiceClient
	opts   DeployOptions
}

// NewControlPlaneClient creates a new control plane client
func NewControlPlaneClient(opts DeployOptions) *ControlPlaneClient {
	httpClient := &http.Client{}
	client := ctrlv1connect.NewDeploymentServiceClient(httpClient, opts.ControlPlaneURL)

	return &ControlPlaneClient{
		client: client,
		opts:   opts,
	}
}

// CreateDeployment creates a new deployment in the control plane
func (c *ControlPlaneClient) CreateDeployment(ctx context.Context, dockerImage string) (string, error) {
	// Get git commit information
	commitInfo, err := git.GetCommitInfo()
	if err != nil {
		// Log warning but don't fail the deployment if git info is unavailable
		log.Printf("Warning: failed to get git commit info: %v", err)
		commitInfo = &git.CommitInfo{} // Use empty struct
	}

	createReq := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		WorkspaceId:              c.opts.WorkspaceID,
		ProjectId:                c.opts.ProjectID,
		KeyspaceId:               &c.opts.KeyspaceID,
		Branch:                   c.opts.Branch,
		SourceType:               ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
		EnvironmentSlug:          c.opts.Environment,
		DockerImage:              dockerImage,
		GitCommitSha:             commitInfo.SHA,
		GitCommitMessage:         commitInfo.Message,
		GitCommitAuthorHandle:    commitInfo.AuthorHandle,
		GitCommitAuthorAvatarUrl: commitInfo.AuthorAvatarURL,
		GitCommitTimestamp:       commitInfo.CommitTimestamp,
	})

	// Use API key for authentication if provided, fallback to auth token
	authHeader := c.opts.APIKey
	if authHeader == "" {
		authHeader = c.opts.AuthToken
	}
	createReq.Header().Set("Authorization", "Bearer "+authHeader)

	createResp, err := c.client.CreateDeployment(ctx, createReq)
	if err != nil {
		return "", c.handleCreateDeploymentError(err)
	}

	deploymentID := createResp.Msg.GetDeploymentId()
	if deploymentID == "" {
		return "", fmt.Errorf("empty deployment ID returned from control plane")
	}

	return deploymentID, nil
}

// GetDeployment retrieves deployment information from the control plane
func (c *ControlPlaneClient) GetDeployment(ctx context.Context, deploymentId string) (*ctrlv1.Deployment, error) {
	getReq := connect.NewRequest(&ctrlv1.GetDeploymentRequest{
		DeploymentId: deploymentId,
	})
	// Use API key for authentication if provided, fallback to auth token
	authHeader := c.opts.APIKey
	if authHeader == "" {
		authHeader = c.opts.AuthToken
	}
	getReq.Header().Set("Authorization", "Bearer "+authHeader)

	getResp, err := c.client.GetDeployment(ctx, getReq)
	if err != nil {
		return nil, err
	}

	return getResp.Msg.GetDeployment(), nil
}

// PollDeploymentStatus polls for deployment changes and calls event handlers
func (c *ControlPlaneClient) PollDeploymentStatus(
	ctx context.Context,
	logger logging.Logger,
	deploymentID string,
	onStatusChange func(DeploymentStatusEvent) error,
	onStepUpdate func(DeploymentStepEvent) error,
) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(300 * time.Second)
	defer timeout.Stop()

	// Track processed steps by creation time to avoid duplicates
	processedSteps := make(map[int64]bool)
	lastStatus := ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("deployment timeout after 5 minutes")
		case <-ticker.C:
			deployment, err := c.GetDeployment(ctx, deploymentID)
			if err != nil {
				logger.Debug("Failed to get deployment status", "error", err, "deployment_id", deploymentID)
				continue
			}

			currentStatus := deployment.GetStatus()

			// Handle deployment status changes
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

			// Process new step updates
			if err := c.processNewSteps(deploymentID, deployment.GetSteps(), processedSteps, currentStatus, onStepUpdate); err != nil {
				return err
			}

			// Check for completion
			if currentStatus == ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY {
				return nil
			}
		}
	}
}

// processNewSteps processes new deployment steps and calls the event handler
func (c *ControlPlaneClient) processNewSteps(
	deploymentID string,
	steps []*ctrlv1.DeploymentStep,
	processedSteps map[int64]bool,
	currentStatus ctrlv1.DeploymentStatus,
	onStepUpdate func(DeploymentStepEvent) error,
) error {
	for _, step := range steps {
		// Creation timestamp as unique identifier
		stepTimestamp := step.GetCreatedAt()

		if processedSteps[stepTimestamp] {
			continue // Already processed this step
		}

		// Handle step errors first
		if step.GetErrorMessage() != "" {
			return fmt.Errorf("deployment failed: %s", step.GetErrorMessage())
		}

		// Call step update handler
		if step.GetMessage() != "" {
			event := DeploymentStepEvent{
				DeploymentID: deploymentID,
				Step:         step,
				Status:       currentStatus,
			}
			if err := onStepUpdate(event); err != nil {
				return err
			}

			// INFO: This is for demo purposes only.
			// Adding a small delay between deployment steps to make the progression
			// visually observable during demos. This allows viewers to see each
			// individual step (VM boot, rootfs loading, etc.) rather than having
			// everything complete too quickly to follow.
			time.Sleep(800 * time.Millisecond)
		}
		// Mark this step as processed
		processedSteps[stepTimestamp] = true
	}
	return nil
}

// getFailureMessage extracts failure message from version
func (c *ControlPlaneClient) getFailureMessage(deployment *ctrlv1.Deployment) string {
	if deployment.GetErrorMessage() != "" {
		return deployment.GetErrorMessage()
	}

	// Check for error in steps
	for _, step := range deployment.GetSteps() {
		if step.GetErrorMessage() != "" {
			return step.GetErrorMessage()
		}
	}

	return "Unknown deployment error"
}

// handleCreateDeploymentError provides specific error handling for deployment creation
func (c *ControlPlaneClient) handleCreateDeploymentError(err error) error {
	// Check if it's a connection error
	if strings.Contains(err.Error(), "connection refused") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAppErrorsInternalServiceUnavailable),
			fault.Internal(fmt.Sprintf("Failed to connect to control plane at %s", c.opts.ControlPlaneURL)),
			fault.Public("Unable to connect to control plane. Is it running?"),
		)
	}

	// Check if it's an auth error
	if connectErr := new(connect.Error); errors.As(err, &connectErr) {
		if connectErr.Code() == connect.CodeUnauthenticated {
			// Determine which auth method was used for better error message
			authMethod := "API key"
			if c.opts.APIKey == "" {
				authMethod = "auth token"
			}
			return fault.Wrap(err,
				fault.Code(codes.UnkeyAuthErrorsAuthenticationMalformed),
				fault.Internal(fmt.Sprintf("Authentication failed with %s", authMethod)),
				fault.Public(fmt.Sprintf("Authentication failed. Check your %s.", authMethod)),
			)
		}
	}

	// Generic API error
	return fault.Wrap(err,
		fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
		fault.Internal(fmt.Sprintf("CreateDeployment API call failed: %v", err)),
		fault.Public("Failed to create deployment. Please try again."),
	)
}
