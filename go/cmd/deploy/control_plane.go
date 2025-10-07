package deploy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
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
	deploymentClient ctrlv1connect.DeploymentServiceClient
	buildClient      ctrlv1connect.BuildServiceClient
	opts             DeployOptions
}

// NewControlPlaneClient creates a new control plane client
func NewControlPlaneClient(opts DeployOptions) *ControlPlaneClient {
	httpClient := &http.Client{}
	deploymentClient := ctrlv1connect.NewDeploymentServiceClient(httpClient, opts.ControlPlaneURL)
	buildClient := ctrlv1connect.NewBuildServiceClient(httpClient, opts.ControlPlaneURL)

	return &ControlPlaneClient{
		deploymentClient: deploymentClient,
		buildClient:      buildClient,
		opts:             opts,
	}
}

// CreateBuild builds and pushes a Docker image via the control plane
func (c *ControlPlaneClient) CreateBuild(ctx context.Context) (string, string, error) {
	// Create tar from context path
	tarPath, err := createContextTar(c.opts.Context)
	if err != nil {
		return "", "", fmt.Errorf("failed to create context tar: %w", err)
	}

	buildReq := connect.NewRequest(&ctrlv1.CreateBuildRequest{
		ImagePath:      c.opts.Dockerfile,
		ContextPath:    tarPath,
		UnkeyProjectID: c.opts.ProjectID,
	})
	authHeader := c.opts.APIKey
	if authHeader == "" {
		authHeader = c.opts.AuthToken
	}

	buildReq.Header().Set("Authorization", "Bearer "+authHeader)
	buildResp, err := c.buildClient.CreateBuild(ctx, buildReq)
	if err != nil {
		return "", "", c.handleBuildError(err)
	}
	imageName := buildResp.Msg.GetImageName()
	buildID := buildResp.Msg.GetBuildId()
	if imageName == "" {
		return "", "", fmt.Errorf("empty image name returned from control plane")
	}
	if buildID == "" {
		return "", "", fmt.Errorf("empty build ID returned from control plane")
	}
	return imageName, buildID, nil
}

// createContextTar creates a tar.gz from the given directory path
// and returns the absolute path to the created tar file
func createContextTar(contextPath string) (string, error) {
	// Validate context path exists
	info, err := os.Stat(contextPath)
	if err != nil {
		return "", fmt.Errorf("context path does not exist: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("context path must be a directory: %s", contextPath)
	}

	absContextPath, err := filepath.Abs(contextPath)
	if err != nil {
		return "", fmt.Errorf("failed to get absolute context path: %w", err)
	}

	// Create temp file in /tmp (mounted volume)
	tmpFile, err := os.CreateTemp("/tmp", "build-context-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFile.Close()
	tarPath := tmpFile.Name()

	cmd := exec.Command("tar", "-czf", tarPath, "-C", absContextPath, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tarPath)
		return "", fmt.Errorf("tar command failed: %w\nOutput: %s", err, string(output))
	}

	return tarPath, nil
}

// CreateDeployment creates a new deployment in the control plane
func (c *ControlPlaneClient) CreateDeployment(ctx context.Context, dockerImage string) (string, error) {
	commitInfo := git.GetInfo()
	createReq := connect.NewRequest(&ctrlv1.CreateDeploymentRequest{
		WorkspaceId:              c.opts.WorkspaceID,
		ProjectId:                c.opts.ProjectID,
		KeyspaceId:               &c.opts.KeyspaceID,
		Branch:                   c.opts.Branch,
		SourceType:               ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
		EnvironmentSlug:          c.opts.Environment,
		DockerImage:              dockerImage,
		GitCommitSha:             commitInfo.CommitSHA,
		GitCommitMessage:         commitInfo.Message,
		GitCommitAuthorHandle:    commitInfo.AuthorHandle,
		GitCommitAuthorAvatarUrl: commitInfo.AuthorAvatarURL,
		GitCommitTimestamp:       commitInfo.CommitTimestamp,
	})

	authHeader := c.opts.APIKey
	if authHeader == "" {
		authHeader = c.opts.AuthToken
	}
	createReq.Header().Set("Authorization", "Bearer "+authHeader)

	createResp, err := c.deploymentClient.CreateDeployment(ctx, createReq)
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

	authHeader := c.opts.APIKey
	if authHeader == "" {
		authHeader = c.opts.AuthToken
	}
	getReq.Header().Set("Authorization", "Bearer "+authHeader)

	getResp, err := c.deploymentClient.GetDeployment(ctx, getReq)
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

			if err := c.processNewSteps(deploymentID, deployment.GetSteps(), processedSteps, currentStatus, onStepUpdate); err != nil {
				return err
			}

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
		stepTimestamp := step.GetCreatedAt()

		if processedSteps[stepTimestamp] {
			continue
		}

		if step.GetErrorMessage() != "" {
			return fmt.Errorf("deployment failed: %s", step.GetErrorMessage())
		}

		if step.GetMessage() != "" {
			event := DeploymentStepEvent{
				DeploymentID: deploymentID,
				Step:         step,
				Status:       currentStatus,
			}
			if err := onStepUpdate(event); err != nil {
				return err
			}

			time.Sleep(800 * time.Millisecond)
		}
		processedSteps[stepTimestamp] = true
	}
	return nil
}

