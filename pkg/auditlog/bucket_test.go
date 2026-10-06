package auditlog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsKnownBucket(t *testing.T) {
	require.True(t, IsKnownBucket(BucketUnkeyMutations))
	require.True(t, IsKnownBucket(BucketBackoffice))
	require.False(t, IsKnownBucket(""))
	require.False(t, IsKnownBucket("audit_xyz"))
}

func TestDashboardBucketsExcludeBackoffice(t *testing.T) {
	require.NotContains(t, DashboardBuckets, BucketBackoffice)
	for _, bucket := range DashboardBuckets {
		require.True(t, IsKnownBucket(bucket))
	}
}
