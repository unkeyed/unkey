package apis

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestDeleteAPI(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2ApisDeleteApiRequestBody
	}{
		{
			name: "minimal required flags",
			args: "apis delete-api --api-id=api_1234abcd",
			want: openapi.V2ApisDeleteApiRequestBody{
				ApiId: "api_1234abcd",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := testutil.CaptureRequest[openapi.V2ApisDeleteApiRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