// getFailureMessage extracts failure message from version
func (c *ControlPlaneClient) getFailureMessage(deployment *ctrlv1.Deployment) string {
	if deployment.GetErrorMessage() != "" {
		return deployment.GetErrorMessage()
	}

	for _, step := range deployment.GetSteps() {
		if step.GetErrorMessage() != "" {
			return step.GetErrorMessage()
		}
	}

	return "Unknown deployment error"
}

// handleBuildError provides specific error handling for build failures
func (c *ControlPlaneClient) handleBuildError(err error) error {
	if strings.Contains(err.Error(), "connection refused") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAppErrorsInternalServiceUnavailable),
			fault.Internal(fmt.Sprintf("Failed to connect to control plane at %s", c.opts.ControlPlaneURL)),
			fault.Public("Unable to connect to control plane. Is it running?"),
		)
	}

	if connectErr := new(connect.Error); errors.As(err, &connectErr) {
		switch connectErr.Code() {
		case connect.CodeUnauthenticated:
			authMethod := "API key"
			if c.opts.APIKey == "" {
				authMethod = "auth token"
			}
			return fault.Wrap(err,
				fault.Code(codes.UnkeyAuthErrorsAuthenticationMalformed),
				fault.Internal(fmt.Sprintf("Authentication failed with %s", authMethod)),
				fault.Public(fmt.Sprintf("Authentication failed. Check your %s.", authMethod)),
			)
		case connect.CodeInvalidArgument:
			return fault.Wrap(err,
				fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
				fault.Internal(fmt.Sprintf("Invalid build configuration: %v", connectErr.Message())),
				fault.Public(fmt.Sprintf("Build configuration error: %v", connectErr.Message())),
			)
		case connect.CodeInternal:
			return fault.Wrap(err,
				fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
				fault.Internal(fmt.Sprintf("Build service error: %v", connectErr.Message())),
				fault.Public("Build failed on server. Please try again."),
			)
		}
	}

	return fault.Wrap(err,
		fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
		fault.Internal(fmt.Sprintf("CreateBuild API call failed: %v", err)),
		fault.Public("Failed to create build. Please try again."),
	)
}

// handleCreateDeploymentError provides specific error handling for deployment creation
func (c *ControlPlaneClient) handleCreateDeploymentError(err error) error {
	if strings.Contains(err.Error(), "connection refused") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAppErrorsInternalServiceUnavailable),
			fault.Internal(fmt.Sprintf("Failed to connect to control plane at %s", c.opts.ControlPlaneURL)),
			fault.Public("Unable to connect to control plane. Is it running?"),
		)
	}

	if connectErr := new(connect.Error); errors.As(err, &connectErr) {
		if connectErr.Code() == connect.CodeUnauthenticated {
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

	return fault.Wrap(err,
		fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
		fault.Internal(fmt.Sprintf("CreateDeployment API call failed: %v", err)),
		fault.Public("Failed to create deployment. Please try again."),
	)
}
