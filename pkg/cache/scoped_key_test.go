package cache_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/uid"
)

// TestScopedKeyRoundTrip guarantees that a scoped key survives conversion to
// and from its string form, including resource keys that contain the
// separator, so two workspaces never share a cache entry.
func TestScopedKeyRoundTrip(t *testing.T) {
	workspaceID := uid.New(uid.WorkspacePrefix)
	tests := []struct {
		name string
		key  string
	}{
		{name: "name", key: "api-calls"},
		{name: "key containing the separator", key: "region:eu:api-calls"},
		{name: "empty key", key: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := cache.ScopedKey{WorkspaceID: workspaceID, Key: tt.key}

			s := cache.ScopedKeyToString(key)
			require.Equal(t, workspaceID+":"+tt.key, s)

			parsed, err := cache.ScopedKeyFromString(s)
			require.NoError(t, err)
			require.Equal(t, key, parsed)
		})
	}
}

func TestParseScopedKeyRejectsMissingSeparator(t *testing.T) {
	_, err := cache.ParseScopedKey(uid.New(uid.WorkspacePrefix))
	require.Error(t, err)
}
