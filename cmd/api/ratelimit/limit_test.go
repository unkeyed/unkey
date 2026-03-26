package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/util"
	"github.com/unkeyed/unkey/pkg/ptr"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestLimit(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitLimitRequestBody
	}{
		{
			name: "minimal required fields only",
			args: "ratelimit limit --namespace=api.requests --identifier=user_abc123 --limit=100 --duration=60000",
			want: openapi.V2RatelimitLimitRequestBody{
				Namespace:  "api.requests",
				Identifier: "user_abc123",
				Limit:      100,
				Duration:   60000,
				Cost:       ptr.P(int64(1)),
			},
		},
		{
			name: "with optional cost",
			args: "ratelimit limit --namespace=api.heavy_operations --identifier=user_def456 --limit=50 --duration=3600000 --cost=5",
			want: openapi.V2RatelimitLimitRequestBody{
				Namespace:  "api.heavy_operations",
				Identifier: "user_def456",
				Limit:      50,
				Duration:   3600000,
				Cost:       ptr.P(int64(5)),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := util.CaptureRequest[openapi.V2RatelimitLimitRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, req)
		})
	}
}
