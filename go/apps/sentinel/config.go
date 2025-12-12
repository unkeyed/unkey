package sentinel

import "github.com/unkeyed/unkey/go/pkg/assert"

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

	OtelEnabled           bool
	OtelTraceSamplingRate float64
	PrometheusPort        int
}

func (c Config) Validate() error {
	return assert.All(
		assert.NotEmpty(c.WorkspaceID, "workspace ID is required"),
		assert.NotEmpty(c.EnvironmentID, "environment ID is required"),
	)
}
