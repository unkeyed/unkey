package logdrain

//go:generate go run github.com/unkeyed/unkey/tools/configdocs -service logdrain -type Config -out ../../docs/engineering/architecture/services/logdrain/configuration.mdx

import (
	"errors"
	"time"

	"github.com/unkeyed/unkey/pkg/config"
)

// ClickHouseConfig identifies the source and telemetry ClickHouse cluster.
type ClickHouseConfig struct {
	// URL is the ClickHouse DSN. The service reads exported streams and
	// writes delivery telemetry over this one connection.
	URL string `toml:"url" config:"required,nonempty"`
}

// Config controls service dependencies, polling, and delivery failure policy.
type Config struct {
	// InstanceID identifies this process in logs and traces. Leave it empty
	// to generate a unique ID at startup.
	InstanceID string `toml:"instance_id"`
	// Region labels telemetry and SQL comments. It does not affect routing:
	// any replica can lease any drain.
	Region string `toml:"region" config:"required,nonempty"`
	// Database is the MySQL DSN. MySQL stores drain configuration, delivery
	// state, and leases.
	Database string `toml:"database" config:"required,nonempty"`
	// ClickHouse hosts the exported streams and the delivery telemetry.
	ClickHouse ClickHouseConfig `toml:"clickhouse"`
	// Vault decrypts destination credentials. URL and token are required,
	// enforced by Validate because the shared struct leaves them optional.
	Vault config.VaultConfig `toml:"vault"`
	// PollInterval controls how often each replica scans for due drains. It
	// is also the pause between delivery cycles of a caught-up drain, so it
	// bounds the steady-state delivery latency.
	PollInterval time.Duration `toml:"poll_interval" config:"default=60s"`
	// WatermarkLag delays each exported timestamp window to protect against late ClickHouse inserts.
	WatermarkLag time.Duration `toml:"watermark_lag" config:"default=5m"`
	// BatchSize caps the events read and shipped per delivery attempt.
	BatchSize int `toml:"batch_size" config:"default=1000,min=1"`
	// PauseThreshold is the number of consecutive failures that moves a
	// drain to paused_by_failure.
	PauseThreshold int `toml:"pause_threshold" config:"default=50,min=1"`
	// MaxConcurrentDrains caps how many drains one replica processes in
	// parallel within a poll cycle.
	MaxConcurrentDrains int `toml:"max_concurrent_drains" config:"default=4,min=1"`
	// Observability configures logging, tracing, and the metrics listener.
	// The metrics listener is the only HTTP server this service has; it
	// serves /metrics and the /health probe endpoints on the same port.
	Observability config.Observability `toml:"observability"`

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
