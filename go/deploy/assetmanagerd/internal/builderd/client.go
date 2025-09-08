package builderd

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
	builderv1 "github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1"
	"github.com/unkeyed/unkey/go/gen/proto/deploy/builderd/v1/builderdv1connect"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// Config holds the configuration for the builderd client
type Config struct {
	Endpoint    string
	Timeout     time.Duration
	MaxRetries  int
	RetryDelay  time.Duration
	TLSProvider tlspkg.Provider
}

// BuildState represents the state of a build
type BuildState int

const (
	BuildStatePending BuildState = iota
	BuildStateRunning
	BuildStateCompleted
	BuildStateFailed
)

func (s BuildState) String() string {
	switch s {
	case BuildStatePending:
		return "pending"
	case BuildStateRunning:
		return "running"
	case BuildStateCompleted:
		return "completed"
	case BuildStateFailed:
		return "failed"
	default:
		return "unknown"
	}
}

// Build represents a build
type Build struct {
	BuildId    string
	State      BuildState
	RootfsPath string
}

// CompletedBuild represents a completed build
type CompletedBuild struct {
	Build *Build
}

// Client is a client for the builderd service
type Client struct {
	cfg           *Config
	logger        *slog.Logger
	builderClient builderdv1connect.BuilderServiceClient
}

// NewClient creates a new builderd client
func NewClient(cfg *Config, logger *slog.Logger) (*Client, error) {
	// Get HTTP client with TLS configuration
	httpClient := cfg.TLSProvider.HTTPClient()

	// Wrap with OpenTelemetry instrumentation for trace propagation
	httpClient.Transport = otelhttp.NewTransport(httpClient.Transport)

	// Create Connect client
	builderClient := builderdv1connect.NewBuilderServiceClient(
		httpClient,
		cfg.Endpoint,
	)

	logger.Info("initialized builderd client",
		slog.String("endpoint", cfg.Endpoint),
	)

	return &Client{
		cfg:           cfg,
		logger:        logger.With("component", "builderd-client"),
		builderClient: builderClient,
	}, nil
}

// BuildDockerRootfs triggers a docker rootfs build with options
func (c *Client) BuildDockerRootfs(ctx context.Context, dockerImage string, labels map[string]string) (string, error) {
	// AIDEV-NOTE: Implemented builderd client method for automatic builds
	c.logger.InfoContext(ctx, "triggering docker rootfs build",
		slog.String("docker_image", dockerImage))

	// Create build request
	req := &builderv1.CreateBuildRequest{
		Config: &builderv1.BuildConfig{
			Source: &builderv1.BuildSource{
				SourceType: &builderv1.BuildSource_DockerImage{
					DockerImage: &builderv1.DockerImageSource{
						ImageUri: dockerImage,
					},
				},
			},
			Target: &builderv1.BuildTarget{
				TargetType: &builderv1.BuildTarget_MicrovmRootfs{
					MicrovmRootfs: &builderv1.MicroVMRootfs{
						InitStrategy: builderv1.InitStrategy_INIT_STRATEGY_TINI,
					},
				},
			},
			Strategy: &builderv1.BuildStrategy{
				StrategyType: &builderv1.BuildStrategy_DockerExtract{
					DockerExtract: &builderv1.DockerExtractStrategy{
						FlattenFilesystem: true,
					},
				},
			},
			Labels: labels,
		},
	}

	// Make the request with timeout context
	ctxWithTimeout, cancel := context.WithTimeout(ctx, c.cfg.Timeout)
	defer cancel()

	connectReq := connect.NewRequest(req)

	resp, err := c.builderClient.CreateBuild(ctxWithTimeout, connectReq)
	if err != nil {
		c.logger.ErrorContext(ctx, "failed to create build",
			slog.String("docker_image", dockerImage),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("failed to create build: %w", err)
	}

	buildID := resp.Msg.GetBuildId()
	c.logger.InfoContext(ctx, "build created successfully",
		slog.String("build_id", buildID),
		slog.String("docker_image", dockerImage),
		slog.String("state", resp.Msg.GetState().String()),
	)

	return buildID, nil
}

// WaitForBuild waits for a build to complete
func (c *Client) WaitForBuild(ctx context.Context, buildID string, timeout time.Duration) (*CompletedBuild, error) {
	c.logger.InfoContext(ctx, "waiting for build to complete",
		slog.String("build_id", buildID),
		slog.Duration("timeout", timeout),
	)

	// Create context with timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// Poll for completion
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctxWithTimeout.Done():
			return nil, fmt.Errorf("timeout waiting for build %s to complete: %w", buildID, ctx.Err())
		case <-ticker.C:
			// Check build status
			req := &builderv1.GetBuildRequest{
				BuildId: buildID,
			}

			connectReq := connect.NewRequest(req)
			resp, err := c.builderClient.GetBuild(ctxWithTimeout, connectReq)
			if err != nil {
				c.logger.WarnContext(ctx, "failed to get build status",
					slog.String("build_id", buildID),
					slog.String("error", err.Error()),
				)
				continue
			}

			build := resp.Msg.GetBuild()
			c.logger.DebugContext(ctx, "build status update",
				slog.String("build_id", buildID),
				slog.String("state", build.GetState().String()),
			)

			switch build.GetState() {
			case builderv1.BuildState_BUILD_STATE_COMPLETED:
				c.logger.InfoContext(ctx, "build completed successfully",
					slog.String("build_id", buildID),
					slog.String("rootfs_path", build.GetRootfsPath()),
				)
				return &CompletedBuild{
					Build: &Build{
						BuildId:    buildID,
						State:      BuildStateCompleted,
						RootfsPath: build.GetRootfsPath(),
					},
				}, nil
			case builderv1.BuildState_BUILD_STATE_FAILED:
				c.logger.ErrorContext(ctx, "build failed",
					slog.String("build_id", buildID),
					slog.String("error", build.GetErrorMessage()),
				)
				return nil, fmt.Errorf("build %s failed: %s", buildID, build.GetErrorMessage())
			case builderv1.BuildState_BUILD_STATE_CANCELLED:
				c.logger.WarnContext(ctx, "build was cancelled",
					slog.String("build_id", buildID),
				)
				return nil, fmt.Errorf("build %s was cancelled", buildID)
			default:
				// Build still in progress, continue polling
				continue
			}
		}
	}
}
