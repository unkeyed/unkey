package app

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/deploy/privatenetwork"
	"github.com/unkeyed/unkey/pkg/uid"
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
		{name: "self", want: false},
		{name: "selfish", want: true},
		{name: "my-self", want: true},
		{name: strings.Repeat("a", 64), want: false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, IsValidName(tt.name), "IsValidName(%q)", tt.name)
		})
	}
}

func TestHostVariable(t *testing.T) {
	caller := uid.New(uid.DeploymentPrefix)
	namespace := uid.DNS1035()
	for _, tt := range []struct {
		name    string
		wantKey string
	}{
		{name: "responder", wantKey: "RESPONDER_HOST"},
		{name: "event-store", wantKey: "EVENT_STORE_HOST"},
		{name: "db2", wantKey: "DB2_HOST"},
		{name: "a--b", wantKey: "A__B_HOST"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			key, value := HostVariable(caller, tt.name, namespace)
			require.Equal(t, tt.wantKey, key)
			require.Equal(t, privatenetwork.ConnectionServiceName(caller, tt.name)+"."+namespace+".unkey.internal", value)
		})
	}
}
