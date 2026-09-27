package keys

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	keysdb "github.com/unkeyed/unkey/internal/services/keys/db"
	"github.com/unkeyed/unkey/pkg/cache"
	"github.com/unkeyed/unkey/pkg/clock"
	"github.com/unkeyed/unkey/pkg/prometheus/lazy"
)

var metricsTestRegistry = prometheus.NewRegistry()

func TestMain(m *testing.M) {
	lazy.SetRegistry(metricsTestRegistry)
	os.Exit(m.Run())
}

type stubQueries struct {
	keysdb.Querier
	row keysdb.FindKeyForVerificationRow
}

func (s stubQueries) FindKeyForVerification(context.Context, keysdb.DBTX, string) (keysdb.FindKeyForVerificationRow, error) {
	return s.row, nil
}

func counterValue(t *testing.T, name string, labels map[string]string) float64 {
	t.Helper()

	families, err := metricsTestRegistry.Gather()
	require.NoError(t, err)

	for _, family := range families {
		if family.GetName() != name {
			continue
		}
		for _, m := range family.GetMetric() {
			match := true
			for key, want := range labels {
				found := false
				for _, pair := range m.GetLabel() {
					if pair.GetName() == key && pair.GetValue() == want {
						found = true
						break
					}
				}
				if !found {
					match = false
					break
				}
			}
			if match {
				return m.GetCounter().GetValue()
			}
		}
	}

	return 0
}

// TestGet_ExpiredKey_AttributesVerificationToWorkspace guards the tenant
// attribution on unkey_key_verifications_total: the counter must carry the
// owning workspace_id so a single tenant's rejection wave (EXPIRED, DISABLED)
// can be alerted on instead of being diluted across the whole platform.
func TestGet_ExpiredKey_AttributesVerificationToWorkspace(t *testing.T) {
	const workspaceID = "ws_fireworks"

	keyCache, err := cache.New[string, keysdb.CachedKeyData](cache.Config[string, keysdb.CachedKeyData]{
		Fresh:    time.Minute,
		Stale:    time.Hour,
		MaxSize:  100,
		Resource: "test",
		Clock:    clock.New(),
	})
	require.NoError(t, err)
	t.Cleanup(keyCache.Close)

	previous := keysdb.Query
	t.Cleanup(func() { keysdb.Query = previous })
	keysdb.Query = stubQueries{row: keysdb.FindKeyForVerificationRow{
		ID:               "key_123",
		KeyAuthID:        "ks_123",
		WorkspaceID:      workspaceID,
		Enabled:          true,
		WorkspaceEnabled: true,
		Expires:          sql.NullTime{Time: time.Now().Add(-time.Hour), Valid: true},
	}}

	s := &service{
		db:       &keysdb.Database{},
		region:   "test",
		source:   "test",
		keyCache: keyCache,
	}

	kv, err := s.Get(context.Background(), nil, "hash")
	require.NoError(t, err)
	require.NotNil(t, kv)
	require.Equal(t, StatusExpired, kv.Status)

	got := counterValue(t, "unkey_key_verifications_total", map[string]string{
		"type":         "key",
		"code":         string(StatusExpired),
		"workspace_id": workspaceID,
	})
	require.Equal(t, float64(1), got)
}
