package krane

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
)

type Backend string

const (
	Docker     Backend = "docker"
	Kubernetes Backend = "kubernetes"
)

type Config struct {
	// InstanceID is the unique identifier for this instance of the API server
	InstanceID string

	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the API server to listen on (default: 7070)
	// Used in production deployments. Ignored if Listener is provided.
	HttpPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// Backend specifies the container manager, either "docker" or "kubernetes"
	Backend Backend

	// DockerSocketPath specifies the Docker socket URL including protocol (e.g., unix:///var/run/docker.sock)
	// Only used when Backend is Docker. Default should be set via CLI flags.
	DockerSocketPath string

	// Enable sending otel data to the  collector endpoint for metrics, traces, and logs
	OtelEnabled           bool
	OtelTraceSamplingRate float64

	Clock clock.Clock
}

func (c Config) Validate() error {

	return nil
}
