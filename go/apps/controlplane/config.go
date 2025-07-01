package controlplane

import (
	"github.com/unkeyed/unkey/go/pkg/clock"
)

type Config struct {
	// InstanceID is the unique identifier for this instance of the controlplane service
	InstanceID string

	// Image specifies the container image identifier including repository and tag
	Image string

	// --- Database configuration ---

	// HydraDatabaseDSN is the primary database connection string for hydra workflows
	HydraDatabaseDSN string

	// --- Business Database Configuration (for quota check workflow) ---

	// UnkeyDatabaseDSN is the business database connection for accessing workspace data
	UnkeyDatabaseDSN string

	// --- External Services Configuration ---

	// ClickHouseURL is the ClickHouse database connection string for analytics
	ClickHouseURL string

	// SlackWebhookURL is the Slack webhook URL for notifications
	SlackWebhookURL string

	// --- OpenTelemetry configuration ---

	// Enable sending otel data to the collector endpoint for metrics, traces, and logs
	OtelEnabled           bool
	OtelTraceSamplingRate float64

	// Clock for time operations (mainly for testing)
	Clock clock.Clock
}

func (c Config) Validate() error {

	return nil

}
