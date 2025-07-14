package deploy

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	ctrlv1 "github.com/unkeyed/unkey/go/gen/proto/ctrl/v1"
	"github.com/unkeyed/unkey/go/gen/proto/ctrl/v1/ctrlv1connect"
	"github.com/unkeyed/unkey/go/pkg/codes"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/git"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

func notifyControlPlane(ctx context.Context, logger logging.Logger, opts *DeployOptions, dockerImage string) error {
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
		EnvironmentId:  "env_prod",
		DockerImageTag: dockerImage,
	})

	// Add auth header
	createReq.Header().Set("Authorization", "Bearer "+opts.AuthToken)

	// Call the API
	createResp, err := client.CreateVersion(ctx, createReq)
	if err != nil {
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

	versionId := createResp.Msg.GetVersionId()
	if versionId != "" {
		fmt.Printf("  Version ID: %s\n", versionId)
	}

	// Poll for version status updates
	if err := pollVersionStatus(ctx, logger, client, opts.AuthToken, versionId); err != nil {
		return fmt.Errorf("deployment failed: %w", err)
	}

	printDeploymentComplete(versionId, opts.WorkspaceID, opts.Branch)
	return nil
}

func printDeploymentComplete(versionId, workspace, branch string) {
	// Use Git info for hostname generation
	gitInfo := git.GetInfo()
	identifier := versionId
	if gitInfo.ShortSHA != "" {
		identifier = gitInfo.ShortSHA
	}

	fmt.Println()
	fmt.Println("Deployment Complete")
	fmt.Printf("  Version ID: %s\n", versionId)
	fmt.Printf("  Status: Ready\n")
	fmt.Printf("  Environment: Production\n")
	fmt.Println()
	fmt.Println("Domains")

	// Replace underscores with dashes for valid hostname format
	cleanIdentifier := strings.ReplaceAll(identifier, "_", "-")
	fmt.Printf("  https://%s-%s-%s.unkey.app\n", branch, cleanIdentifier, workspace)
	fmt.Printf("  https://api.acme.com\n")
}
