package gateway

import (
	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
	"testing"
)

func TestListPolicies(t *testing.T) {
	tests := []struct {
		name, args string
		want       openapi.V2GatewayListPoliciesRequestBody
	}{{"minimal", "gateway list-policies --project=p --app=a --environment=e", openapi.V2GatewayListPoliciesRequestBody{Project: "p", App: "a", Environment: "e"}}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.Equal(t, tt.want, util.CaptureRequestWithResponse[openapi.V2GatewayListPoliciesRequestBody](t, Cmd(), tt.args, `{"meta":{"requestId":"test"},"data":[]}`))
		})
	}
}
