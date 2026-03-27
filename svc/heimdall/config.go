package heimdall

import "github.com/unkeyed/unkey/pkg/config"

type Config struct {
	Region             string               `toml:"region" config:"required,nonempty"`
	Platform           string               `toml:"platform" config:"required,nonempty"`
	ClickHouseURL      string               `toml:"clickhouse_url" config:"required,nonempty"`
	CollectionInterval string               `toml:"collection_interval" config:"default=15s"`
	Observability      config.Observability  `toml:"observability"`
}

func (c *Config) Validate() error {
	return nil
}
