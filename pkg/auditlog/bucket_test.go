package auditlog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

// TestResolveBucket guarantees that writers can only stamp events with a
// bucket some reader knows about. An unknown bucket would be written but
// never shown anywhere, silently losing the audit trail.
func TestResolveBucket(t *testing.T) {
	tests := []struct {
		name    string
		bucket  string
		want    string
		wantErr bool
	}{
		{name: "empty resolves to unkey_mutations", bucket: "", want: BucketUnkeyMutations},
		{name: "unkey_mutations passes through", bucket: BucketUnkeyMutations, want: BucketUnkeyMutations},
		{name: "unkey_backoffice passes through", bucket: BucketBackoffice, want: BucketBackoffice},
		{name: "unknown bucket is rejected", bucket: "audit_xyz", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ResolveBucket(tt.bucket)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

// TestDashboardBucketsExcludeBackoffice guarantees the allowlist used by the
// dashboard and log drains never includes the back office bucket, since every
// workspace member can read those two surfaces.
func TestDashboardBucketsExcludeBackoffice(t *testing.T) {
	require.NotContains(t, DashboardBuckets, BucketBackoffice)
	for _, bucket := range DashboardBuckets {
		require.True(t, IsKnownBucket(bucket))
	}
}
