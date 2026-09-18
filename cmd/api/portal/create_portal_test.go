package portal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/pkg/ptr"
)

func TestCreatePortal(t *testing.T) {
	tests := []struct {
		name string
		args string
		want components.V2PortalCreatePortalRequestBodyUnion
	}{
		{
			name: "keyspace with defaults",
			args: "portal create-portal --slug=acme-portal --display-name=Acme --keyspace-id=ks_1234abcd",
			want: components.CreateV2PortalCreatePortalRequestBodyUnionV2PortalCreatePortalRequestBody1(components.V2PortalCreatePortalRequestBody1{
				Slug:         "acme-portal",
				DisplayName:  "Acme",
				KeyspaceID:   "ks_1234abcd",
				AppID:        nil,
				Enabled:      ptr.P(true),
				LogoURL:      nil,
				PrimaryColor: nil,
			}),
		},
		{
			name: "app with options",
			args: "portal create-portal --slug=developer-portal --display-name='Developer Portal' --app-id=app_1234abcd --enabled=false --logo-url=https://cdn.example.com/logo.svg --primary-color=#6366f1",
			want: components.CreateV2PortalCreatePortalRequestBodyUnionV2PortalCreatePortalRequestBody2(components.V2PortalCreatePortalRequestBody2{
				Slug:         "developer-portal",
				DisplayName:  "Developer Portal",
				KeyspaceID:   nil,
				AppID:        "app_1234abcd",
				Enabled:      ptr.P(false),
				LogoURL:      ptr.P("https://cdn.example.com/logo.svg"),
				PrimaryColor: ptr.P("#6366f1"),
			}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[components.V2PortalCreatePortalRequestBodyUnion](t, Cmd(), tt.args))
		})
	}
}
