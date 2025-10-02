package clickhouse

import (
	"github.com/unkeyed/unkey/go/pkg/analytics"
	"github.com/unkeyed/unkey/go/pkg/clickhouse"
	"github.com/unkeyed/unkey/go/pkg/fault"
	"github.com/unkeyed/unkey/go/pkg/otel/logging"
)

// Config contains ClickHouse-specific configuration for analytics.
type Config struct {
	// URL is the ClickHouse connection string
	URL string `json:"url"`
}

// New creates a new ClickHouse analytics writer from configuration.
func New(config Config, logger logging.Logger) (analytics.Writer, error) {
	if config.URL == "" {
		return nil, fault.New("clickhouse URL is required")
	}

	chConfig := clickhouse.Config{
		URL:    config.URL,
		Logger: logger,
	}

	client, err := clickhouse.New(chConfig)
	if err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to create clickhouse client"))
	}

	return newWriter(client), nil
}
