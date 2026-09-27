package portal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/sdks/api/go/v3/optionalnullable"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestUpdatePortal(t *testing.T) {
	tests := []struct {
		name string
		args string
		want components.V2PortalUpdatePortalRequestBody
	}{
		{
			"portal only omits updates",
			"portal update-portal --portal=acme-portal",
			components.V2PortalUpdatePortalRequestBody{
				Portal:       "acme-portal",
				Slug:         nil,
				DisplayName:  nil,
				KeyspaceID:   nil,
				AppID:        nil,
				Enabled:      nil,
				LogoURL:      nil,
				PrimaryColor: nil,
			},
		},
		{
			"all updates",
			"portal update-portal --portal=acme-portal --slug=developer-portal --display-name='Developer Portal' --keyspace-id=ks_1234abcd --enabled=false --logo-url=https://cdn.example.com/logo.svg --primary-color=#6366f1",
			components.V2PortalUpdatePortalRequestBody{
				Portal:       "acme-portal",
				Slug:         new("developer-portal"),
				DisplayName:  new("Developer Portal"),
				KeyspaceID:   new("ks_1234abcd"),
				AppID:        nil,
				Enabled:      new(false),
				LogoURL:      optionalnullable.From(new("https://cdn.example.com/logo.svg")),
				PrimaryColor: optionalnullable.From(new("#6366f1")),
			},
		},
		{
			"clear branding and repoint app",
			"portal update-portal --portal=acme-portal --app-id=app_1234abcd --logo-url=null --primary-color=null",
			components.V2PortalUpdatePortalRequestBody{
				Portal:       "acme-portal",
				Slug:         nil,
				DisplayName:  nil,
				KeyspaceID:   nil,
				AppID:        new("app_1234abcd"),
				Enabled:      nil,
				LogoURL:      optionalnullable.From[string](nil),
				PrimaryColor: optionalnullable.From[string](nil),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[components.V2PortalUpdatePortalRequestBody](t, Cmd(), tt.args))
		})
	}
}
