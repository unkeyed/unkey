package ratelimit

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/cmd/api/internal/testutil"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

func TestCreateNamespace(t *testing.T) {
	tests := []struct {
		name string
		args string
		want openapi.V2RatelimitCreateNamespaceRequestBody
	}{
		{
			name: "create namespace",
			args: "ratelimit create-namespace --name=email.outbound",
			want: openapi.V2RatelimitCreateNamespaceRequestBody{
				Name: "email.outbound",
			},
		},
		{
			name: "preserve case and spaces",
			args: `ratelimit create-namespace --name="Email Outbound"`,
			want: openapi.V2RatelimitCreateNamespaceRequestBody{
				Name: "Email Outbound",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := testutil.CaptureRequest[openapi.V2RatelimitCreateNamespaceRequestBody](t, Cmd(), tt.args)
			require.Equal(t, tt.want, got)
		})
	}
}
