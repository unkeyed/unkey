package readiness

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRequiredRunningInstances(t *testing.T) {
	require.Equal(t, uint32(1), RequiredRunningInstances(0))
	require.Equal(t, uint32(1), RequiredRunningInstances(1))
	require.Equal(t, uint32(4), RequiredRunningInstances(4))
}

func TestHealthyRegions(t *testing.T) {
	t.Run("a region with no running instance is never healthy", func(t *testing.T) {
		require.Equal(t, 0, HealthyRegions(
			map[string]uint32{},
			map[string]uint32{"us-east-1": 0, "eu-central-1": 1},
		))
	})

	t.Run("a region counts only once it runs its declared minimum", func(t *testing.T) {
		require.Equal(t, 1, HealthyRegions(
			map[string]uint32{"eu-central-1": 3},
			map[string]uint32{"us-east-1": 0, "eu-central-1": 3},
		))
	})

	t.Run("every running region with a satisfied minimum counts", func(t *testing.T) {
		require.Equal(t, 2, HealthyRegions(
			map[string]uint32{"us-east-1": 1, "eu-central-1": 2},
			map[string]uint32{"us-east-1": 1, "eu-central-1": 2},
		))
	})

	t.Run("instances in undeclared regions do not count", func(t *testing.T) {
		require.Equal(t, 0, HealthyRegions(
			map[string]uint32{"ap-southeast-1": 5},
			map[string]uint32{"us-east-1": 1},
		))
	})
}
