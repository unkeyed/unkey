package privatenetwork

import (
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
