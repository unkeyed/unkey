package sentinel

import (
	"fmt"
	"slices"
	"time"

	"github.com/unkeyed/unkey/pkg/assert"
)

type Config struct {
	SentinelID string

	WorkspaceID string

	// EnvironmentID identifies which environment this sentinel serves
	// A single environment may have multiple deployments, and this sentinel
	// handles all of them based on the deployment ID passed in each request
	EnvironmentID string

	Region string

	HttpPort int

	DatabasePrimary         string
	DatabaseReadonlyReplica string

	ClickhouseURL string

	OtelEnabled           bool
	OtelTraceSamplingRate float64
	PrometheusPort        int

	// --- Gossip cluster configuration ---

	// GossipEnabled controls whether gossip-based cache invalidation is active
	GossipEnabled bool

	// GossipBindAddr is the address to bind gossip listeners on (default "0.0.0.0")
	GossipBindAddr string

	// GossipLANPort is the LAN memberlist port (default 7946)
	GossipLANPort int

	// GossipWANPort is the WAN memberlist port for bridges (default 7947)
	GossipWANPort int

	// GossipLANSeeds are addresses of existing LAN cluster members (e.g. k8s headless service DNS)
	GossipLANSeeds []string

	// GossipWANSeeds are addresses of cross-region bridges
	GossipWANSeeds []string

	// --- Logging sampler configuration ---

	// LogSampleRate is the baseline probability (0.0-1.0) of emitting log events.
	LogSampleRate float64

	// LogSlowThreshold defines what duration qualifies as "slow" for sampling.
	LogSlowThreshold time.Duration
}

func (c Config) Validate() error {
	err := assert.All(
		assert.NotEmpty(c.WorkspaceID, "workspace ID is required"),
		assert.NotEmpty(c.EnvironmentID, "environment ID is required"),
	)

	if err != nil {
		return err
	}

	validRegions := []string{
		"local.dev",
		"us-east-1.aws",
		"us-east-2.aws",
		"us-west-1.aws",
		"us-west-2.aws",
		"eu-central-1.aws",
	}

	if !slices.Contains(validRegions, c.Region) {
		return fmt.Errorf("invalid region: %s, must be one of %v", c.Region, validRegions)

	}

	return nil
}
