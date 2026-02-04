package sentinel

import (
	"fmt"
	"slices"

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

	// --- Wide configuration ---

	// WideSuccessSampleRate is the sampling rate for successful requests (0.0 - 1.0).
	// Errors, 5xx responses, and slow requests are always logged.
	// Default is 0.01 (1%).
	WideSuccessSampleRate float64

	// WideSlowThresholdMs is the threshold in milliseconds above which a request
	// is considered slow and always logged. Default is 500.
	WideSlowThresholdMs int
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
