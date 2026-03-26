package apis

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestListKeys(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2ApisListKeysRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "apis list-keys --api-id=api_123",
			want: openapi.V2ApisListKeysRequestBody{
				ApiId:               "api_123",
				Limit:               ptr.P(100),
				Decrypt:             ptr.P(false),
				RevalidateKeysCache: ptr.P(false),
			},
		},
		{
			name: "with limit",
			args: "apis list-keys --api-id=api_123 --limit=50",
			want: openapi.V2ApisListKeysRequestBody{
				ApiId:               "api_123",
				Limit:               ptr.P(50),
				Decrypt:             ptr.P(false),
				RevalidateKeysCache: ptr.P(false),
			},
		},
		{
			name: "with cursor",
			args: "apis list-keys --api-id=api_123 --cursor=abc_next_page",
			want: openapi.V2ApisListKeysRequestBody{
				ApiId:               "api_123",
				Cursor:              ptr.P("abc_next_page"),
				Limit:               ptr.P(100),
				Decrypt:             ptr.P(false),
				RevalidateKeysCache: ptr.P(false),
			},
		},
		{
			name: "with external id",
			args: "apis list-keys --api-id=api_123 --external-id=user_456",
			want: openapi.V2ApisListKeysRequestBody{
				ApiId:               "api_123",
				ExternalId:          ptr.P("user_456"),
				Limit:               ptr.P(100),
				Decrypt:             ptr.P(false),
				RevalidateKeysCache: ptr.P(false),
			},
		},
		{
			name: "decrypt true",
			args: "apis list-keys --api-id=api_123 --decrypt",
			want: openapi.V2ApisListKeysRequestBody{
				ApiId:               "api_123",
				Limit:               ptr.P(100),
				Decrypt:             ptr.P(true),
				RevalidateKeysCache: ptr.P(false),
			},
		},
		{
			name: "revalidate-keys-cache true",
			args: "apis list-keys --api-id=api_123 --revalidate-keys-cache",
			want: openapi.V2ApisListKeysRequestBody{
				ApiId:               "api_123",
				Limit:               ptr.P(100),
				Decrypt:             ptr.P(false),
				RevalidateKeysCache: ptr.P(true),
			},
		},
		{
			name: "all optional flags",
			args: "apis list-keys --api-id=api_123 --limit=100 --cursor=cur_xyz --external-id=user_789 --decrypt --revalidate-keys-cache",
			want: openapi.V2ApisListKeysRequestBody{
				ApiId:               "api_123",
				Limit:               ptr.P(100),
				Cursor:              ptr.P("cur_xyz"),
				ExternalId:          ptr.P("user_789"),
				Decrypt:             ptr.P(true),
				RevalidateKeysCache: ptr.P(true),
			},
		},
	}

	listResponse := `{"meta":{"requestId":"test"},"data":[]}`

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2ApisListKeysRequestBody](t, Cmd(), tt.args, listResponse)
			require.Equal(t, tt.want, req)
		})
	}
}
