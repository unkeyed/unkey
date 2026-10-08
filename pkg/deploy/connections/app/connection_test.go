package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestIsValidName(t *testing.T) {
	for _, tt := range []struct {
		name string
		want bool
	}{
		{name: "responder", want: true},
		{name: "event-store", want: true},
		{name: "db2", want: true},
		{name: "a", want: true},
		{name: "a--b", want: true},
		{name: "unke", want: true},
		{name: "", want: false},
		{name: "Payments", want: false},
		{name: "-payments", want: false},
		{name: "payments-", want: false},
		{name: "2db", want: false},
		{name: "with_underscore", want: false},
		{name: "api.service", want: false},
		{name: "api service", want: false},
		{name: "unkey", want: false},
		{name: "unkey-api", want: false},
		{name: "unkeyed", want: false},
		{name: strings.Repeat("a", 64), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsValidName(tt.name), "IsValidName(%q)", tt.name)
		})
	}
}

func TestHostVariable(t *testing.T) {
	for _, tt := range []struct {
		name      string
		wantKey   string
		wantValue string
	}{
		{name: "responder", wantKey: "RESPONDER_HOST", wantValue: "responder.unkey.internal"},
		{name: "event-store", wantKey: "EVENT_STORE_HOST", wantValue: "event-store.unkey.internal"},
		{name: "db2", wantKey: "DB2_HOST", wantValue: "db2.unkey.internal"},
		{name: "a--b", wantKey: "A__B_HOST", wantValue: "a--b.unkey.internal"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key, value := HostVariable(tt.name)
			require.Equal(t, tt.wantKey, key)
			require.Equal(t, tt.wantValue, value)
			require.NotContains(t, value, ":", "hostname must not carry a port or scheme")
		})
	}
}
