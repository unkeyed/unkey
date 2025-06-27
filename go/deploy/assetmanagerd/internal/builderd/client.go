package builderd

import (
	"context"
	"log/slog"
	"time"

	tlspkg "github.com/unkeyed/unkey/go/deploy/pkg/tls"
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
	cfg    *Config
	logger *slog.Logger
}

// NewClient creates a new builderd client
func NewClient(cfg *Config, logger *slog.Logger) (*Client, error) {
	return &Client{
		cfg:    cfg,
		logger: logger.With("component", "builderd-client"),
	}, nil
}

// BuildDockerRootfs triggers a docker rootfs build
func (c *Client) BuildDockerRootfs(ctx context.Context, dockerImage string, labels map[string]string) (string, error) {
	// AIDEV-TODO: Implement actual builderd client method
	return "mock-build-id", nil
}

// WaitForBuild waits for a build to complete
func (c *Client) WaitForBuild(ctx context.Context, buildID string, timeout time.Duration) (*CompletedBuild, error) {
	// AIDEV-TODO: Implement actual builderd client method
	return &CompletedBuild{
		Build: &Build{
			BuildId:    buildID,
			State:      BuildStateCompleted,
			RootfsPath: "/mock/rootfs/path",
		},
	}, nil
}

// BuildDockerRootfsWithOptions triggers a docker rootfs build with options
func (c *Client) BuildDockerRootfsWithOptions(ctx context.Context, dockerImage string, labels map[string]string, tenantID string, customerID string) (string, error) {
	// AIDEV-TODO: Implement actual builderd client method
	return "mock-build-id", nil
}

// WaitForBuildWithTenant waits for a build to complete with tenant context
func (c *Client) WaitForBuildWithTenant(ctx context.Context, buildID string, timeout time.Duration, tenantID string) (*CompletedBuild, error) {
	// AIDEV-TODO: Implement actual builderd client method
	return &CompletedBuild{
		Build: &Build{
			BuildId:    buildID,
			State:      BuildStateCompleted,
			RootfsPath: "/mock/rootfs/path",
		},
	}, nil
}