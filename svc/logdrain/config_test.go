package logdrain

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/unkeyed/unkey/svc/logdrain/internal/coordinator"
)

func TestConfig_Validate_ReplicasInRange(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		ClickHouseURL: "tcp://localhost:9000",
		Replicas:      1,
	}
	require.NoError(t, cfg.Validate())
}

func TestConfig_Validate_ReplicasAtTotalShardsBoundary(t *testing.T) {
	t.Parallel()

	// Replicas == TotalShards is the densest valid layout (one shard
	// per pod). Anything past that would leave trailing pods with empty
	// ranges; the boundary itself must still pass.
	cfg := &Config{
		ClickHouseURL: "tcp://localhost:9000",
		Replicas:      coordinator.TotalShards,
	}
	require.NoError(t, cfg.Validate())
}

func TestConfig_Validate_TooManyReplicasFails(t *testing.T) {
	t.Parallel()

	// Misconfiguration guard. Replicas > TotalShards means the trailing
	// pods compute an empty [start, end) and silently own zero work.
	// Fail at startup so the operator notices instead of half the fleet
	// being idle.
	cfg := &Config{
		ClickHouseURL: "tcp://localhost:9000",
		Replicas:      coordinator.TotalShards + 1,
	}
	err := cfg.Validate()
	require.Error(t, err)
	require.Contains(t, err.Error(), "replicas")
}
