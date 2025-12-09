package credentials

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	cliv1 "github.com/depot/depot-go/proto/depot/cli/v1"
	"github.com/depot/depot-go/proto/depot/cli/v1/cliv1connect"

	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

const (
	depotRegistry = "registry.depot.dev"
	depotAPIURL   = "https://api.depot.dev"
)

// Depot fetches on-demand pull tokens from the Depot API for each image.
type Depot struct {
	logger      logging.Logger
	buildClient cliv1connect.BuildServiceClient
}

type DepotConfig struct {
	Logger logging.Logger
	Token  string
}

// NewDepot creates a Depot registry that fetches on-demand pull tokens.
func NewDepot(cfg *DepotConfig) *Depot {
	return &Depot{
		logger: cfg.Logger,
		buildClient: cliv1connect.NewBuildServiceClient(
			http.DefaultClient,
			depotAPIURL,
			connect.WithInterceptors(connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
				return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
					req.Header().Set("Authorization", "Bearer "+cfg.Token)
					return next(ctx, req)
				}
			})),
		),
	}
}

func (d *Depot) Matches(image string) bool {
	return strings.HasPrefix(image, depotRegistry)
}

func (d *Depot) GetCredentials(ctx context.Context, image, buildID string) (*DockerConfigJSON, error) {
	projectID, err := extractDepotProjectID(image)
	if err != nil {
		return nil, err
	}

	token, err := d.getPullToken(ctx, projectID, buildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull token for project %s: %w", projectID, err)
	}

	// Depot uses "x-token" as username and the pull token as password
	return NewDockerConfig(depotRegistry, "x-token", token), nil
}

func (d *Depot) getPullToken(ctx context.Context, projectID, buildID string) (string, error) {
	reqMsg := &cliv1.GetPullTokenRequest{
		ProjectId: &projectID,
	}
	if buildID != "" {
		reqMsg.BuildId = &buildID
	}

	res, err := d.buildClient.GetPullToken(ctx, connect.NewRequest(reqMsg))
	if err != nil {
		return "", fmt.Errorf("GetPullToken failed: %w", err)
	}

	return res.Msg.GetToken(), nil
}

// extractDepotProjectID extracts the project ID from a Depot image reference.
// Format: registry.depot.dev/<project_id>:<tag> or registry.depot.dev/<project_id>/<name>:<tag>
func extractDepotProjectID(image string) (string, error) {
	if !strings.HasPrefix(image, depotRegistry) {
		return "", fmt.Errorf("not a depot image: %s", image)
	}

	path := strings.TrimPrefix(image, depotRegistry+"/")

	// Remove tag if present
	if idx := strings.LastIndex(path, ":"); idx != -1 {
		path = path[:idx]
	}

	// Remove digest if present
	if idx := strings.LastIndex(path, "@"); idx != -1 {
		path = path[:idx]
	}

	// The project ID is the first path segment
	parts := strings.SplitN(path, "/", 2)
	if len(parts) == 0 || parts[0] == "" {
		return "", fmt.Errorf("could not extract project ID from image: %s", image)
	}

	return parts[0], nil
}
