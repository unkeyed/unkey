package privatenetwork

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

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
