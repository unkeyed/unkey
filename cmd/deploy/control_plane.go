package deploy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/sdks/api/go/v2/models/operations"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/fault"
	"github.com/unkeyed/unkey/pkg/git"
	"github.com/unkeyed/unkey/pkg/otel/logging"
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

// handleSDKError converts SDK errors to appropriate fault types
func (c *ControlPlaneClient) handleSDKError(err error, operation string) error {
	errMsg := err.Error()

	// SDK errors contain HTTP status information
	if strings.Contains(errMsg, "401") || strings.Contains(errMsg, "Unauthorized") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAuthErrorsAuthenticationMalformed),
			fault.Internal(fmt.Sprintf("%s failed: invalid or missing root key", operation)),
			fault.Public("Authentication failed. Check your UNKEY_ROOT_KEY."),
		)
	}

	if strings.Contains(errMsg, "403") || strings.Contains(errMsg, "Forbidden") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAuthErrorsAuthorizationInsufficientPermissions),
			fault.Internal(fmt.Sprintf("%s failed: insufficient permissions", operation)),
			fault.Public("Permission denied. Root key must have project.*.create_deployment, project.*.generate_upload_url, and project.*.read_deployment permissions."),
		)
	}

	if strings.Contains(errMsg, "404") || strings.Contains(errMsg, "Not Found") {
		return fault.Wrap(err,
			fault.Code(codes.Data.Project.NotFound.URN()),
			fault.Internal(fmt.Sprintf("%s failed: resource not found", operation)),
			fault.Public("The requested resource does not exist."),
		)
	}

	return fault.Wrap(err,
		fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
		fault.Internal(fmt.Sprintf("%s failed: %v", operation, err)),
		fault.Public("An unexpected error occurred. Please try again."),
	)
}

