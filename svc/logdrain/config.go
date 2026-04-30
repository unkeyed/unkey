package logdrain

import (
	"fmt"
	"time"

	"github.com/unkeyed/unkey/pkg/config"
)

// Config is the runtime configuration for the logdrain service.
//
// shard_count and shard_index are reserved for the v2 multi-replica path.
// v1 runs as a single replica with shard_count=1 and shard_index=0.
type Config struct {
	// ClickHouseURL is the DSN for ClickHouse Cloud. logdrain reads from
	// runtime_logs_raw_v1 and sentinel_requests_raw_v1 in the default
	// database; the user requires SELECT on both plus the materialized
	// attributes_text column.
	ClickHouseURL string `toml:"clickhouse_url" config:"required,nonempty"`

	// Database holds MySQL connection strings for log_drains and friends.
	Database config.DatabaseConfig `toml:"database"`

	// Vault is the connection to svc/vault. Required: paste-token drains
	// store ciphertext that this process must decrypt before the first
	// delivery, and OAuth grant access tokens are encrypted with the same
	// keys.
	Vault config.VaultConfig `toml:"vault"`

	// PollInterval is how often the coordinator looks for new groups and
	// kicks off CH queries for groups whose cursor is older than the last
	// successful flush. The first lag floor (~30-60s) comes from this plus
	// the typical CH ingestion lag from Vector.
	PollInterval time.Duration `toml:"poll_interval" config:"default=10s"`

	// BatchWindow caps how far ahead of the cursor a single CH query
	// fetches. Bigger windows amortise query cost but increase the worst-
	// case end-to-end latency on idle groups.
	BatchWindow time.Duration `toml:"batch_window" config:"default=60s"`

	// MaxBatchRecords caps the number of rows pulled from CH per group per
	// poll. Provider sinks chunk further to honour their own batch size limits.
	MaxBatchRecords int `toml:"max_batch_records" config:"default=5000,min=100"`

	// PauseAfterFailures is the consecutive-failure threshold after which a
	// drain auto-pauses. paused_reason is set on log_drain_state and the
	// dashboard surfaces the verbatim provider error. Operators resume
	// from the dashboard after fixing the underlying problem.
	PauseAfterFailures int `toml:"pause_after_failures" config:"default=50,min=1"`

	// MaxGroupsPerShard prevents query fan-out explosion by limiting the
	// number of ClickHouse queries this shard will execute concurrently.
	MaxGroupsPerShard int `toml:"max_groups_per_shard" config:"default=500,min=10"`

	// MaxDrainsPerWorkspace prevents customers from creating unlimited
	// drains that would overwhelm the group limits.
	MaxDrainsPerWorkspace int `toml:"max_drains_per_workspace" config:"default=100,min=1"`

	// CredentialCacheTTL controls how long decrypted credentials stay in
	// memory. Shorter TTL reduces credential exposure window but increases
	// Vault load. 0 disables TTL (cache forever until restart).
	CredentialCacheTTL time.Duration `toml:"credential_cache_ttl" config:"default=1h"`

	// ShardCount is the total number of logdrain replicas claiming groups
	// via cityHash64(workspace_id) % shard_count. v1 = 1.
	ShardCount int `toml:"shard_count" config:"default=2,min=1"`

	// ShardIndex identifies this replica within ShardCount. Must be
	// 0 <= ShardIndex < ShardCount.
	ShardIndex int `toml:"shard_index" config:"default=0,min=0"`

	Observability config.Observability `toml:"observability"`
}

// Validate enforces invariants the struct tag validators cannot express.
func (c *Config) Validate() error {
	if c.ShardIndex >= c.ShardCount {
		return fmt.Errorf(
			"shard_index (%d) must be < shard_count (%d) — otherwise this replica would never claim a group",
			c.ShardIndex, c.ShardCount,
		)
	}
	return nil
}
