package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// listResponse returns an array-shaped data envelope so the SDK can unmarshal
// the response without error (list endpoints expect "data":[] not "data":{}).
const listResponse = `{"meta":{"requestId":"test"},"data":[]}`

func TestListOverrides(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitListOverridesRequestBody
	}{
		{
			name: "namespace only",
			args: "ratelimit list-overrides --namespace=api.requests",
			want: openapi.V2RatelimitListOverridesRequestBody{
				Namespace: "api.requests",
				Limit:     ptr.P(10),
			},
		},
		{
			name: "with limit",
			args: "ratelimit list-overrides --namespace=api.requests --limit=50",
			want: openapi.V2RatelimitListOverridesRequestBody{
				Namespace: "api.requests",
				Limit:     ptr.P(50),
			},
		},
		{
			name: "with cursor",
			args: "ratelimit list-overrides --namespace=api.requests --cursor=cursor_eyJsYXN0SWQiOiJvdnJfM2RITGNOeVN6SnppRHlwMkpla2E5ciJ9",
			want: openapi.V2RatelimitListOverridesRequestBody{
				Namespace: "api.requests",
				Cursor:    ptr.P("cursor_eyJsYXN0SWQiOiJvdnJfM2RITGNOeVN6SnppRHlwMkpla2E5ciJ9"),
				Limit:     ptr.P(10),
			},
		},
		{
			name: "with limit and cursor",
			args: "ratelimit list-overrides --namespace=billing --limit=100 --cursor=next_page_token",
			want: openapi.V2RatelimitListOverridesRequestBody{
				Namespace: "billing",
				Limit:     ptr.P(100),
				Cursor:    ptr.P("next_page_token"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequestWithResponse[openapi.V2RatelimitListOverridesRequestBody](t, Cmd(), tt.args, listResponse)
			require.Equal(t, tt.want, req)
		})
	}
}
