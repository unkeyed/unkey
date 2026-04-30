package logdrain

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConfig_Validate_ShardIndexInRange(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ClickHouseURL: "tcp://localhost:9000",
		ShardCount:    1,
		ShardIndex:    0,
	}
	require.NoError(t, cfg.Validate())
}

func TestConfig_Validate_ShardIndexOutOfRangeFails(t *testing.T) {
	t.Parallel()

	// Misconfiguration guard. shard_index >= shard_count means the replica
	// would never claim any group: cityHash64(workspace) % shard_count is
	// strictly less than shard_count. Fail at startup, not silently.
	cfg := &Config{
		ClickHouseURL: "tcp://localhost:9000",
		ShardCount:    2,
		ShardIndex:    2,
	}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "shard_index")
}
