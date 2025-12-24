package sentinel

import (
	"fmt"

	"github.com/unkeyed/unkey/go/pkg/assert"
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

	OtelEnabled           bool
	OtelTraceSamplingRate float64
	PrometheusPort        int
}

func (c Config) Validate() error {
	err := assert.All(
		assert.NotEmpty(c.WorkspaceID, "workspace ID is required"),
		assert.NotEmpty(c.EnvironmentID, "environment ID is required"),
	)

	if err != nil {
		return err
	}

	validRegions := map[string]bool{
		"aws:us-east-1":    true,
		"aws:us-east-2":    true,
		"aws:us-west-1":    true,
		"aws:us-west-2":    true,
		"aws:eu-central-1": true,
	}

	if valid := validRegions[c.Region]; !valid {
		return fmt.Errorf("invalid region: %s, must be one of %v", c.Region, validRegions)
	}

	return nil
}
