package logdrain

import (
	"errors"
	"time"

	"github.com/unkeyed/unkey/pkg/config"
)

// ClickHouseConfig identifies the source and telemetry ClickHouse cluster.
type ClickHouseConfig struct {
	URL string `toml:"url" config:"required,nonempty"`
}

// Config controls service dependencies, polling, and delivery failure policy.
type Config struct {
	InstanceID   string             `toml:"instance_id"`
	Region       string             `toml:"region" config:"required,nonempty"`
	Database     string             `toml:"database" config:"required,nonempty"`
	ClickHouse   ClickHouseConfig   `toml:"clickhouse"`
	Vault        config.VaultConfig `toml:"vault"`
	PollInterval time.Duration      `toml:"poll_interval" config:"default=60s"`
	// WatermarkLag delays each exported timestamp window to protect against late ClickHouse inserts.
	WatermarkLag   time.Duration `toml:"watermark_lag" config:"default=5m"`
	BatchSize      int           `toml:"batch_size" config:"default=1000,min=1"`
	PauseThreshold int           `toml:"pause_threshold" config:"default=50,min=1"`
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
	if c.Database == "" || c.ClickHouse.URL == "" || c.Vault.URL == "" || c.Vault.Token == "" {
		return errors.New("database, clickhouse.url, vault.url, and vault.token are required")
	}
	if c.PollInterval <= 0 || c.WatermarkLag < 0 || c.BatchSize <= 0 || c.PauseThreshold <= 0 || c.MaxConcurrentDrains <= 0 {
		return errors.New("poll interval, batch size, pause threshold, and max concurrent drains must be positive; watermark lag must not be negative")
	}
	return nil
}
