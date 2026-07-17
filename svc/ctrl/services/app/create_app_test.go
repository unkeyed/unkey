package app

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

func TestPickDefaultRegion(t *testing.T) {
	region := func(id, name string, canSchedule bool) db.ListRegionsRow {
		return db.ListRegionsRow{ID: id, Name: name, Platform: "aws", CanSchedule: canSchedule}
	}

	t.Run("no regions", func(t *testing.T) {
		id, ok := pickDefaultRegion(nil)
		require.False(t, ok)
		require.Empty(t, id)
	})

	t.Run("no schedulable regions", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_1", "us-east-1", false),
			region("rgn_2", "eu-west-1", false),
		})
		require.False(t, ok)
		require.Empty(t, id)
	})

	t.Run("picks alphabetically-first schedulable region", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_use", "us-east-1", true),
			region("rgn_euw", "eu-west-1", true),
			region("rgn_usw", "us-west-2", true),
		})
		require.True(t, ok)
		require.Equal(t, "rgn_euw", id, "eu-west-1 sorts first")
	})

	t.Run("skips unschedulable regions when picking", func(t *testing.T) {
		id, ok := pickDefaultRegion([]db.ListRegionsRow{
			region("rgn_ap", "ap-south-1", false),
			region("rgn_use", "us-east-1", true),
		})
		require.True(t, ok)
		require.Equal(t, "rgn_use", id, "ap-south-1 sorts first but is not schedulable")
	})
}
