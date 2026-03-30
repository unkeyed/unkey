package heimdall

import (
	"time"

	"github.com/unkeyed/unkey/pkg/config"
)

type Config struct {
	Region             string               `toml:"region" config:"required,nonempty"`
	Platform           string               `toml:"platform" config:"required,nonempty"`
	ClickHouseURL      string               `toml:"clickhouse_url" config:"required,nonempty"`
	CollectionInterval time.Duration        `toml:"collection_interval" config:"default=15s"`
	NodeName           string               `toml:"node_name" config:"required,nonempty"`
	Observability      config.Observability  `toml:"observability"`
}

func (c *Config) Validate() error {
	return nil
}
