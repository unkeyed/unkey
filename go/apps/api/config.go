package api

import (
	"github.com/unkeyed/unkey/go/pkg/assert"
	"github.com/unkeyed/unkey/go/pkg/clock"
)

type Config struct {
	// Platform identifies the cloud platform where the node is running (e.g., aws, gcp, hetzner)
	Platform string

	// Image specifies the container image identifier including repository and tag
	Image string

	// HttpPort defines the HTTP port for the API server to listen on (default: 7070)
	HttpPort int

	// Region identifies the geographic region where this node is deployed
	Region string

	// --- Cluster configuration ---

	ClusterEnabled bool

	// ClusterNodeID is the unique identifier for this node within the cluster
	ClusterNodeID string

	// --- Advertise Address configuration ---

	// ClusterAdvertiseAddrStatic is a static IP address or hostname for node discovery
	ClusterAdvertiseAddrStatic string

	// ClusterAdvertiseAddrAwsEcsMetadata enables automatic address discovery using AWS ECS container metadata
	ClusterAdvertiseAddrAwsEcsMetadata bool

	// ClusterRpcPort is the port used for internal RPC communication between nodes (default: 7071)
	ClusterRpcPort int

	// ClusterGossipPort is the port used for cluster membership and failure detection (default: 7072)
	ClusterGossipPort int

	// --- Discovery configuration ---

	// ClusterDiscoveryStaticAddrs lists seed node addresses for static cluster configuration
	ClusterDiscoveryStaticAddrs []string

	// ClusterDiscoveryRedisURL provides a Redis connection string for dynamic cluster discovery
	ClusterDiscoveryRedisURL string

	// --- Logs configuration ---

	// LogsColor enables ANSI color codes in log output
	LogsColor bool

	// --- ClickHouse configuration ---

	// ClickhouseURL is the ClickHouse database connection string
	ClickhouseURL string

	// --- Database configuration ---

	// DatabasePrimary is the primary database connection string for read and write operations
	DatabasePrimary string

	// DatabaseReadonlyReplica is an optional read-replica database connection string for read operations
	DatabaseReadonlyReplica string

	// --- OpenTelemetry configuration ---

	// OtelOtlpEndpoint specifies the OpenTelemetry collector endpoint for metrics, traces, and logs
	OtelEnabled bool

	Clock clock.Clock
}

func (c Config) Validate() error {

	if c.ClusterEnabled {
		err := assert.Multi(
			assert.NotEmpty(c.ClusterNodeID, "node id must not be empty"),
			assert.Greater(c.ClusterRpcPort, 0),
			assert.Greater(c.ClusterGossipPort, 0),
			assert.True(c.ClusterAdvertiseAddrStatic != "" || c.ClusterAdvertiseAddrAwsEcsMetadata),
		)
		if err != nil {
			return err
		}
	}

	return nil
}
