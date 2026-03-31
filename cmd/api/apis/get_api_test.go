package apis

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestGetAPI(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2ApisGetApiRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "apis get-api --api-id=api_1234abcd",
			want: openapi.V2ApisGetApiRequestBody{
				ApiId: "api_1234abcd",
			},
		},
		{
			name: "different api id",
			args: "apis get-api --api-id=api_xyz789",
			want: openapi.V2ApisGetApiRequestBody{
				ApiId: "api_xyz789",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2ApisGetApiRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
