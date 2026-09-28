package privatenetwork

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestHostVariable(t *testing.T) {
	for _, tt := range []struct {
		name      string
		wantKey   string
		wantValue string
	}{
		{name: "responder", wantKey: "RESPONDER_HOST", wantValue: "responder.unkey.internal"},
		{name: "event-store", wantKey: "EVENT_STORE_HOST", wantValue: "event-store.unkey.internal"},
		{name: "db2", wantKey: "DB2_HOST", wantValue: "db2.unkey.internal"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key, value := HostVariable(tt.name)
			require.Equal(t, tt.wantKey, key)
			require.Equal(t, tt.wantValue, value)
			require.NotContains(t, value, ":", "hostname must not carry a port or scheme")
		})
	}
}

// TestReplicaHost guarantees that a deployment's own replicas resolve under
// its app slug only when that slug is a valid DNS label, so an app with an
// unusable slug gets no replica hostname instead of an unresolvable one.
func TestReplicaHost(t *testing.T) {
	for _, tt := range []struct {
		slug     string
		wantHost string
		wantOK   bool
	}{
		{slug: "api", wantHost: "api.unkey.internal", wantOK: true},
		{slug: "event-store", wantHost: "event-store.unkey.internal", wantOK: true},
		{slug: "Api", wantOK: false},
		{slug: "under_score", wantOK: false},
		{slug: "", wantOK: false},
		{slug: strings.Repeat("a", 64), wantOK: false},
	} {
		t.Run(tt.slug, func(t *testing.T) {
			host, ok := ReplicaHost(tt.slug)
			require.Equal(t, tt.wantOK, ok, "ReplicaHost(%q) ok", tt.slug)
			require.Equal(t, tt.wantHost, host, "ReplicaHost(%q) host", tt.slug)
		})
	}
}
