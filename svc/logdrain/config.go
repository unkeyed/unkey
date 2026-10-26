package logdrain

import (
	"errors"

	"github.com/unkeyed/unkey/pkg/config"
)

// ClickHouseConfig identifies the source and telemetry ClickHouse cluster.
type ClickHouseConfig struct {
	URL string `toml:"url" config:"required,nonempty"`
}

// VaultConfig adds per-RPC token file authentication for logdrain only.
type VaultConfig struct {
	config.VaultConfig
	TokenFile string `toml:"token_file"`
}

// Config controls service dependencies and delivery failure policy.
type Config struct {
	InstanceID     string           `toml:"instance_id"`
	Region         string           `toml:"region" config:"required,nonempty"`
	Database       string           `toml:"database" config:"required,nonempty"`
	ClickHouse     ClickHouseConfig `toml:"clickhouse"`
	Vault          VaultConfig      `toml:"vault"`
	PauseThreshold int              `toml:"pause_threshold" config:"default=50,min=1"`
	// MaxConcurrentDrains caps how many drains one replica processes in
	// parallel within a poll cycle.
	MaxConcurrentDrains int                  `toml:"max_concurrent_drains" config:"default=4,min=1"`
	Observability       config.Observability `toml:"observability"`

	// InsecureAllowPrivateEndpoints disables the HTTP sink SSRF guard so
	// drains can target loopback or private addresses. Development only.
	InsecureAllowPrivateEndpoints bool `toml:"insecure_allow_private_endpoints"`
}

// Validate returns an error naming any missing or invalid setting.
func (c *Config) Validate() error {
	if c.Database == "" || c.ClickHouse.URL == "" || c.Vault.URL == "" {
		return errors.New("database, clickhouse.url, and vault.url are required")
	}
	if (c.Vault.Token == "") == (c.Vault.TokenFile == "") {
		return errors.New("must set exactly one of vault.token or vault.token_file")
	}
	if c.PauseThreshold <= 0 || c.MaxConcurrentDrains <= 0 {
		return errors.New("pause threshold and max concurrent drains must be positive")
	}
	return nil
}
