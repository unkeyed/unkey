package apis

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateAPI(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2ApisCreateApiRequestBody
	}{
		{
			name: "basic",
			args: "apis create-api --name=payment-service-prod",
			want: openapi.V2ApisCreateApiRequestBody{
				Name: "payment-service-prod",
			},
		},
		{
			name: "short name",
			args: "apis create-api --name=dev",
			want: openapi.V2ApisCreateApiRequestBody{
				Name: "dev",
			},
		},
		{
			name: "name with dots and hyphens",
			args: "apis create-api --name=my-org.api-v2",
			want: openapi.V2ApisCreateApiRequestBody{
				Name: "my-org.api-v2",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2ApisCreateApiRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
