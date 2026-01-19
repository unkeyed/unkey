package deploy

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	unkey "github.com/unkeyed/sdks/api/go/v2"
	"github.com/unkeyed/sdks/api/go/v2/models/components"
	"github.com/unkeyed/sdks/api/go/v2/models/operations"
	ctrlv1 "github.com/unkeyed/unkey/gen/proto/ctrl/v1"
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

// UploadBuildContext uploads the build context to S3 and returns the context key
func (c *ControlPlaneClient) UploadBuildContext(ctx context.Context, contextPath string) (string, error) {
	res, err := c.sdk.Internal.GenerateUploadURL(ctx, components.V2DeployGenerateUploadURLRequestBody{
		ProjectID: c.opts.ProjectID,
	})
	if err != nil {
		return "", err
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

	// Create and upload tar
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

// buildCommonFields extracts common deployment fields
func (c *ControlPlaneClient) buildCommonFields() (projectID, branch, env string, keyspaceID *string, gitCommit *components.V2DeployGitCommit) {
	commitInfo := git.GetInfo()

	projectID = c.opts.ProjectID
	branch = c.opts.Branch
	env = c.opts.Environment

	if c.opts.KeyspaceID != "" {
		keyspaceID = &c.opts.KeyspaceID
	}

	if commitInfo.CommitSHA != "" {
		gitCommit = &components.V2DeployGitCommit{
			CommitSha:       &commitInfo.CommitSHA,
			CommitMessage:   &commitInfo.Message,
			AuthorHandle:    &commitInfo.AuthorHandle,
			AuthorAvatarURL: &commitInfo.AuthorAvatarURL,
			Timestamp:       &commitInfo.CommitTimestamp,
		}
	}

	return
}

// CreateDeployment creates a new deployment in the control plane
// Pass either buildContextPath (for build from source) or dockerImage (for prebuilt image), not both
func (c *ControlPlaneClient) CreateDeployment(ctx context.Context, buildContextPath, dockerImage string) (string, error) {
	projectID, branch, env, keyspaceID, gitCommit := c.buildCommonFields()

	var reqBody operations.DeployCreateDeploymentRequest

	if buildContextPath != "" {
		dockerfilePath := c.opts.Dockerfile
		if dockerfilePath == "" {
			dockerfilePath = "Dockerfile"
		}
		reqBody = operations.CreateDeployCreateDeploymentRequestV2DeployBuildSource(
			operations.V2DeployBuildSource{
				ProjectID:       projectID,
				Branch:          branch,
				EnvironmentSlug: env,
				KeyspaceID:      keyspaceID,
				GitCommit:       gitCommit,
				Build: operations.Build{
					Context:    buildContextPath,
					Dockerfile: &dockerfilePath,
				},
			},
		)
	} else if dockerImage != "" {
		reqBody = operations.CreateDeployCreateDeploymentRequestV2DeployImageSource(
			operations.V2DeployImageSource{
				ProjectID:       projectID,
				Branch:          branch,
				EnvironmentSlug: env,
				KeyspaceID:      keyspaceID,
				GitCommit:       gitCommit,
				Image:           dockerImage,
			},
		)
	} else {
		return "", fmt.Errorf("either buildContextPath or dockerImage must be provided")
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
func (c *ControlPlaneClient) GetDeployment(ctx context.Context, deploymentID string) (*ctrlv1.Deployment, error) {
	res, err := c.sdk.Internal.GetDeployment(ctx, components.V2DeployGetDeploymentRequestBody{
		DeploymentID: deploymentID,
	})
	if err != nil {
		return nil, err
	}

	if res.V2DeployGetDeploymentResponseBody == nil {
		return nil, fmt.Errorf("empty response from get deployment")
	}

	return convertSDKDeploymentToProto(res.V2DeployGetDeploymentResponseBody.Data), nil
}

// convertSDKDeploymentToProto converts SDK deployment data to protobuf format
func convertSDKDeploymentToProto(data components.V2DeployGetDeploymentResponseData) *ctrlv1.Deployment {
	deployment := &ctrlv1.Deployment{
		Id:           data.ID,
		Status:       stringToDeploymentStatus(string(data.Status)),
		ErrorMessage: ptr.SafeDeref(data.ErrorMessage),
		Hostnames:    data.Hostnames,
		Steps:        make([]*ctrlv1.DeploymentStep, 0, len(data.Steps)),
	}

	for _, step := range data.Steps {
		deployment.Steps = append(deployment.Steps, &ctrlv1.DeploymentStep{
			Status:       ptr.SafeDeref(step.Status),
			Message:      ptr.SafeDeref(step.Message),
			ErrorMessage: ptr.SafeDeref(step.ErrorMessage),
			CreatedAt:    ptr.SafeDeref(step.CreatedAt),
		})
	}

	return deployment
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