// UploadBuildContext uploads the build context to S3 and returns the context key
func (c *ControlPlaneClient) UploadBuildContext(ctx context.Context, contextPath string) (string, error) {
	// Call SDK method with proper parameters
	res, err := c.sdk.Internal.GenerateUploadURL(ctx, components.V2DeployGenerateUploadURLRequestBody{
		ProjectID: c.opts.ProjectID,
	})
	if err != nil {
		return "", c.handleSDKError(err, "generate upload URL")
	}

	// Extract response data
	if res.V2DeployGenerateUploadURLResponseBody == nil {
		return "", fmt.Errorf("empty response from generate upload URL")
	}

	uploadURL := res.V2DeployGenerateUploadURLResponseBody.Data.UploadURL
	buildContextPath := res.V2DeployGenerateUploadURLResponseBody.Data.Context

	if uploadURL == "" || buildContextPath == "" {
		return "", fmt.Errorf("empty upload URL or context key returned")
	}

	// Create and upload tar (existing logic remains unchanged)
	tarPath, err := createContextTar(contextPath)
	if err != nil {
		return "", fmt.Errorf("failed to create tar archive: %w", err)
	}
	defer func() { _ = os.Remove(tarPath) }()

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
	defer func() { _ = file.Close() }()

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return fmt.Errorf("upload failed with status %d (failed to read response body: %w)", resp.StatusCode, readErr)
		}
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
	if err = tmpFile.Close(); err != nil {
		return "", fmt.Errorf("failed to close temp file: %w", err)
	}
	tarPath := tmpFile.Name()

	if err = os.Chmod(tarPath, 0o666); err != nil {
		_ = os.Remove(tarPath)
		return "", fmt.Errorf("failed to set file permissions: %w", err)
	}

	cmd := exec.Command("tar", "-czf", tarPath, "-C", absContextPath, ".")
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(tarPath)
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

	// Build request using SDK operations types
	var reqBody operations.DeployCreateDeploymentRequest

	if buildContextPath != "" {
		// Build source
		buildSource := operations.V2DeployBuildSource{
			ProjectID:       c.opts.ProjectID,
			Branch:          c.opts.Branch,
			EnvironmentSlug: c.opts.Environment,
			Build: operations.Build{
				Context:    buildContextPath,
				Dockerfile: &dockerfilePath,
			},
		}

		// Add optional keyspace ID
		if c.opts.KeyspaceID != "" {
			buildSource.KeyspaceID = &c.opts.KeyspaceID
		}

		// Add git commit info
		if commitInfo.CommitSHA != "" {
			buildSource.GitCommit = &components.V2DeployGitCommit{
				CommitSha:       &commitInfo.CommitSHA,
				CommitMessage:   &commitInfo.Message,
				AuthorHandle:    &commitInfo.AuthorHandle,
				AuthorAvatarURL: &commitInfo.AuthorAvatarURL,
				Timestamp:       &commitInfo.CommitTimestamp,
			}
		}

		reqBody = operations.CreateDeployCreateDeploymentRequestV2DeployBuildSource(buildSource)
	} else if dockerImage != "" {
		// Image source
		imageSource := operations.V2DeployImageSource{
			ProjectID:       c.opts.ProjectID,
			Branch:          c.opts.Branch,
			EnvironmentSlug: c.opts.Environment,
			Image:           dockerImage,
		}

		// Add optional keyspace ID
		if c.opts.KeyspaceID != "" {
			imageSource.KeyspaceID = &c.opts.KeyspaceID
		}

		// Add git commit info
		if commitInfo.CommitSHA != "" {
			imageSource.GitCommit = &components.V2DeployGitCommit{
				CommitSha:       &commitInfo.CommitSHA,
				CommitMessage:   &commitInfo.Message,
				AuthorHandle:    &commitInfo.AuthorHandle,
				AuthorAvatarURL: &commitInfo.AuthorAvatarURL,
				Timestamp:       &commitInfo.CommitTimestamp,
			}
		}

		reqBody = operations.CreateDeployCreateDeploymentRequestV2DeployImageSource(imageSource)
	} else {
		return "", fmt.Errorf("either buildContextPath or dockerImage must be provided")
	}

	res, err := c.sdk.Internal.CreateDeployment(ctx, reqBody)
	if err != nil {
		return "", c.handleCreateDeploymentError(c.handleSDKError(err, "create deployment"))
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
func (c *ControlPlaneClient) GetDeployment(ctx context.Context, deploymentID string) (*ctrlv1.Deployment, error) {
	// Call SDK method
	res, err := c.sdk.Internal.GetDeployment(ctx, components.V2DeployGetDeploymentRequestBody{
		DeploymentID: deploymentID,
	})
	if err != nil {
		return nil, c.handleSDKError(err, "get deployment")
	}

	if res.V2DeployGetDeploymentResponseBody == nil {
		return nil, fmt.Errorf("empty response from get deployment")
	}

	data := res.V2DeployGetDeploymentResponseBody.Data

	// Convert SDK response to ctrlv1.Deployment for compatibility with existing code
	deployment := &ctrlv1.Deployment{
		Id:           data.ID,
		Status:       stringToDeploymentStatus(string(data.Status)),
		ErrorMessage: "",
		Hostnames:    []string{},
		Steps:        []*ctrlv1.DeploymentStep{},
	}

	if data.ErrorMessage != nil {
		deployment.ErrorMessage = *data.ErrorMessage
	}

	if len(data.Hostnames) > 0 {
		deployment.Hostnames = data.Hostnames
	}

	if len(data.Steps) > 0 {
		for _, step := range data.Steps {
			protoStep := &ctrlv1.DeploymentStep{}
			if step.Status != nil {
				protoStep.Status = *step.Status
			}
			if step.Message != nil {
				protoStep.Message = *step.Message
			}
			if step.ErrorMessage != nil {
				protoStep.ErrorMessage = *step.ErrorMessage
			}
			if step.CreatedAt != nil {
				protoStep.CreatedAt = *step.CreatedAt
			}
			deployment.Steps = append(deployment.Steps, protoStep)
		}
	}

	return deployment, nil
}

func stringToDeploymentStatus(status string) ctrlv1.DeploymentStatus {
	switch status {
	case "PENDING":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_PENDING
	case "BUILDING":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_BUILDING
	case "DEPLOYING":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_DEPLOYING
	case "NETWORK":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_NETWORK
	case "READY":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_READY
	case "FAILED":
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_FAILED
	default:
		return ctrlv1.DeploymentStatus_DEPLOYMENT_STATUS_UNSPECIFIED
	}
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
	errMsg := err.Error()

	// Check for common error patterns
	if strings.Contains(errMsg, "authentication failed") || strings.Contains(errMsg, "401") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAuthErrorsAuthenticationMalformed),
			fault.Internal("Authentication failed with root key"),
			fault.Public("Authentication failed. Check your UNKEY_ROOT_KEY."),
		)
	}

	if strings.Contains(errMsg, "permission denied") || strings.Contains(errMsg, "403") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAuthErrorsAuthorizationInsufficientPermissions),
			fault.Internal("Insufficient permissions"),
			fault.Public("Permission denied. Ensure your root key has project.*.create_deployment permission."),
		)
	}

	return fault.Wrap(err,
		fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
		fault.Internal(fmt.Sprintf("CreateDeployment API call failed: %v", err)),
		fault.Public("Failed to create deployment. Please try again."),
	)
}
