package deploy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/unkeyed/unkey/go/cmd/cli/progress"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

func notifyControlPlane(ctx context.Context, logger logging.Logger, opts *DeployOptions, dockerImage string, tracker *progress.Tracker) error {
	// Create control plane client
	httpClient := &http.Client{}
	client := ctrlv1connect.NewVersionServiceClient(httpClient, opts.ControlPlaneURL)

	// Create version request
	createReq := connect.NewRequest(&ctrlv1.CreateVersionRequest{
		WorkspaceId:    opts.WorkspaceID,
		ProjectId:      opts.ProjectID,
		Branch:         opts.Branch,
		SourceType:     ctrlv1.SourceType_SOURCE_TYPE_CLI_UPLOAD,
		GitCommitSha:   opts.Commit,
		EnvironmentId:  "env_prod", // TODO: Make this configurable
		DockerImageTag: dockerImage,
	})

	// Add auth header
	createReq.Header().Set("Authorization", "Bearer "+opts.AuthToken)

	// Call the API
	createResp, err := client.CreateVersion(ctx, createReq)
	if err != nil {
		return handleCreateVersionError(err, opts)
	}

	versionId := createResp.Msg.GetVersionId()
	if versionId == "" {
		return fault.Wrap(
			fmt.Errorf("empty version ID returned from control plane"),
			fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
			fault.Internal("CreateVersion API returned empty version ID"),
			fault.Public("Failed to create version. Please try again."),
		)
	}

	tracker.UpdateStep("deploy", fmt.Sprintf("Version created: %s", versionId))

	// Poll for version status updates with integrated progress tracking
	if err := pollVersionStatus(ctx, logger, client, opts.AuthToken, versionId, tracker); err != nil {
		return err
	}

	tracker.StartStep("complete", "Generating deployment summary")
	gitInfo := git.GetInfo()
	completionInfo := buildCompletionInfo(versionId, opts.WorkspaceID, opts.Branch, gitInfo)

	if completionInfo == "" {
		tracker.FailStep("complete", "failed to generate deployment summary")
		return fault.Wrap(
			fmt.Errorf("empty completion info generated"),
			fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
			fault.Internal("buildCompletionInfo returned empty string"),
			fault.Public("Failed to generate deployment summary"),
		)
	}

	tracker.CompleteStep("complete", completionInfo)

	// Remove this sleep hack - fix it in the tracker instead
	return nil
}

func buildCompletionInfo(versionId, workspace, branch string, gitInfo git.Info) string {
	var parts []string

	// Version ID
	parts = append(parts, fmt.Sprintf("Version: %s", versionId))

	// Status
	parts = append(parts, "Status: Ready")

	// Environment
	parts = append(parts, "Env: Production")

	// Main domain
	identifier := versionId
	if gitInfo.ShortSHA != "" {
		identifier = gitInfo.ShortSHA
	}
	cleanIdentifier := strings.ReplaceAll(identifier, "_", "-")
	domain := fmt.Sprintf("https://%s-%s-%s.unkey.app", branch, cleanIdentifier, workspace)
	parts = append(parts, fmt.Sprintf("URL: %s", domain))

	return strings.Join(parts, " | ")
}

func handleCreateVersionError(err error, opts *DeployOptions) error {
	// Check if it's a connection error
	if strings.Contains(err.Error(), "connection refused") {
		return fault.Wrap(err,
			fault.Code(codes.UnkeyAppErrorsInternalServiceUnavailable),
			fault.Internal(fmt.Sprintf("Failed to connect to control plane at %s", opts.ControlPlaneURL)),
			fault.Public("Unable to connect to control plane. Is it running?"),
		)
	}

	// Check if it's an auth error
	if connectErr := new(connect.Error); errors.As(err, &connectErr) {
		if connectErr.Code() == connect.CodeUnauthenticated {
			return fault.Wrap(err,
				fault.Code(codes.UnkeyAuthErrorsAuthenticationMalformed),
				fault.Internal(fmt.Sprintf("Authentication failed with token: %s", opts.AuthToken)),
				fault.Public("Authentication failed. Check your auth token."),
			)
		}
	}

	// Generic API error
	return fault.Wrap(err,
		fault.Code(codes.UnkeyAppErrorsInternalUnexpectedError),
		fault.Internal(fmt.Sprintf("CreateVersion API call failed: %v", err)),
		fault.Public("Failed to create version. Please try again."),
	)
}

// getFailureMessage extracts failure message from version
func getFailureMessage(version *ctrlv1.Version) string {
	if version.GetErrorMessage() != "" {
		return version.GetErrorMessage()
	}

	// Check for error in steps
	for _, step := range version.GetSteps() {
		if step.GetErrorMessage() != "" {
			return step.GetErrorMessage()
		}
	}

	return "Unknown deployment error"
}
