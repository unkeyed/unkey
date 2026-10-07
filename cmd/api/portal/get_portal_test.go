package portal

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/sdks/api/go/v3/models/components"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
)

func TestGetPortal(t *testing.T) {
	tests := []struct {
		name string
		args string
		want components.V2PortalGetPortalRequestBodyUnion
	}{
		{
			"by portal",
			"portal get-portal --portal=acme-portal",
			components.CreateV2PortalGetPortalRequestBodyUnionV2PortalGetPortalRequestBody1(components.V2PortalGetPortalRequestBody1{
				Portal:     "acme-portal",
				KeyspaceID: nil,
				AppID:      nil,
			}),
		},
		{
			"by keyspace",
			"portal get-portal --keyspace-id=ks_1234abcd",
			components.CreateV2PortalGetPortalRequestBodyUnionV2PortalGetPortalRequestBody2(components.V2PortalGetPortalRequestBody2{
				Portal:     nil,
				KeyspaceID: "ks_1234abcd",
				AppID:      nil,
			}),
		},
		{
			"by app",
			"portal get-portal --app-id=app_1234abcd",
			components.CreateV2PortalGetPortalRequestBodyUnionV2PortalGetPortalRequestBody3(components.V2PortalGetPortalRequestBody3{
				Portal:     nil,
				KeyspaceID: nil,
				AppID:      "app_1234abcd",
			}),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, testutil.CaptureRequest[components.V2PortalGetPortalRequestBodyUnion](t, Cmd(), tt.args))
		})
	}
}
