package deploy

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/git"
	"github.com/unkeyed/unkey/pkg/otel/logging"
	"github.com/unkeyed/unkey/pkg/ptr"
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

// UploadBuildContext uploads the build context to S3 and returns the context key
func (c *ControlPlaneClient) UploadBuildContext(ctx context.Context, contextPath string) (string, error) {
	uploadReq := connect.NewRequest(&ctrlv1.GenerateUploadURLRequest{
		UnkeyProjectId: c.opts.ProjectID,
	})

	authHeader := c.opts.APIKey
	if authHeader == "" {
		authHeader = c.opts.AuthToken
	}
	uploadReq.Header().Set("Authorization", "Bearer "+authHeader)

	uploadResp, err := c.buildClient.GenerateUploadURL(ctx, uploadReq)
	if err != nil {
		return "", fmt.Errorf("failed to generate upload URL: %w", err)
	}

	uploadURL := uploadResp.Msg.GetUploadUrl()
	buildContextPath := uploadResp.Msg.GetBuildContextPath()

	if uploadURL == "" || buildContextPath == "" {
		return "", fmt.Errorf("empty upload URL or context key returned")
	}

	tarPath, err := createContextTar(contextPath)
	if err != nil {
		return "", fmt.Errorf("failed to create tar archive: %w", err)
	}
	defer os.Remove(tarPath)

	if err := uploadToPresignedURL(ctx, uploadURL, tarPath); err != nil {
		return "", fmt.Errorf("failed to upload build context: %w", err)
	}

	return buildContextPath, nil
}

// uploadToPresignedURL uploads a file to a presigned S3 URL
func uploadToPresignedURL(ctx context.Context, presignedURL, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	stat, err := file.Stat()
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, presignedURL, file)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.ContentLength = stat.Size()
	req.Header.Set("Content-Type", "application/gzip")

	client := &http.Client{
		Timeout: 5 * time.Minute,
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("upload request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

// createContextTar creates a tar.gz from the given directory path
// and returns the absolute path to the created tar file
func createContextTar(contextPath string) (string, error) {
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

	sharedDir := "/tmp/ctrl"
	if err = os.MkdirAll(sharedDir, 0o777); err != nil {
		return "", fmt.Errorf("failed to create shared dir: %w", err)
	}

	tmpFile, err := os.CreateTemp(sharedDir, "build-context-*.tar.gz")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpFile.Close()
	tarPath := tmpFile.Name()

	if err = os.Chmod(tarPath, 0o666); err != nil {
		os.Remove(tarPath)
		return "", fmt.Errorf("failed to set file permissions: %w", err)
	}

	cmd := exec.Command("tar", "-czf", tarPath, "-C", absContextPath, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(tarPath)
		return "", fmt.Errorf("tar command failed: %w\nOutput: %s", err, string(output))
	}

	return tarPath, nil
}

// CreateDeployment creates a new deployment in the control plane
// Pass either buildContextPath (for build from source) or dockerImage (for prebuilt image), not both
func (c *ControlPlaneClient) CreateDeployment(ctx context.Context, buildContextPath, dockerImage string) (string, error) {
	commitInfo := git.GetInfo()

	dockerfilePath := c.opts.Dockerfile
	if dockerfilePath == "" {
		dockerfilePath = "Dockerfile"
	}

	req := &ctrlv1.CreateDeploymentRequest{
		Source:          nil,
		ProjectId:       c.opts.ProjectID,
		KeyspaceId:      &c.opts.KeyspaceID,
		Branch:          c.opts.Branch,
		EnvironmentSlug: c.opts.Environment,
		GitCommit: &ctrlv1.GitCommitInfo{
			CommitSha:       commitInfo.CommitSHA,
			CommitMessage:   commitInfo.Message,
			AuthorHandle:    commitInfo.AuthorHandle,
			AuthorAvatarUrl: commitInfo.AuthorAvatarURL,
			Timestamp:       commitInfo.CommitTimestamp,
		},
	}

	if buildContextPath != "" {
		req.Source = &ctrlv1.CreateDeploymentRequest_BuildContext{
			BuildContext: &ctrlv1.BuildContext{
				BuildContextPath: buildContextPath,
				DockerfilePath:   ptr.P(dockerfilePath),
			},
		}
	} else if dockerImage != "" {
		req.Source = &ctrlv1.CreateDeploymentRequest_DockerImage{
			DockerImage: dockerImage,
		}
	} else {
		return "", fmt.Errorf("either buildContextPath or dockerImage must be provided")
	}

	createReq := connect.NewRequest(req)

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
func (c *ControlPlaneClient) GetDeployment(ctx context.Context, deploymentID string) (*ctrlv1.Deployment, error) {
	getReq := connect.NewRequest(&ctrlv1.GetDeploymentRequest{
		DeploymentId: deploymentID,
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
) error {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeout := time.NewTimer(300 * time.Second)
	defer timeout.Stop()

	// Track processed steps by creation time to avoid duplicates
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
			if currentStatus == ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY {
				return nil
			}
		}
	}
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
